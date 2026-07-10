package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"go-service-template/internal/notify"
)

// Provider delivers notifications to Telegram via raw Bot API HTTP. It prefers
// the new sendRichMessage method and falls back to classic sendMessage+HTML if
// the API reports that method unsupported.
type Provider struct {
	token   string
	http    *http.Client
	logger  *slog.Logger
	richOff atomic.Bool // set once sendRichMessage proves unsupported
}

func New(token string, logger *slog.Logger) *Provider {
	if logger == nil {
		logger = slog.Default()
	}
	return &Provider{
		token:  token,
		http:   &http.Client{Timeout: 10 * time.Second},
		logger: logger,
	}
}

func (p *Provider) Name() string  { return "telegram" }
func (p *Provider) Enabled() bool { return p.token != "" }

type apiResponse struct {
	OK          bool            `json:"ok"`
	ErrorCode   int             `json:"error_code"`
	Description string          `json:"description"`
	Result      json.RawMessage `json:"result"`
}

func (p *Provider) call(ctx context.Context, method string, payload any) (*apiResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/%s", p.token, method)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	var ar apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return nil, err
	}
	return &ar, nil
}

// parseMessageID extracts the message_id from a send result. ok is false when
// the result is missing/malformed or the id is zero — the message was
// delivered but is not addressable for a later edit.
func parseMessageID(raw json.RawMessage) (int64, bool) {
	var r struct {
		MessageID int64 `json:"message_id"`
	}
	if err := json.Unmarshal(raw, &r); err != nil || r.MessageID == 0 {
		return 0, false
	}
	return r.MessageID, true
}

// isUnknownMethod reports whether the API response indicates the method itself
// is not available (as opposed to a per-request error like a bad chat).
func isUnknownMethod(ar *apiResponse) bool {
	if ar.ErrorCode == 404 {
		return true
	}
	d := strings.ToLower(ar.Description)
	return strings.Contains(d, "method not found") || strings.Contains(d, "method is not")
}

// sentRef builds the SentRef for a delivered message. If the message id can't
// be determined, MessageRef is left empty so the Router does not record a
// non-editable ref (the message was still delivered).
func (p *Provider) sentRef(chatRef string, ar *apiResponse) notify.SentRef {
	ref := notify.SentRef{Provider: "telegram", ChatRef: chatRef}
	if id, ok := parseMessageID(ar.Result); ok {
		ref.MessageRef = strconv.FormatInt(id, 10)
	}
	return ref
}

// Send renders msg and delivers it to chatRef. Prefers sendRichMessage.
func (p *Provider) Send(ctx context.Context, chatRef string, msg notify.Message) (notify.SentRef, error) {
	chatID, err := strconv.ParseInt(chatRef, 10, 64)
	if err != nil {
		return notify.SentRef{}, fmt.Errorf("telegram: bad chat ref %q: %w", chatRef, err)
	}
	html := notify.RenderTelegram(msg.Body)
	var kb *inlineKeyboard
	if len(msg.Buttons) > 0 {
		k := toInlineKeyboard(msg.Buttons)
		kb = &k
	}

	if !p.richOff.Load() {
		rich := map[string]any{"html": html, "skip_entity_detection": true}
		payload := map[string]any{"chat_id": chatID, "rich_message": rich}
		if kb != nil {
			payload["reply_markup"] = kb
		}
		ar, err := p.call(ctx, "sendRichMessage", payload)
		if err != nil {
			return notify.SentRef{}, err
		}
		if ar.OK {
			return p.sentRef(chatRef, ar), nil
		}
		if ar.ErrorCode == 403 {
			return notify.SentRef{}, notify.Blocked(fmt.Errorf("telegram 403: %s", ar.Description))
		}
		if isUnknownMethod(ar) {
			p.richOff.Store(true)
			p.logger.Warn("telegram: sendRichMessage unsupported, falling back to classic sendMessage", "description", ar.Description)
			// fall through to classic path below
		} else {
			return notify.SentRef{}, fmt.Errorf("telegram sendRichMessage failed: %s (%d)", ar.Description, ar.ErrorCode)
		}
	}

	// Classic fallback: sendMessage with HTML parse mode. Reuses the same
	// rendered string, which is safe because notification templates use only
	// nodes valid in classic HTML mode (bold/italic/underline/strike/code/
	// link/blockquote/pre). Rich-only nodes (tg-spoiler, tg-emoji, headings)
	// must not be used in notification bodies or this path would 400.
	payload := map[string]any{"chat_id": chatID, "text": html, "parse_mode": "HTML"}
	if kb != nil {
		payload["reply_markup"] = kb
	}
	ar, err := p.call(ctx, "sendMessage", payload)
	if err != nil {
		return notify.SentRef{}, err
	}
	if !ar.OK {
		if ar.ErrorCode == 403 {
			return notify.SentRef{}, notify.Blocked(fmt.Errorf("telegram 403: %s", ar.Description))
		}
		return notify.SentRef{}, fmt.Errorf("telegram sendMessage failed: %s (%d)", ar.Description, ar.ErrorCode)
	}
	return p.sentRef(chatRef, ar), nil
}

// Edit updates a previously sent message: replaces the text and drops the
// inline buttons. Best-effort — if editMessageText fails (e.g. the message is
// too old or was rich), it still tries to strip the buttons.
func (p *Provider) Edit(ctx context.Context, ref notify.SentRef, msg notify.Message) error {
	chatID, err := strconv.ParseInt(ref.ChatRef, 10, 64)
	if err != nil {
		return fmt.Errorf("telegram: bad chat ref %q: %w", ref.ChatRef, err)
	}
	msgID, err := strconv.ParseInt(ref.MessageRef, 10, 64)
	if err != nil {
		return fmt.Errorf("telegram: bad message ref %q: %w", ref.MessageRef, err)
	}
	html := notify.RenderTelegram(msg.Body)
	empty := inlineKeyboard{InlineKeyboard: [][]inlineButton{}}
	ar, err := p.call(ctx, "editMessageText", map[string]any{
		"chat_id":      chatID,
		"message_id":   msgID,
		"text":         html,
		"parse_mode":   "HTML",
		"reply_markup": empty,
	})
	if err != nil {
		return err
	}
	if !ar.OK {
		// Best effort: at least remove the buttons so they can't be re-clicked.
		_, _ = p.call(ctx, "editMessageReplyMarkup", map[string]any{
			"chat_id":      chatID,
			"message_id":   msgID,
			"reply_markup": empty,
		})
		return fmt.Errorf("telegram editMessageText failed: %s (%d)", ar.Description, ar.ErrorCode)
	}
	return nil
}

// compile-time interface assertion
var _ notify.Provider = (*Provider)(nil)
