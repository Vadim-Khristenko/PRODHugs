package telegram

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Client is a minimal Telegram Bot API client.
// If botToken is empty, all methods are no-ops (feature disabled).
type Client struct {
	token      string
	httpClient *http.Client
}

// New creates a new Telegram Bot API client.
// Passing an empty token results in a disabled (no-op) client.
func New(botToken string) *Client {
	return &Client{
		token: botToken,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Enabled returns true if the client has a bot token configured.
func (c *Client) Enabled() bool {
	return c.token != ""
}

// Token returns the configured bot token. Used by the inbound bot runner to
// construct the go-telegram long-polling client from the same credentials.
func (c *Client) Token() string {
	return c.token
}

// telegramResponse is the common response wrapper from the Telegram API.
type telegramResponse struct {
	OK          bool            `json:"ok"`
	Description string          `json:"description"`
	Result      json.RawMessage `json:"result"`
}

// SendMessage sends a text message to the given chat ID.
// Returns nil if the client is disabled.
func (c *Client) SendMessage(chatID int64, text string) error {
	if !c.Enabled() {
		return nil
	}

	params := url.Values{}
	params.Set("chat_id", strconv.FormatInt(chatID, 10))
	params.Set("text", text)
	params.Set("parse_mode", "HTML")

	return c.call("sendMessage", params)
}

// GetChat validates that the bot can reach the given chat ID.
// Returns nil on success, error if the chat is unreachable.
func (c *Client) GetChat(chatID int64) error {
	if !c.Enabled() {
		return nil
	}

	params := url.Values{}
	params.Set("chat_id", strconv.FormatInt(chatID, 10))

	return c.call("getChat", params)
}

func (c *Client) call(method string, params url.Values) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/%s", c.token, method)

	resp, err := c.httpClient.Post(apiURL, "application/x-www-form-urlencoded", strings.NewReader(params.Encode()))
	if err != nil {
		return fmt.Errorf("telegram %s: %w", method, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("telegram %s: failed to read response: %w", method, err)
	}

	var tResp telegramResponse
	if err := json.Unmarshal(body, &tResp); err != nil {
		return fmt.Errorf("telegram %s: failed to decode response: %w", method, err)
	}

	if !tResp.OK {
		return fmt.Errorf("telegram %s: %s", method, tResp.Description)
	}

	return nil
}
