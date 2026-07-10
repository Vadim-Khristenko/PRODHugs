// Package matrix implements a notify.Provider for the Matrix protocol using
// the Matrix Client-Server REST API directly (no external SDK). It is dormant
// unless a homeserver URL, user id and access token are all configured.
package matrix

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"go-service-template/internal/notify"
)

// Provider delivers notifications to Matrix rooms. chatRef is a room id.
type Provider struct {
	homeserver string // base URL, e.g. https://matrix.org
	userID     string // @bot:server — informational, sender is the token owner
	token      string
	http       *http.Client
	logger     *slog.Logger
	txn        atomic.Int64 // monotonic transaction-id counter
}

func New(homeserver, userID, token string, logger *slog.Logger) *Provider {
	if logger == nil {
		logger = slog.Default()
	}
	return &Provider{
		homeserver: strings.TrimRight(homeserver, "/"),
		userID:     userID,
		token:      token,
		http:       &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
	}
}

func (p *Provider) Name() string { return "matrix" }

// Enabled reports whether all three credentials are present.
func (p *Provider) Enabled() bool {
	return p.homeserver != "" && p.userID != "" && p.token != ""
}

// nextTxn returns a per-request transaction id. Matrix uses it to dedupe
// retries; a monotonic counter is sufficient for our fire-and-forget sends.
func (p *Provider) nextTxn() string {
	return "prodhugs-" + strconv.FormatInt(p.txn.Add(1), 10)
}

// buildContent renders msg into a Matrix m.notice event content. Matrix has
// no native inline buttons, so buttons degrade to a trailing text list.
func buildContent(msg notify.Message) map[string]any {
	plain := notify.RenderPlain(msg.Body)
	html := notify.RenderMatrixHTML(msg.Body)
	if len(msg.Buttons) > 0 {
		var labels []string
		for _, row := range msg.Buttons {
			for _, b := range row {
				labels = append(labels, b.Label)
			}
		}
		hint := strings.Join(labels, " · ")
		plain += "\n" + hint
		html += "<br><em>" + hint + "</em>"
	}
	return map[string]any{
		"msgtype":        "m.notice",
		"body":           plain,
		"format":         "org.matrix.custom.html",
		"formatted_body": html,
	}
}

type sendResponse struct {
	EventID string `json:"event_id"`
	ErrCode string `json:"errcode"`
	Error   string `json:"error"`
}

// putEvent PUTs an m.room.message event to a room and returns the event id.
func (p *Provider) putEvent(ctx context.Context, roomID string, content map[string]any) (string, error) {
	txn := p.nextTxn()
	endpoint := fmt.Sprintf("%s/_matrix/client/v3/rooms/%s/send/m.room.message/%s",
		p.homeserver, url.PathEscape(roomID), url.PathEscape(txn))
	body, err := json.Marshal(content)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.token)
	resp, err := p.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var sr sendResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return "", err
	}
	if sr.EventID == "" {
		return "", fmt.Errorf("matrix send failed: %s %s", sr.ErrCode, sr.Error)
	}
	return sr.EventID, nil
}

// Send delivers msg to the room identified by chatRef.
func (p *Provider) Send(ctx context.Context, chatRef string, msg notify.Message) (notify.SentRef, error) {
	eventID, err := p.putEvent(ctx, chatRef, buildContent(msg))
	if err != nil {
		return notify.SentRef{}, err
	}
	return notify.SentRef{Provider: "matrix", ChatRef: chatRef, MessageRef: eventID}, nil
}

// Edit replaces a previously sent event using an m.replace relation.
func (p *Provider) Edit(ctx context.Context, ref notify.SentRef, msg notify.Message) error {
	newContent := buildContent(msg)
	// An m.replace carries the replacement under m.new_content and a
	// fallback (prefixed) copy at the top level for clients that don't
	// understand edits.
	content := map[string]any{
		"msgtype":        newContent["msgtype"],
		"body":           "* " + newContent["body"].(string),
		"format":         newContent["format"],
		"formatted_body": newContent["formatted_body"],
		"m.new_content":  newContent,
		"m.relates_to": map[string]any{
			"rel_type": "m.replace",
			"event_id": ref.MessageRef,
		},
	}
	_, err := p.putEvent(ctx, ref.ChatRef, content)
	return err
}

// compile-time interface assertion
var _ notify.Provider = (*Provider)(nil)
