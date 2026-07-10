package user

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// signWidget computes a valid Telegram Login Widget hash for d against botToken
// and sets d.Hash. It mirrors the reference data_check_string algorithm.
func signWidget(d *TelegramWidgetData, botToken string) {
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
	d.Hash = hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyTelegramWidget(t *testing.T) {
	const botToken = "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
	const now int64 = 1_700_000_000

	base := TelegramWidgetData{
		ID:        42,
		FirstName: "Ada",
		LastName:  "Lovelace",
		Username:  "ada",
		PhotoURL:  "https://t.me/i/userpic/320/ada.jpg",
		AuthDate:  now - 10,
	}

	t.Run("valid full payload verifies", func(t *testing.T) {
		d := base
		signWidget(&d, botToken)
		if !VerifyTelegramWidget(d, botToken, now) {
			t.Fatal("expected valid payload to verify")
		}
	})

	t.Run("valid minimal payload verifies", func(t *testing.T) {
		d := TelegramWidgetData{
			ID:        7,
			FirstName: "Solo",
			AuthDate:  now - 5,
		}
		signWidget(&d, botToken)
		if !VerifyTelegramWidget(d, botToken, now) {
			t.Fatal("expected minimal payload to verify")
		}
	})

	t.Run("mutated field fails", func(t *testing.T) {
		d := base
		signWidget(&d, botToken)
		d.FirstName = "Grace"
		if VerifyTelegramWidget(d, botToken, now) {
			t.Fatal("expected mutated payload to fail")
		}
	})

	t.Run("mutated id fails", func(t *testing.T) {
		d := base
		signWidget(&d, botToken)
		d.ID = 43
		if VerifyTelegramWidget(d, botToken, now) {
			t.Fatal("expected mutated id to fail")
		}
	})

	t.Run("old auth_date fails", func(t *testing.T) {
		d := base
		d.AuthDate = now - 86401
		signWidget(&d, botToken)
		if VerifyTelegramWidget(d, botToken, now) {
			t.Fatal("expected stale auth_date to fail")
		}
	})

	t.Run("auth_date exactly at limit passes", func(t *testing.T) {
		d := base
		d.AuthDate = now - 86400
		signWidget(&d, botToken)
		if !VerifyTelegramWidget(d, botToken, now) {
			t.Fatal("expected auth_date at exactly 24h to pass")
		}
	})

	t.Run("wrong token fails", func(t *testing.T) {
		d := base
		signWidget(&d, botToken)
		if VerifyTelegramWidget(d, "999999:WRONG-token-value", now) {
			t.Fatal("expected wrong token to fail")
		}
	})

	t.Run("empty hash fails", func(t *testing.T) {
		d := base
		d.Hash = ""
		if VerifyTelegramWidget(d, botToken, now) {
			t.Fatal("expected empty hash to fail")
		}
	})
}
