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

	"go-service-template/internal/notify"

	"github.com/google/uuid"
)

// botUserRepo is the slice of the user repository the Matrix bot needs to
// link accounts.
type botUserRepo interface {
	IsMatrixIDTaken(ctx context.Context, matrixID string, excludeUserID uuid.UUID) (bool, error)
	SetMatrixLink(ctx context.Context, userID uuid.UUID, matrixID, roomID string) error
}

// Bot is a long-running Matrix client that syncs, auto-joins direct-message
// invites, and consumes account-link commands ("link <token>") users send to
// it. It reuses the Provider's homeserver/token/http for sends and joins, but
// keeps its own long-poll HTTP client for /sync.
type Bot struct {
	provider  *Provider
	userRepo  botUserRepo
	linkStore *LinkStore
	logger    *slog.Logger
	syncHTTP  *http.Client
}

func NewBot(provider *Provider, userRepo botUserRepo, linkStore *LinkStore, logger *slog.Logger) *Bot {
	if logger == nil {
		logger = slog.Default()
	}
	return &Bot{
		provider:  provider,
		userRepo:  userRepo,
		linkStore: linkStore,
		logger:    logger,
		// Long-poll client: its timeout must exceed the /sync timeout below.
		syncHTTP: &http.Client{Timeout: 60 * time.Second},
	}
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
						MsgType string `json:"msgtype"`
						Body    string `json:"body"`
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
			if ev.Type != "m.room.message" || ev.Content.MsgType != "m.text" {
				continue
			}
			// Ignore our own messages (replies echo back in sync).
			if ev.Sender == b.provider.userID {
				continue
			}
			b.handleMessage(ctx, roomID, ev.Sender, ev.Content.Body)
		}
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

// handleMessage consumes a link command and links the sender's Matrix account.
func (b *Bot) handleMessage(ctx context.Context, roomID, sender, body string) {
	token, ok := parseLinkCommand(body)
	if !ok {
		return
	}
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

// reply sends a plain notice into a room via the provider.
func (b *Bot) reply(ctx context.Context, roomID, text string) {
	msg := notify.Message{Body: notify.New().Text(text).Build()}
	if _, err := b.provider.Send(ctx, roomID, msg); err != nil {
		b.logger.Warn("matrix bot: reply failed", "room", roomID, "error", err)
	}
}
