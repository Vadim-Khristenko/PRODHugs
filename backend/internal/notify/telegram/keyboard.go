package telegram

import "go-service-template/internal/notify"

// inlineButton / inlineKeyboard mirror the Telegram Bot API reply_markup
// JSON for an inline keyboard.
type inlineButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

type inlineKeyboard struct {
	InlineKeyboard [][]inlineButton `json:"inline_keyboard"`
}

// toInlineKeyboard maps provider-agnostic buttons to Telegram's inline
// keyboard, carrying each Button.Action verbatim as callback_data.
func toInlineKeyboard(btns [][]notify.Button) inlineKeyboard {
	rows := make([][]inlineButton, 0, len(btns))
	for _, row := range btns {
		r := make([]inlineButton, 0, len(row))
		for _, b := range row {
			r = append(r, inlineButton{Text: b.Label, CallbackData: b.Action})
		}
		rows = append(rows, r)
	}
	return inlineKeyboard{InlineKeyboard: rows}
}
