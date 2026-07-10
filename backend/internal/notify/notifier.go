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
	b := New().Text("🤗 ").Bold(name).Text(" " + hugTypeSuggestionPhrase(hugType) + "!")
	if comment != nil && *comment != "" {
		b.Line().Line().Text("💬 ").Italic(*comment)
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

	// Giver message: "🎉 <b>receiver</b> <verb> <hugWord>! <coins> [plural]"
	gb := New().Text("🎉 ").Bold(displayName(receiver)).
		Text(" " + receiverVerb + " " + hugWord + "! " + giverCoinText)
	if comment == nil {
		gb.Text(" " + pluralObnimani(int(totalCoins)))
	}
	// Receiver message: "🎉 Вы обнялись с <b>giver</b>! <coins> <plural>"
	rb := New().Text("🎉 Вы обнялись с ").Bold(displayName(giver)).
		Text("! " + coinText + " " + pluralObnimani(int(totalCoins)))
	if comment != nil && *comment != "" {
		rb.Line().Line().Text("💬 ").Italic(*comment)
	}

	n.r.Dispatch(ctx, giverID, "hug_completed", hugID, Message{Body: gb.Build()})
	n.r.Dispatch(ctx, receiverID, "hug_completed", hugID, Message{Body: rb.Build()})
}

// NotifyHugDeclined notifies the giver that their hug was declined.
func (n *Notifier) NotifyHugDeclined(ctx context.Context, giverID, receiverID, hugID uuid.UUID) {
	receiver, err := n.users.GetByID(ctx, receiverID)
	if err != nil {
		n.logger.Error("notify: look up receiver failed", "receiver_id", receiverID, "error", err)
		return
	}
	verb := genderVerb(receiver.Gender, "отклонил", "отклонила", "отклонил(а)")
	body := New().Text("😔 ").Bold(displayName(receiver)).Text(" " + verb + " объятие").Build()
	n.r.Dispatch(ctx, giverID, "hug_declined", hugID, Message{Body: body})
}

// NotifyHugCancelled notifies the receiver that the request was cancelled.
func (n *Notifier) NotifyHugCancelled(ctx context.Context, receiverID, giverID, hugID uuid.UUID) {
	giver, err := n.users.GetByID(ctx, giverID)
	if err != nil {
		n.logger.Error("notify: look up giver failed", "giver_id", giverID, "error", err)
		return
	}
	verb := genderVerb(giver.Gender, "отменил", "отменила", "отменил(а)")
	body := New().Text("❌ ").Bold(displayName(giver)).Text(" " + verb + " запрос на объятие").Build()
	n.r.Dispatch(ctx, receiverID, "hug_cancelled", hugID, Message{Body: body})
}
