package notify

import "go-service-template/internal/notify/richtext"

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
