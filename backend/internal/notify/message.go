// Package notify is the provider-agnostic notification core. Domain code
// calls the Notifier facade; the Router renders each Message via the
// enabled Providers (Telegram, Matrix) and records SentRefs for later edits.
package notify

import "go-service-template/internal/notify/richtext"

// Message is a provider-agnostic notification: a rich body plus optional
// rows of inline buttons.
type Message struct {
	Body    richtext.Doc
	Buttons [][]Button
}

// Button carries a typed action token routed back to a shared handler.
// Action format is "<domain>.<verb>:<arg>", e.g. "hug.accept:<uuid>".
type Button struct {
	Label  string
	Action string
}

// SentRef identifies a delivered message so it can be edited later.
type SentRef struct {
	Provider   string
	ChatRef    string
	MessageRef string
}

// Address holds a user's per-channel identities. A nil field means the user
// has not linked that channel.
type Address struct {
	TelegramID *int64
	MatrixID   *string // Matrix user id — identity, used for uniqueness checks
	MatrixRoom *string // Matrix DM room id — delivery target for notifications
}
