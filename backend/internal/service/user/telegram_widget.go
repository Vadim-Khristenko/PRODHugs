package user

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"time"

	"go-service-template/internal/errorz"
	"go-service-template/internal/models"
	"go-service-template/internal/telegram"
)

// widgetAuthMaxAge is the maximum accepted age of a Login Widget auth_date.
const widgetAuthMaxAge int64 = 86400 // 24h in seconds

// TelegramWidgetData is a Telegram Login Widget payload.
type TelegramWidgetData struct {
	ID        int64
	FirstName string
	LastName  string // optional ("" if absent)
	Username  string // optional
	PhotoURL  string // optional
	AuthDate  int64
	Hash      string
}

// VerifyTelegramWidget checks the HMAC signature and freshness of a Login
// Widget payload against the bot token. now is the current unix time (inject
// for testability).
//
// See https://core.telegram.org/widgets/login#checking-authorization
func VerifyTelegramWidget(d TelegramWidgetData, botToken string, now int64) bool {
	if d.Hash == "" || botToken == "" {
		return false
	}

	// Reject stale payloads.
	if now-d.AuthDate > widgetAuthMaxAge {
		return false
	}

	// Collect present fields (everything except hash).
	fields := map[string]string{
		"id":         strconv.FormatInt(d.ID, 10),
		"first_name": d.FirstName,
		"auth_date":  strconv.FormatInt(d.AuthDate, 10),
	}
	if d.LastName != "" {
		fields["last_name"] = d.LastName
	}
	if d.Username != "" {
		fields["username"] = d.Username
	}
	if d.PhotoURL != "" {
		fields["photo_url"] = d.PhotoURL
	}

	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, k+"="+fields[k])
	}
	dataCheckString := strings.Join(pairs, "\n")

	secret := sha256.Sum256([]byte(botToken))
	mac := hmac.New(sha256.New, secret[:])
	mac.Write([]byte(dataCheckString))
	computed := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(computed), []byte(d.Hash))
}

// LoginViaTelegramWidget verifies a Telegram Login Widget payload against the
// configured bot token and, if valid, resolves/registers the user via the
// existing Telegram login path. Returns errorz.ErrTelegramLoginFailed on an
// invalid or stale payload.
func (s *service) LoginViaTelegramWidget(ctx context.Context, d TelegramWidgetData) (*models.User, error) {
	if !VerifyTelegramWidget(d, s.telegramBotToken, time.Now().Unix()) {
		return nil, errorz.ErrTelegramLoginFailed
	}

	info := &telegram.TelegramUserInfo{
		TelegramID: d.ID,
		Username:   d.Username,
		FirstName:  d.FirstName,
		LastName:   d.LastName,
	}
	return s.LoginViaTelegram(ctx, info)
}
