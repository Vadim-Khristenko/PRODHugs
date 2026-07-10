package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"go-service-template/internal/models"
	legacytg "go-service-template/internal/telegram"

	tgbot "github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
	"github.com/google/uuid"
)

// pendingHugComment carries a /hug comment from the group chat where it was
// typed across to the inline-keyboard callback that finally picks the hug
// type. The callback_data field is capped at 64 bytes, so we can't smuggle a
// 200-char comment through it — keep it in-memory keyed by (initiator,target)
// with a short TTL.
type pendingHugComment struct {
	comment    string
	sourceChat int64
	isGroup    bool
	createdAt  time.Time
}

const pendingHugCommentTTL = 10 * time.Minute

// botUserRepo is the minimal interface the bot needs for account linking,
// lookups and admin moderation commands. Repository-level methods only.
type botUserRepo interface {
	SetTelegramID(ctx context.Context, userID uuid.UUID, telegramID int64) (*models.User, error)
	IsTelegramIDTaken(ctx context.Context, telegramID int64, excludeUserID uuid.UUID) (bool, error)
	GetTelegramID(ctx context.Context, userID uuid.UUID) (*int64, error)
	GetByTelegramID(ctx context.Context, telegramID int64) (*models.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	BanUser(ctx context.Context, id uuid.UUID) (*models.User, error)
	UnbanUser(ctx context.Context, id uuid.UUID) (*models.User, error)
	// Block lifecycle. MarkTelegramBlocked is set when Telegram returns 403
	// during a send; ClearTelegramBlocked is best-effort and called on
	// every incoming update so a user who unblocked the bot is re-enabled
	// without operator intervention.
	MarkTelegramBlocked(ctx context.Context, userID uuid.UUID) error
	ClearTelegramBlocked(ctx context.Context, userID uuid.UUID) error
}

// botAnnouncementSvc is the admin surface backed by the user service:
// announcements plus admin balance grants.
type botAnnouncementSvc interface {
	CreateAnnouncement(ctx context.Context, adminID uuid.UUID, message string) (*models.Announcement, error)
	DeactivateAnnouncement(ctx context.Context, id uuid.UUID) error
	// AdminUpdateBalance SETS the target user's coin balance to amount
	// (absolute value, mirroring the admin panel), returning the new balance.
	AdminUpdateBalance(ctx context.Context, userID uuid.UUID, amount int32) (*models.Balance, error)
}

// hugAcceptor is the slice of the hug service the bot consumes: button
// callbacks (Accept/Decline), the /stats activity, /me, /hug, /daily.
type hugAcceptor interface {
	SuggestHug(ctx context.Context, giverID, receiverID uuid.UUID, hugType string, comment *string, captchaToken *string) (*models.Hug, *models.User, error)
	AcceptHug(ctx context.Context, hugID, receiverID uuid.UUID) (*models.Hug, error)
	DeclineHug(ctx context.Context, hugID, receiverID uuid.UUID) error
	ClaimDailyReward(ctx context.Context, userID uuid.UUID) (amount, streakDays, newBalance int32, alreadyClaimed bool, err error)
	GetHugActivity(ctx context.Context) ([]*models.HugActivityItem, error)
	GetUserStats(ctx context.Context, userID uuid.UUID, gender *string) (*models.UserStats, error)
	GetHugHistory(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]*models.HugFeedItem, error)
	GetBalance(ctx context.Context, userID uuid.UUID) (*models.Balance, error)
}

// telegramLoginService handles the auth/registration logic for Telegram login.
type telegramLoginService interface {
	LoginViaTelegram(ctx context.Context, info *legacytg.TelegramUserInfo) (*models.User, error)
}

// Bot is a long-polling Telegram bot that handles /start deep-link commands,
// inline keyboard callbacks for hug actions, /me, /stats, /daily, /hug, and
// admin /announce /unannounce /ban /unban commands. It is inbound-only:
// outbound hug notifications are delivered by the notify Provider/Router.
type Bot struct {
	tg          *tgbot.Bot
	client      *legacytg.Client
	linkStore   *legacytg.LinkStore
	loginStore  *legacytg.LoginStore
	loginSvc    telegramLoginService
	userRepo    botUserRepo
	hugSvc      hugAcceptor
	announceSvc botAnnouncementSvc
	logger      *slog.Logger
	enabled     bool

	pendingMu       sync.Mutex
	pendingComments map[string]pendingHugComment
}

// pendingKey builds the map key for a pending /hug comment.
func pendingKey(initiatorTG int64, targetID uuid.UUID) string {
	return fmt.Sprintf("%d:%s", initiatorTG, targetID.String())
}

// stashPendingComment stores a /hug comment for later use by the
// type-picker callback. Also opportunistically GCs stale entries.
func (b *Bot) stashPendingComment(initiatorTG int64, targetID uuid.UUID, c pendingHugComment) {
	b.pendingMu.Lock()
	defer b.pendingMu.Unlock()
	now := time.Now()
	for k, v := range b.pendingComments {
		if now.Sub(v.createdAt) > pendingHugCommentTTL {
			delete(b.pendingComments, k)
		}
	}
	b.pendingComments[pendingKey(initiatorTG, targetID)] = c
}

// popPendingComment retrieves and removes the pending comment for this
// (initiator,target) pair. Returns ok=false when there isn't one, or it
// has aged past the TTL.
func (b *Bot) popPendingComment(initiatorTG int64, targetID uuid.UUID) (pendingHugComment, bool) {
	b.pendingMu.Lock()
	defer b.pendingMu.Unlock()
	key := pendingKey(initiatorTG, targetID)
	c, ok := b.pendingComments[key]
	if !ok {
		return pendingHugComment{}, false
	}
	delete(b.pendingComments, key)
	if time.Since(c.createdAt) > pendingHugCommentTTL {
		return pendingHugComment{}, false
	}
	return c, true
}

// NewBot creates a new inbound Telegram bot. If the client is disabled (no
// token), Run() is a no-op and replies fall back to the raw HTTP client.
func NewBot(client *legacytg.Client, linkStore *legacytg.LinkStore, userRepo botUserRepo, hugSvc hugAcceptor, announceSvc botAnnouncementSvc, logger *slog.Logger) *Bot {
	b := &Bot{
		client:          client,
		linkStore:       linkStore,
		userRepo:        userRepo,
		hugSvc:          hugSvc,
		announceSvc:     announceSvc,
		logger:          logger,
		pendingComments: make(map[string]pendingHugComment),
	}

	// enabled reflects only whether a token is configured. The actual API
	// client (which does a getMe round-trip) is created lazily in Run, off the
	// startup path and with retries, so a transient network blip at boot never
	// permanently disables the bot.
	b.enabled = client.Enabled()
	return b
}

// SetLoginStore configures the login store and service for Telegram login.
// Called after construction to break circular dependencies.
func (b *Bot) SetLoginStore(store *legacytg.LoginStore, svc telegramLoginService) {
	b.loginStore = store
	b.loginSvc = svc
}

// Run starts the long-polling bot. Blocks until ctx is cancelled.
func (b *Bot) Run(ctx context.Context) {
	if !b.enabled {
		b.logger.Info("telegram bot disabled (no token)")
		return
	}

	// Create the API client here (does a getMe) with retries, so a slow or
	// flaky network at startup delays the bot without blocking app boot and
	// without permanently disabling it.
	for attempt := 1; ; attempt++ {
		if ctx.Err() != nil {
			return
		}
		tg, err := tgbot.New(
			b.client.Token(),
			tgbot.WithDefaultHandler(b.handleUpdate),
			tgbot.WithCheckInitTimeout(30*time.Second),
		)
		if err == nil {
			b.tg = tg
			break
		}
		b.logger.Warn("telegram bot: init failed, retrying", "attempt", attempt, "error", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Second):
		}
	}

	b.logger.Info("telegram bot started (long-polling)")
	b.tg.Start(ctx)
	b.logger.Info("telegram bot stopped")
}

func (b *Bot) handleUpdate(ctx context.Context, _ *tgbot.Bot, update *tgmodels.Update) {
	if update.CallbackQuery != nil {
		// Receiving any callback means the user can talk to us — clear a
		// stale blocked flag if it was set.
		b.maybeClearBlocked(ctx, update.CallbackQuery.From.ID)
		b.handleCallback(ctx, update.CallbackQuery)
		return
	}

	if update.Message == nil {
		return
	}
	// Same reasoning for plain messages.
	if update.Message.From != nil {
		b.maybeClearBlocked(ctx, update.Message.From.ID)
	}

	text := strings.TrimSpace(update.Message.Text)
	switch {
	case strings.HasPrefix(text, "/start"):
		b.handleStart(ctx, update.Message)
	case text == "/me" || strings.HasPrefix(text, "/me "):
		b.handleMe(ctx, update.Message)
	case text == "/stats" || strings.HasPrefix(text, "/stats "):
		b.handleStats(ctx, update.Message)
	case text == "/daily" || strings.HasPrefix(text, "/daily "):
		b.handleDaily(ctx, update.Message)
	case text == "/hug" || strings.HasPrefix(text, "/hug "):
		b.handleHug(ctx, update.Message)
	case strings.HasPrefix(text, "/announce"):
		b.handleAnnounce(ctx, update.Message)
	case strings.HasPrefix(text, "/unannounce"):
		b.handleUnannounce(ctx, update.Message)
	case strings.HasPrefix(text, "/grant"):
		b.handleGrant(ctx, update.Message)
	case strings.HasPrefix(text, "/userinfo"):
		b.handleUserinfo(ctx, update.Message)
	case strings.HasPrefix(text, "/ban"):
		b.handleBan(ctx, update.Message, true)
	case strings.HasPrefix(text, "/unban"):
		b.handleBan(ctx, update.Message, false)
	case text == "/help" || strings.HasPrefix(text, "/help "):
		b.handleHelp(ctx, update.Message)
	}
}

// canReachPM reports whether the bot can currently send a private message
// to the given user. Used by /hug to decide whether to forward a group-typed
// comment to the recipient's DM or to nudge the group that the DM isn't
// open yet. Conservatively returns false on any uncertainty.
func (b *Bot) canReachPM(ctx context.Context, u *models.User) bool {
	if u == nil || u.TelegramID == nil {
		return false
	}
	if !b.enabled {
		// Without the API client the raw fallback won't reach a fresh DM either.
		return false
	}
	_, err := b.tg.GetChat(ctx, &tgbot.GetChatParams{ChatID: *u.TelegramID})
	if err != nil {
		// "Bad Request: chat not found" — user never started the bot.
		// "Forbidden: bot was blocked by the user" — explicit block.
		// Both mean: don't try to PM, ask group to fix it.
		return false
	}
	return true
}

// maybeClearBlocked is a best-effort cleanup: a user we hear from just
// proved they can talk to the bot, so any standing telegram_blocked_at
// is definitely stale. The repo update is idempotent (no-op if already
// cleared), so we don't bother filtering — just fire it on every update
// we get from a linked account. Not-found and errors are ignored (debug).
func (b *Bot) maybeClearBlocked(ctx context.Context, tgID int64) {
	u, err := b.userRepo.GetByTelegramID(ctx, tgID)
	if err != nil || u == nil {
		b.logger.Debug("telegram bot: clear blocked — user not linked", "telegram_id", tgID)
		return
	}
	if err := b.userRepo.ClearTelegramBlocked(ctx, u.ID); err != nil {
		b.logger.Debug("telegram bot: clear blocked failed", "user_id", u.ID, "error", err)
	}
}

func (b *Bot) handleStart(ctx context.Context, msg *tgmodels.Message) {
	chatID := msg.Chat.ID

	parts := strings.SplitN(msg.Text, " ", 2)
	if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
		b.reply(ctx, chatID, "Чтобы привязать аккаунт, используй настройки на сайте. Доступные команды: /me, /stats")
		return
	}

	token := strings.TrimSpace(parts[1])

	// Handle login_ prefixed tokens for Telegram login flow
	if strings.HasPrefix(token, "login_") {
		b.handleLoginStart(ctx, msg, strings.TrimPrefix(token, "login_"))
		return
	}

	// Original account-linking flow
	userID, ok := b.linkStore.ConsumeToken(token)
	if !ok {
		b.reply(ctx, chatID, "Ссылка недействительна или истекла. Попробуй снова через настройки приложения")
		return
	}

	taken, err := b.userRepo.IsTelegramIDTaken(ctx, chatID, userID)
	if err != nil {
		b.logger.Error("telegram bot: failed to check telegram_id", "error", err)
		b.reply(ctx, chatID, "Произошла ошибка. Попробуй позже :(")
		return
	}
	if taken {
		b.reply(ctx, chatID, "Этот Telegram аккаунт уже привязан к другому пользователю :(")
		return
	}

	_, err = b.userRepo.SetTelegramID(ctx, userID, chatID)
	if err != nil {
		b.logger.Error("telegram bot: failed to set telegram_id", "user_id", userID, "chat_id", chatID, "error", err)
		b.reply(ctx, chatID, "Произошла ошибка при привязке. Попробуй позже :(")
		return
	}

	b.logger.Info("telegram bot: account linked", "user_id", userID, "chat_id", chatID)
	b.reply(ctx, chatID, "✅ Аккаунт привязан! Теперь ты не пропустишь обнимашки от любимых продовцев")
}

func (b *Bot) handleLoginStart(ctx context.Context, msg *tgmodels.Message, botToken string) {
	chatID := msg.Chat.ID

	if b.loginStore == nil || b.loginSvc == nil {
		b.reply(ctx, chatID, "Вход через Telegram временно недоступен")
		return
	}

	pollToken, ok := b.loginStore.ConsumeBotToken(botToken)
	if !ok {
		b.reply(ctx, chatID, "Ссылка недействительна или истекла. Попробуй снова")
		return
	}

	// Build Telegram user info from the message sender
	info := &legacytg.TelegramUserInfo{
		TelegramID: chatID,
		FirstName:  msg.From.FirstName,
		LastName:   msg.From.LastName,
	}
	if msg.From.Username != "" {
		info.Username = msg.From.Username
	}

	// Store user info on the session
	b.loginStore.SetSessionUserInfo(pollToken, info)

	// Attempt login/registration via the service
	user, err := b.loginSvc.LoginViaTelegram(ctx, info)
	if err != nil {
		b.logger.Error("telegram bot: login failed", "chat_id", chatID, "error", err)
		b.loginStore.FailSession(pollToken, err.Error())
		b.reply(ctx, chatID, "Не удалось войти: "+friendlyLoginError(err))
		return
	}

	// Mark session as authenticated
	b.loginStore.AuthenticateSession(pollToken, user.ID)
	b.logger.Info("telegram bot: login successful", "user_id", user.ID, "chat_id", chatID)
	b.reply(ctx, chatID, "✅ Вход выполнен! Можешь вернуться в приложение")
}

func friendlyLoginError(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "banned"):
		return "ваш аккаунт заблокирован"
	default:
		return "попробуйте позже"
	}
}

// handleCallback dispatches inline-keyboard callbacks. All callback data is
// in the unified action-token format "<domain>.<verb>:<arg>" (parseAction).
// Hug notification buttons (hug.accept:/hug.decline:, sent by the Provider)
// and the /hug type picker (hug.pick:) all route through here.
func (b *Bot) handleCallback(ctx context.Context, cb *tgmodels.CallbackQuery) {
	domain, verb, arg, ok := parseAction(cb.Data)
	if !ok || domain != "hug" {
		return
	}

	if verb == "pick" {
		b.handleHugPickCallback(ctx, cb, arg)
		return
	}

	if verb != "accept" && verb != "decline" {
		return
	}

	chatID := cb.Message.Message.Chat.ID
	msgID := cb.Message.Message.ID
	originalText := cb.Message.Message.Text

	hugID, err := uuid.Parse(arg)
	if err != nil {
		b.answerCallback(ctx, cb.ID, "Ошибка: некорректные данные")
		return
	}

	// Look up the user by their Telegram chat ID
	user, err := b.userRepo.GetByTelegramID(ctx, chatID)
	if err != nil {
		b.answerCallback(ctx, cb.ID, "Ваш Telegram не привязан к аккаунту")
		return
	}

	switch verb {
	case "accept":
		_, err = b.hugSvc.AcceptHug(ctx, hugID, user.ID)
		if err != nil {
			b.logger.Error("telegram bot: failed to accept hug", "hug_id", hugID, "error", err)
			b.answerCallback(ctx, cb.ID, "Не удалось принять объятие: "+friendlyError(err))
			return
		}
		b.answerCallback(ctx, cb.ID, "Объятие принято! 🤗")
		b.editMessageText(ctx, chatID, msgID, originalText+"\n\n✅ <b>Принято!</b>")

	case "decline":
		err = b.hugSvc.DeclineHug(ctx, hugID, user.ID)
		if err != nil {
			b.logger.Error("telegram bot: failed to decline hug", "hug_id", hugID, "error", err)
			b.answerCallback(ctx, cb.ID, "Не удалось отклонить: "+friendlyError(err))
			return
		}
		b.answerCallback(ctx, cb.ID, "Объятие отклонено")
		b.editMessageText(ctx, chatID, msgID, originalText+"\n\n❌ <b>Отклонено</b>")
	}
}

func (b *Bot) reply(ctx context.Context, chatID int64, text string) {
	if !b.enabled {
		_ = b.client.SendMessage(chatID, text)
		return
	}
	_, err := b.tg.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: tgmodels.ParseModeHTML,
	})
	if err != nil {
		b.logger.Error("telegram bot: failed to send message", "chat_id", chatID, "error", err)
	}
}

func (b *Bot) answerCallback(ctx context.Context, callbackID string, text string) {
	_, err := b.tg.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
		CallbackQueryID: callbackID,
		Text:            text,
		ShowAlert:       false,
	})
	if err != nil {
		b.logger.Error("telegram bot: failed to answer callback", "error", err)
	}
}

func (b *Bot) editMessageText(ctx context.Context, chatID int64, messageID int, newText string) {
	_, err := b.tg.EditMessageText(ctx, &tgbot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        newText,
		ParseMode:   tgmodels.ParseModeHTML,
		ReplyMarkup: &tgmodels.InlineKeyboardMarkup{InlineKeyboard: [][]tgmodels.InlineKeyboardButton{}},
	})
	if err != nil {
		b.logger.Error("telegram bot: failed to edit message", "error", err)
	}
}

func friendlyError(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "not found"):
		return "объятие не найдено"
	case strings.Contains(msg, "not pending"):
		return "объятие уже обработано"
	case strings.Contains(msg, "expired"):
		return "объятие истекло"
	default:
		return "попробуйте позже"
	}
}

// htmlEscape escapes the four HTML-significant chars Telegram's HTML
// parse mode cares about. We only ever build messages with controlled
// fields, but display names and usernames are user-supplied, so escape
// those at the rendering boundary.
func htmlEscape(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
	)
	return r.Replace(s)
}

// displayName returns the user's display name, falling back to username.
func displayName(u *models.User) string {
	if u.DisplayName != nil && *u.DisplayName != "" {
		return *u.DisplayName
	}
	return u.Username
}
