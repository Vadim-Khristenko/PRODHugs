package notify

import (
	"context"
	"errors"
)

// Provider is a single delivery channel (Telegram, Matrix).
type Provider interface {
	Name() string
	Enabled() bool
	// Send renders and delivers msg to chatRef, returning a SentRef.
	Send(ctx context.Context, chatRef string, msg Message) (SentRef, error)
	// Edit replaces a previously sent message identified by ref.
	Edit(ctx context.Context, ref SentRef, msg Message) error
}

// blockedError signals that the recipient has blocked the bot (Telegram 403).
type blockedError struct{ err error }

func (e blockedError) Error() string { return e.err.Error() }
func (e blockedError) Unwrap() error { return e.err }

// Blocked wraps err as a blocked-recipient signal.
func Blocked(err error) error { return blockedError{err: err} }

// IsBlocked reports whether err signals a blocked recipient.
func IsBlocked(err error) bool {
	var b blockedError
	return errors.As(err, &b)
}
