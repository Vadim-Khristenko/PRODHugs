// Package richtext is a provider-agnostic rich-text model. A Doc is a flat
// list of Nodes; each renderer (Telegram, Matrix) turns a Doc into that
// provider's wire format. This is the single source of truth for message
// formatting — the "Rich Formatting" layer.
package richtext

type Kind int

const (
	KindPlain Kind = iota
	KindBold
	KindItalic
	KindUnderline
	KindStrike
	KindCode      // inline fixed-width
	KindSpoiler   // Telegram tg-spoiler; Matrix data-mx-spoiler
	KindLink      // Text=label, Href=url
	KindMention   // Text=label, UserID set; renders to native user link
	KindEmoji     // Text=fallback glyph, EmojiID set (Telegram custom emoji)
	KindLine      // hard line break
	KindQuote     // blockquote; Text is the quoted content (plain)
	KindCodeBlock // Text=code, Lang optional (stored in Href)
	KindHeading   // Text=heading, Level 1..6
)

type Node struct {
	Kind    Kind
	Text    string
	Href    string
	UserID  string
	EmojiID string
	Level   int
}

type Doc struct{ Nodes []Node }

type Builder struct{ nodes []Node }

func New() *Builder { return &Builder{} }

func (b *Builder) push(n Node) *Builder { b.nodes = append(b.nodes, n); return b }

func (b *Builder) Text(s string) *Builder      { return b.push(Node{Kind: KindPlain, Text: s}) }
func (b *Builder) Bold(s string) *Builder      { return b.push(Node{Kind: KindBold, Text: s}) }
func (b *Builder) Italic(s string) *Builder    { return b.push(Node{Kind: KindItalic, Text: s}) }
func (b *Builder) Underline(s string) *Builder { return b.push(Node{Kind: KindUnderline, Text: s}) }
func (b *Builder) Strike(s string) *Builder    { return b.push(Node{Kind: KindStrike, Text: s}) }
func (b *Builder) Code(s string) *Builder      { return b.push(Node{Kind: KindCode, Text: s}) }
func (b *Builder) Spoiler(s string) *Builder   { return b.push(Node{Kind: KindSpoiler, Text: s}) }
func (b *Builder) Line() *Builder              { return b.push(Node{Kind: KindLine}) }
func (b *Builder) Quote(s string) *Builder     { return b.push(Node{Kind: KindQuote, Text: s}) }

func (b *Builder) Link(label, href string) *Builder {
	return b.push(Node{Kind: KindLink, Text: label, Href: href})
}
func (b *Builder) Mention(label, userID string) *Builder {
	return b.push(Node{Kind: KindMention, Text: label, UserID: userID})
}
func (b *Builder) Emoji(fallback, emojiID string) *Builder {
	return b.push(Node{Kind: KindEmoji, Text: fallback, EmojiID: emojiID})
}
func (b *Builder) CodeBlock(code, lang string) *Builder {
	return b.push(Node{Kind: KindCodeBlock, Text: code, Href: lang})
}
func (b *Builder) Heading(level int, s string) *Builder {
	return b.push(Node{Kind: KindHeading, Text: s, Level: level})
}

func (b *Builder) Build() Doc { return Doc{Nodes: b.nodes} }
