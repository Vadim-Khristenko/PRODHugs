package matrix

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go-service-template/internal/models"
	"go-service-template/internal/notify"
	"go-service-template/internal/notify/richtext"

	"github.com/google/uuid"
)

// botUserRepo is the slice of the user repository the Matrix bot needs to
// link accounts and resolve senders for commands and reactions.
type botUserRepo interface {
	IsMatrixIDTaken(ctx context.Context, matrixID string, excludeUserID uuid.UUID) (bool, error)
	SetMatrixLink(ctx context.Context, userID uuid.UUID, matrixID, roomID string) error
	GetByMatrixID(ctx context.Context, matrixID string) (*models.User, error)
}

// hugService is the slice of the hug service the Matrix bot consumes for
// reaction-driven accept/decline and the /me, /stats, /daily commands.
type hugService interface {
	AcceptHug(ctx context.Context, hugID, receiverID uuid.UUID) (*models.Hug, error)
	DeclineHug(ctx context.Context, hugID, receiverID uuid.UUID) error
	GetUserStats(ctx context.Context, userID uuid.UUID, gender *string) (*models.UserStats, error)
	GetHugHistory(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]*models.HugFeedItem, error)
	GetHugActivity(ctx context.Context) ([]*models.HugActivityItem, error)
	ClaimDailyReward(ctx context.Context, userID uuid.UUID) (amount, streakDays, newBalance int32, alreadyClaimed bool, err error)
}

// matrixLoginService handles the auth/registration logic for Matrix bot-login.
type matrixLoginService interface {
	LoginViaMatrix(ctx context.Context, matrixID, roomID string) (*models.User, error)
}

// refStore resolves a Matrix event id back to the domain object it notified
// about, so a reaction on a hug-suggestion message can act on that hug.
type refStore interface {
	GetRefByMessage(ctx context.Context, provider, messageRef string) (string, uuid.UUID, bool, error)
}

// Bot is a long-running Matrix client that syncs, auto-joins direct-message
// invites, and consumes account-link commands ("link <token>") users send to
// it. It reuses the Provider's homeserver/token/http for sends and joins, but
// keeps its own long-poll HTTP client for /sync.
type Bot struct {
	provider   *Provider
	userRepo   botUserRepo
	hugSvc     hugService
	refs       refStore
	linkStore  *LinkStore
	loginStore *LoginStore
	loginSvc   matrixLoginService
	logger     *slog.Logger
	syncHTTP   *http.Client
}

// SetLoginStore configures the login store and service for Matrix bot-login.
// Called after construction to break circular dependencies.
func (b *Bot) SetLoginStore(store *LoginStore, svc matrixLoginService) {
	b.loginStore = store
	b.loginSvc = svc
}

func NewBot(provider *Provider, userRepo botUserRepo, hugSvc hugService, refs refStore, linkStore *LinkStore, logger *slog.Logger) *Bot {
	if logger == nil {
		logger = slog.Default()
	}
	return &Bot{
		provider:  provider,
		userRepo:  userRepo,
		hugSvc:    hugSvc,
		refs:      refs,
		linkStore: linkStore,
		logger:    logger,
		// Long-poll client: its timeout must exceed the /sync timeout below.
		syncHTTP: &http.Client{Timeout: 60 * time.Second},
	}
}

// relatesTo captures an event's m.relates_to, used to read m.reaction
// annotations (rel_type=m.annotation) back off the sync timeline.
type relatesTo struct {
	RelType string `json:"rel_type"`
	EventID string `json:"event_id"`
	Key     string `json:"key"`
}

// syncResponse is the minimal slice of the Matrix /sync response we consume.
type syncResponse struct {
	NextBatch string `json:"next_batch"`
	Rooms     struct {
		Invite map[string]json.RawMessage `json:"invite"`
		Join   map[string]struct {
			Timeline struct {
				Events []struct {
					Type    string `json:"type"`
					Sender  string `json:"sender"`
					Content struct {
						MsgType   string     `json:"msgtype"`
						Body      string     `json:"body"`
						RelatesTo *relatesTo `json:"m.relates_to"`
					} `json:"content"`
				} `json:"events"`
			} `json:"timeline"`
		} `json:"join"`
	} `json:"rooms"`
}

// Run starts the sync loop and blocks until ctx is cancelled. No-op if the
// Matrix provider is not configured.
func (b *Bot) Run(ctx context.Context) {
	if !b.provider.Enabled() {
		b.logger.Info("matrix bot disabled (not configured)")
		return
	}
	b.logger.Info("matrix bot started (sync loop)")

	// Initial sync with timeout=0 gets a fresh since token without replaying
	// history, so we never act on messages sent before startup.
	since, err := b.doSync(ctx, "", 0)
	next := ""
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		b.logger.Warn("matrix bot: initial sync failed, starting from live", "error", err)
	} else {
		next = since.NextBatch
	}

	for {
		if ctx.Err() != nil {
			b.logger.Info("matrix bot stopped")
			return
		}
		resp, err := b.doSync(ctx, next, 30000)
		if err != nil {
			if ctx.Err() != nil {
				b.logger.Info("matrix bot stopped")
				return
			}
			b.logger.Warn("matrix bot: sync error", "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
			continue
		}
		b.process(ctx, resp)
		next = resp.NextBatch
	}
}

// doSync performs one GET /sync call.
func (b *Bot) doSync(ctx context.Context, since string, timeoutMs int) (*syncResponse, error) {
	q := url.Values{}
	q.Set("timeout", strconv.Itoa(timeoutMs))
	if since != "" {
		q.Set("since", since)
	}
	endpoint := b.provider.homeserver + "/_matrix/client/v3/sync?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+b.provider.token)
	resp, err := b.syncHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("matrix sync status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var sr syncResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, err
	}
	return &sr, nil
}

// process handles invites (auto-join) and joined-room messages from a sync.
func (b *Bot) process(ctx context.Context, sr *syncResponse) {
	for roomID := range sr.Rooms.Invite {
		if err := b.joinRoom(ctx, roomID); err != nil {
			b.logger.Warn("matrix bot: auto-join failed", "room", roomID, "error", err)
		}
	}
	for roomID, room := range sr.Rooms.Join {
		for _, ev := range room.Timeline.Events {
			// Ignore our own events (replies/reactions echo back in sync).
			if ev.Sender == b.provider.userID {
				continue
			}
			switch ev.Type {
			case "m.reaction":
				b.handleReaction(ctx, roomID, ev.Sender, ev.Content.RelatesTo)
			case "m.room.message":
				if ev.Content.MsgType != "m.text" {
					continue
				}
				b.handleMessage(ctx, roomID, ev.Sender, ev.Content.Body)
			}
		}
	}
}

// handleReaction acts on a user's reaction to a bot message. A reaction that
// maps to a hug verb and targets a hug-suggestion notification accepts or
// declines that hug on the reacting (linked) user's behalf.
func (b *Bot) handleReaction(ctx context.Context, roomID, sender string, rel *relatesTo) {
	if rel == nil || rel.RelType != "m.annotation" || rel.EventID == "" {
		return
	}
	verb, ok := verbForReaction(rel.Key)
	if !ok {
		return
	}
	kind, hugID, found, err := b.refs.GetRefByMessage(ctx, "matrix", rel.EventID)
	if err != nil || !found || kind != "hug_suggestion" {
		return
	}
	user, err := b.userRepo.GetByMatrixID(ctx, sender)
	if err != nil {
		return // not linked; ignore
	}
	switch verb {
	case "accept":
		if _, err := b.hugSvc.AcceptHug(ctx, hugID, user.ID); err != nil {
			b.logger.Warn("matrix: accept via reaction failed", "error", err)
			return
		}
		b.reply(ctx, roomID, "🤗 Обнимашка принята!")
	case "decline":
		if err := b.hugSvc.DeclineHug(ctx, hugID, user.ID); err != nil {
			b.logger.Warn("matrix: decline via reaction failed", "error", err)
			return
		}
		b.reply(ctx, roomID, "Обнимашка отклонена.")
	}
}

// joinRoom accepts an invite.
func (b *Bot) joinRoom(ctx context.Context, roomID string) error {
	endpoint := b.provider.homeserver + "/_matrix/client/v3/join/" + url.PathEscape(roomID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader("{}"))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+b.provider.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.provider.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("join status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

// parseLinkCommand extracts the token from a "link <token>" message body.
// The keyword is matched case-insensitively; anything else returns ok=false.
func parseLinkCommand(body string) (token string, ok bool) {
	fields := strings.Fields(body)
	if len(fields) != 2 || !strings.EqualFold(fields[0], "link") {
		return "", false
	}
	return fields[1], true
}

// handleMessage is the "/"-prefix command router. Familiar with Telegram, the
// bot uses the same slash commands; non-command messages are ignored.
func (b *Bot) handleMessage(ctx context.Context, roomID, sender, body string) {
	text := strings.TrimSpace(body)
	if !strings.HasPrefix(text, "/") {
		return
	}
	fields := strings.Fields(text)
	cmd := strings.ToLower(fields[0])
	switch cmd {
	case "/start", "/help":
		b.handleHelp(ctx, roomID)
	case "/link":
		b.handleLink(ctx, roomID, sender, fields)
	case "/login":
		b.handleLoginCmd(ctx, roomID, sender, fields)
	case "/me":
		b.handleMe(ctx, roomID, sender)
	case "/stats":
		b.handleStats(ctx, roomID)
	case "/daily":
		b.handleDaily(ctx, roomID, sender)
	}
}

// handleLink consumes a "/link <token>" command and links the sender's Matrix
// account to the user the token belongs to.
func (b *Bot) handleLink(ctx context.Context, roomID, sender string, fields []string) {
	if len(fields) != 2 {
		b.reply(ctx, roomID, "Использование: /link <токен>. Токен можно получить в настройках на сайте.")
		return
	}
	token := fields[1]
	userID, ok := b.linkStore.ConsumeToken(token)
	if !ok {
		b.reply(ctx, roomID, "Ссылка недействительна или истекла. Сгенерируйте новую команду в настройках приложения.")
		return
	}
	taken, err := b.userRepo.IsMatrixIDTaken(ctx, sender, userID)
	if err != nil {
		b.logger.Error("matrix bot: check matrix id taken failed", "sender", sender, "error", err)
		b.reply(ctx, roomID, "Произошла ошибка, попробуйте позже.")
		return
	}
	if taken {
		b.reply(ctx, roomID, "Этот Matrix-аккаунт уже привязан к другому пользователю.")
		return
	}
	if err := b.userRepo.SetMatrixLink(ctx, userID, sender, roomID); err != nil {
		b.logger.Error("matrix bot: set matrix link failed", "user_id", userID, "error", err)
		b.reply(ctx, roomID, "Не удалось привязать аккаунт, попробуйте позже.")
		return
	}
	b.logger.Info("matrix bot: account linked", "user_id", userID, "matrix_id", sender)
	b.reply(ctx, roomID, "✅ Аккаунт привязан! Уведомления об обнимашках будут приходить сюда.")
}

// handleLoginCmd consumes a "/login <botToken>" command and authenticates (or
// auto-registers) the sender for the pending web login session.
func (b *Bot) handleLoginCmd(ctx context.Context, roomID, sender string, fields []string) {
	if b.loginStore == nil || b.loginSvc == nil {
		b.reply(ctx, roomID, "Вход через Matrix временно недоступен.")
		return
	}
	if len(fields) != 2 {
		b.reply(ctx, roomID, "Использование: /login <токен>. Токен показывается на странице входа.")
		return
	}
	botToken := fields[1]
	pollToken, ok := b.loginStore.ConsumeBotToken(botToken)
	if !ok {
		b.reply(ctx, roomID, "Ссылка для входа недействительна или истекла.")
		return
	}
	user, err := b.loginSvc.LoginViaMatrix(ctx, sender, roomID)
	if err != nil {
		b.logger.Error("matrix bot: login failed", "matrix_id", sender, "error", err)
		b.loginStore.FailSession(pollToken, err.Error())
		b.reply(ctx, roomID, "Не удалось войти, попробуйте позже.")
		return
	}
	b.loginStore.AuthenticateSession(pollToken, user.ID)
	b.logger.Info("matrix bot: login successful", "user_id", user.ID, "matrix_id", sender)
	b.reply(ctx, roomID, "✅ Вход выполнен! Вернитесь на сайт.")
}

// reply sends a plain notice into a room via the provider.
func (b *Bot) reply(ctx context.Context, roomID, text string) {
	b.replyDoc(ctx, roomID, notify.New().Text(text).Build())
}

// replyDoc sends a richtext document into a room via the provider.
func (b *Bot) replyDoc(ctx context.Context, roomID string, doc richtext.Doc) {
	msg := notify.Message{Body: doc}
	if _, err := b.provider.Send(ctx, roomID, msg); err != nil {
		b.logger.Warn("matrix bot: reply failed", "room", roomID, "error", err)
	}
}
