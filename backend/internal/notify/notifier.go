package notify

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

// Notifier is the semantic facade the domain calls for hug events. It
// resolves display names / gender, builds provider-agnostic messages, and
// dispatches them through the Router. Fire-and-forget: errors are logged.
type Notifier struct {
	r      *Router
	users  UserLookup
	logger *slog.Logger
}

func NewNotifier(r *Router, users UserLookup, logger *slog.Logger) *Notifier {
	if logger == nil {
		logger = slog.Default()
	}
	return &Notifier{r: r, users: users, logger: logger}
}

// NotifyHugSuggestion notifies the receiver with Accept/Decline buttons.
func (n *Notifier) NotifyHugSuggestion(ctx context.Context, receiverID, hugID, giverID uuid.UUID, hugType string, comment *string) {
	giver, err := n.users.GetByID(ctx, giverID)
	if err != nil {
		n.logger.Error("notify: look up giver failed", "giver_id", giverID, "error", err)
		return
	}
	name := displayName(giver)
	b := New().Heading(1, "🤗 Новое объятие").
		Bold(name).Text(" " + hugTypeSuggestionPhrase(hugType) + "!")
	if comment != nil && *comment != "" {
		b.Line().Quote("💬 " + *comment)
	}
	msg := Message{
		Body: b.Build(),
		Buttons: [][]Button{{
			{Label: "Обнять 🤗", Action: "hug.accept:" + hugID.String()},
			{Label: "Отклонить", Action: "hug.decline:" + hugID.String()},
		}},
	}
	n.r.Dispatch(ctx, receiverID, "hug_suggestion", hugID, msg)
}

// NotifyHugCompleted notifies both participants that the hug was accepted.
func (n *Notifier) NotifyHugCompleted(ctx context.Context, giverID, receiverID, hugID uuid.UUID, hugType string, bonusCoins int32, comment *string) {
	giver, err := n.users.GetByID(ctx, giverID)
	if err != nil {
		n.logger.Error("notify: look up giver failed", "giver_id", giverID, "error", err)
		return
	}
	receiver, err := n.users.GetByID(ctx, receiverID)
	if err != nil {
		n.logger.Error("notify: look up receiver failed", "receiver_id", receiverID, "error", err)
		return
	}

	totalCoins := 1 + bonusCoins
	coinText := fmt.Sprintf("+%d", totalCoins)
	if bonusCoins > 0 {
		coinText = fmt.Sprintf("+%d (бонус +%d)", totalCoins, bonusCoins)
	}
	hugWord := hugTypeCompletedNoun(hugType)
	giverCoinText := coinText
	if comment != nil {
		giverCoinText = "0 (оплата комментария)"
	}
	receiverVerb := genderVerb(receiver.Gender, "принял", "приняла", "принял(а)")

	// Giver message: heading + "<b>receiver</b> <verb> <hugWord>! <coins> [plural]"
	gb := New().Heading(1, "🎉 Обнялись!").
		Bold(displayName(receiver)).
		Text(" " + receiverVerb + " " + hugWord + "! ").Bold(giverCoinText)
	if comment == nil {
		gb.Text(" " + pluralObnimani(int(totalCoins)))
	}
	// Receiver message: heading + "Вы обнялись с <b>giver</b>! <coins> <plural>"
	rb := New().Heading(1, "🎉 Обнялись!").
		Text("Вы обнялись с ").Bold(displayName(giver)).
		Text("! ").Bold(coinText).Text(" " + pluralObnimani(int(totalCoins)))
	if comment != nil && *comment != "" {
		rb.Line().Quote("💬 " + *comment)
	}

	// Completion notices are terminal — never edited — so they don't record a
	// ref (both go to distinct users under the same hug id, which would
	// otherwise collide on the ref key).
	n.r.Notify(ctx, giverID, Message{Body: gb.Build()})
	n.r.Notify(ctx, receiverID, Message{Body: rb.Build()})
	// Update the original suggestion (shown to the receiver) to reflect that it
	// was accepted, dropping the Accept/Decline buttons.
	n.r.EditByEvent(ctx, "hug_suggestion", hugID, Message{Body: New().Text("🤗 Обнимашка принята ✅").Build()})
}

// NotifyHugDeclined notifies the giver that their hug was declined.
func (n *Notifier) NotifyHugDeclined(ctx context.Context, giverID, receiverID, hugID uuid.UUID) {
	receiver, err := n.users.GetByID(ctx, receiverID)
	if err != nil {
		n.logger.Error("notify: look up receiver failed", "receiver_id", receiverID, "error", err)
		return
	}
	verb := genderVerb(receiver.Gender, "отклонил", "отклонила", "отклонил(а)")
	body := New().Heading(1, "😔 Объятие отклонено").
		Bold(displayName(receiver)).Text(" " + verb + " объятие.").Build()
	n.r.Notify(ctx, giverID, Message{Body: body})
	n.r.EditByEvent(ctx, "hug_suggestion", hugID, Message{Body: New().Text("🤗 Обнимашка отклонена ❌").Build()})
}

// NotifyHugCancelled updates the receiver's suggestion message to reflect that
// the giver cancelled the request, dropping the Accept/Decline buttons. There
// is no separate push — the edited message is the notification (matching the
// legacy behavior, which never pushed a cancel notice).
func (n *Notifier) NotifyHugCancelled(ctx context.Context, hugID uuid.UUID) {
	n.r.EditByEvent(ctx, "hug_suggestion", hugID, Message{Body: New().Text("🤗 Запрос на обнимашку отменён ❌").Build()})
}

// NotifyDailyReminder pings a user to claim their daily reward.
func (n *Notifier) NotifyDailyReminder(ctx context.Context, userID uuid.UUID) {
	body := New().Heading(1, "🎁 Ежедневная награда").
		Text("Не забудьте забрать ежедневную награду сегодня — загляните в приложение!").Build()
	n.r.Notify(ctx, userID, Message{Body: body})
}
