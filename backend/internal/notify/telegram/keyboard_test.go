package telegram

import (
	"testing"

	"go-service-template/internal/notify"
)

func TestToInlineKeyboard(t *testing.T) {
	btns := [][]notify.Button{{
		{Label: "Обнять 🤗", Action: "hug.accept:abc"},
		{Label: "Отклонить", Action: "hug.decline:abc"},
	}}
	kb := toInlineKeyboard(btns)
	if len(kb.InlineKeyboard) != 1 || len(kb.InlineKeyboard[0]) != 2 {
		t.Fatalf("shape wrong: %+v", kb.InlineKeyboard)
	}
	if kb.InlineKeyboard[0][0].CallbackData != "hug.accept:abc" {
		t.Fatalf("callback wrong: %q", kb.InlineKeyboard[0][0].CallbackData)
	}
	if kb.InlineKeyboard[0][0].Text != "Обнять 🤗" {
		t.Fatalf("label wrong: %q", kb.InlineKeyboard[0][0].Text)
	}
}
