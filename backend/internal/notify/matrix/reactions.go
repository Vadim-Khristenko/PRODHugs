package matrix

import "strings"

// Matrix has no inline buttons, so the bot places reactions as button
// substitutes and listens for the user's reaction. Keys are text (Matrix
// reactions accept arbitrary strings) derived deterministically from a
// notify.Button action verb, so no per-message state is needed.
const (
	reactAccept  = "🤗 Обнять"
	reactDecline = "❌ Отклонить"
)

// reactionForAction maps a button action ("hug.accept:<id>") to the reaction
// key the bot should place. ok=false for actions with no reaction mapping.
func reactionForAction(action string) (string, bool) {
	switch {
	case strings.HasPrefix(action, "hug.accept:"):
		return reactAccept, true
	case strings.HasPrefix(action, "hug.decline:"):
		return reactDecline, true
	}
	return "", false
}

// verbForReaction maps a reaction key back to a hug verb.
func verbForReaction(key string) (string, bool) {
	switch key {
	case reactAccept:
		return "accept", true
	case reactDecline:
		return "decline", true
	}
	return "", false
}
