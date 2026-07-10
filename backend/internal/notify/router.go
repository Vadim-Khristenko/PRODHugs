package notify

import (
	"context"
	"log/slog"
	"strconv"

	"go-service-template/internal/notify/richtext"

	"github.com/google/uuid"
)

// New re-exports a richtext builder so notify consumers build bodies without
// a second import. Body := notify.New().Bold(name).Build().
func New() *Builder { return &Builder{b: richtext.New()} }

// Builder is a thin pass-through over richtext.Builder for the subset of
// nodes notify consumers use when composing messages.
type Builder struct{ b *richtext.Builder }

func (b *Builder) Text(s string) *Builder             { b.b.Text(s); return b }
func (b *Builder) Bold(s string) *Builder             { b.b.Bold(s); return b }
func (b *Builder) Italic(s string) *Builder           { b.b.Italic(s); return b }
func (b *Builder) Underline(s string) *Builder        { b.b.Underline(s); return b }
func (b *Builder) Strike(s string) *Builder           { b.b.Strike(s); return b }
func (b *Builder) Code(s string) *Builder             { b.b.Code(s); return b }
func (b *Builder) Spoiler(s string) *Builder          { b.b.Spoiler(s); return b }
func (b *Builder) Line() *Builder                     { b.b.Line(); return b }
func (b *Builder) Quote(s string) *Builder            { b.b.Quote(s); return b }
func (b *Builder) Link(l, h string) *Builder          { b.b.Link(l, h); return b }
func (b *Builder) Mention(l, id string) *Builder      { b.b.Mention(l, id); return b }
func (b *Builder) CodeBlock(c, lang string) *Builder  { b.b.CodeBlock(c, lang); return b }
func (b *Builder) Heading(lvl int, s string) *Builder { b.b.Heading(lvl, s); return b }
func (b *Builder) Build() richtext.Doc                { return b.b.Build() }

// Render helpers expose the richtext renderers to providers.
func RenderTelegram(d richtext.Doc) string   { return richtext.RenderTelegramHTML(d) }
func RenderMatrixHTML(d richtext.Doc) string { return richtext.RenderMatrixHTML(d) }
func RenderPlain(d richtext.Doc) string      { return richtext.RenderPlain(d) }

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

// Dispatch renders msg and delivers it to every enabled provider the user
// has linked, recording each SentRef under (kind, eventID). Fire-and-forget:
// errors are logged, not returned.
func (r *Router) Dispatch(ctx context.Context, userID uuid.UUID, kind string, eventID uuid.UUID, msg Message) {
	a, err := r.addr.ResolveAddress(ctx, userID)
	if err != nil {
		r.logger.Error("notify: resolve address failed", "user_id", userID, "error", err)
		return
	}
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
				if mErr := r.blocks.MarkTelegramBlocked(ctx, userID); mErr != nil {
					r.logger.Error("notify: mark blocked failed", "user_id", userID, "error", mErr)
				}
				continue
			}
			r.logger.Error("notify: send failed", "provider", p.Name(), "user_id", userID, "error", err)
			continue
		}
		if err := r.refs.SaveRef(ctx, kind, eventID, ref); err != nil {
			r.logger.Error("notify: save ref failed", "provider", p.Name(), "error", err)
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
