package richtext

import (
	"fmt"
	"html"
	"strings"
)

// RenderMatrixHTML renders a Doc to Matrix org.matrix.custom.html
// (formatted_body). Telegram-only nodes degrade: spoiler -> data-mx-spoiler,
// custom emoji -> its fallback glyph.
func RenderMatrixHTML(d Doc) string {
	esc := html.EscapeString
	var sb strings.Builder
	for _, n := range d.Nodes {
		switch n.Kind {
		case KindPlain:
			sb.WriteString(esc(n.Text))
		case KindBold:
			fmt.Fprintf(&sb, "<strong>%s</strong>", esc(n.Text))
		case KindItalic:
			fmt.Fprintf(&sb, "<em>%s</em>", esc(n.Text))
		case KindUnderline:
			fmt.Fprintf(&sb, "<u>%s</u>", esc(n.Text))
		case KindStrike:
			fmt.Fprintf(&sb, "<del>%s</del>", esc(n.Text))
		case KindCode:
			fmt.Fprintf(&sb, "<code>%s</code>", esc(n.Text))
		case KindSpoiler:
			fmt.Fprintf(&sb, "<span data-mx-spoiler>%s</span>", esc(n.Text))
		case KindLink:
			fmt.Fprintf(&sb, `<a href="%s">%s</a>`, esc(n.Href), esc(n.Text))
		case KindMention:
			fmt.Fprintf(&sb, `<a href="https://matrix.to/#/%s">%s</a>`, esc(n.UserID), esc(n.Text))
		case KindEmoji:
			sb.WriteString(esc(n.Text))
		case KindLine:
			sb.WriteString("<br>")
		case KindQuote:
			fmt.Fprintf(&sb, "<blockquote>%s</blockquote>", esc(n.Text))
		case KindCodeBlock:
			fmt.Fprintf(&sb, "<pre><code>%s</code></pre>", esc(n.Text))
		case KindHeading:
			lvl := n.Level
			if lvl < 1 || lvl > 6 {
				lvl = 3
			}
			fmt.Fprintf(&sb, "<h%d>%s</h%d>", lvl, esc(n.Text), lvl)
		}
	}
	return sb.String()
}

// RenderPlain renders a Doc to a plain-text fallback (Matrix body, and the
// classic-HTML-free fallback for Telegram if rich is unavailable).
func RenderPlain(d Doc) string {
	var sb strings.Builder
	for _, n := range d.Nodes {
		switch n.Kind {
		case KindLine:
			sb.WriteString("\n")
		case KindQuote:
			fmt.Fprintf(&sb, "> %s", n.Text)
		case KindCodeBlock:
			sb.WriteString(n.Text)
		default:
			sb.WriteString(n.Text)
		}
	}
	return sb.String()
}
