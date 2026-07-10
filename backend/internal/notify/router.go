package notify

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/google/uuid"
)

// Router renders and delivers Messages across enabled providers.
type Router struct {
	providers []Provider
	addr      AddressResolver
	refs      RefStore
	blocks    BlockMarker
	logger    *slog.Logger
}

func NewRouter(providers []Provider, addr AddressResolver, refs RefStore, blocks BlockMarker, logger *slog.Logger) *Router {
	if logger == nil {
		logger = slog.Default()
	}
	return &Router{providers: providers, addr: addr, refs: refs, blocks: blocks, logger: logger}
}

// chatRefFor returns the provider-specific chat reference for a user, and
// whether the user has that channel linked.
func chatRefFor(name string, a Address) (string, bool) {
	switch name {
	case "telegram":
		if a.TelegramID == nil {
			return "", false
		}
		return strconv.FormatInt(*a.TelegramID, 10), true
	case "matrix":
		if a.MatrixID == nil {
			return "", false
		}
		return *a.MatrixID, true
	}
	return "", false
}

// deliver sends msg to every enabled provider the user has linked and returns
// the resulting SentRefs. A blocked recipient is marked; other send errors are
// logged. Refs with an empty MessageRef (message delivered but not
// addressable for a later edit) are dropped so the store never holds a
// non-editable ref. Fire-and-forget: nothing is returned to the caller.
func (r *Router) deliver(ctx context.Context, userID uuid.UUID, msg Message) []SentRef {
	a, err := r.addr.ResolveAddress(ctx, userID)
	if err != nil {
		r.logger.Error("notify: resolve address failed", "user_id", userID, "error", err)
		return nil
	}
	var out []SentRef
	for _, p := range r.providers {
		if !p.Enabled() {
			continue
		}
		chatRef, ok := chatRefFor(p.Name(), a)
		if !ok {
			continue
		}
		ref, err := p.Send(ctx, chatRef, msg)
		if err != nil {
			if IsBlocked(err) {
				// The blocked signal is Telegram-specific today; only the
				// Telegram provider maps to the telegram_blocked_at flag.
				if p.Name() == "telegram" {
					if mErr := r.blocks.MarkTelegramBlocked(ctx, userID); mErr != nil {
						r.logger.Error("notify: mark blocked failed", "user_id", userID, "error", mErr)
					}
				}
				continue
			}
			r.logger.Error("notify: send failed", "provider", p.Name(), "user_id", userID, "error", err)
			continue
		}
		if ref.MessageRef == "" {
			continue
		}
		out = append(out, ref)
	}
	return out
}

// Notify delivers a terminal message to the user without recording a ref.
// Use for notifications that are never edited later (completion, decline,
// cancellation notices).
func (r *Router) Notify(ctx context.Context, userID uuid.UUID, msg Message) {
	r.deliver(ctx, userID, msg)
}

// Dispatch delivers msg and records each SentRef under (kind, eventID) so the
// message can be edited later via EditByEvent. Use for messages whose
// originating event can change state (e.g. a hug suggestion).
func (r *Router) Dispatch(ctx context.Context, userID uuid.UUID, kind string, eventID uuid.UUID, msg Message) {
	for _, ref := range r.deliver(ctx, userID, msg) {
		if err := r.refs.SaveRef(ctx, kind, eventID, ref); err != nil {
			r.logger.Error("notify: save ref failed", "provider", ref.Provider, "error", err)
		}
	}
}

// EditByEvent edits every recorded message for (kind, eventID).
func (r *Router) EditByEvent(ctx context.Context, kind string, eventID uuid.UUID, msg Message) {
	refs, err := r.refs.GetRefs(ctx, kind, eventID)
	if err != nil {
		r.logger.Error("notify: get refs failed", "kind", kind, "event_id", eventID, "error", err)
		return
	}
	byName := map[string]Provider{}
	for _, p := range r.providers {
		byName[p.Name()] = p
	}
	for _, ref := range refs {
		p, ok := byName[ref.Provider]
		if !ok || !p.Enabled() {
			continue
		}
		if err := p.Edit(ctx, ref, msg); err != nil {
			r.logger.Warn("notify: edit failed", "provider", ref.Provider, "error", err)
		}
	}
}
