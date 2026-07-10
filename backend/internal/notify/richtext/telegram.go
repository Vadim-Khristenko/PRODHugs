package richtext

import (
	"fmt"
	"strings"
)

// tgEscape neutralises user text for Telegram's HTML/Rich parser. Telegram
// only accepts a small set of NAMED entities; we stick to the always-safe
// four, which fully cover the HTML-significant characters.
func tgEscape(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
	)
	return r.Replace(s)
}

// RenderTelegramHTML renders a Doc to Telegram Rich HTML (the body of
// InputRichMessage.html). See the new sendRichMessage tag set.
func RenderTelegramHTML(d Doc) string {
	var sb strings.Builder
	for _, n := range d.Nodes {
		switch n.Kind {
		case KindPlain:
			sb.WriteString(tgEscape(n.Text))
		case KindBold:
			fmt.Fprintf(&sb, "<b>%s</b>", tgEscape(n.Text))
		case KindItalic:
			fmt.Fprintf(&sb, "<i>%s</i>", tgEscape(n.Text))
		case KindUnderline:
			fmt.Fprintf(&sb, "<u>%s</u>", tgEscape(n.Text))
		case KindStrike:
			fmt.Fprintf(&sb, "<s>%s</s>", tgEscape(n.Text))
		case KindCode:
			fmt.Fprintf(&sb, "<code>%s</code>", tgEscape(n.Text))
		case KindSpoiler:
			fmt.Fprintf(&sb, "<tg-spoiler>%s</tg-spoiler>", tgEscape(n.Text))
		case KindLink:
			fmt.Fprintf(&sb, `<a href="%s">%s</a>`, tgEscape(n.Href), tgEscape(n.Text))
		case KindMention:
			fmt.Fprintf(&sb, `<a href="tg://user?id=%s">%s</a>`, tgEscape(n.UserID), tgEscape(n.Text))
		case KindEmoji:
			fmt.Fprintf(&sb, `<tg-emoji emoji-id="%s">%s</tg-emoji>`, tgEscape(n.EmojiID), tgEscape(n.Text))
		case KindLine:
			sb.WriteString("\n")
		case KindQuote:
			fmt.Fprintf(&sb, "<blockquote>%s</blockquote>", tgEscape(n.Text))
		case KindCodeBlock:
			if n.Href != "" {
				fmt.Fprintf(&sb, `<pre><code class="language-%s">%s</code></pre>`, tgEscape(n.Href), tgEscape(n.Text))
			} else {
				fmt.Fprintf(&sb, "<pre>%s</pre>", tgEscape(n.Text))
			}
		case KindHeading:
			lvl := n.Level
			if lvl < 1 || lvl > 6 {
				lvl = 3
			}
			fmt.Fprintf(&sb, "<h%d>%s</h%d>", lvl, tgEscape(n.Text), lvl)
		}
	}
	return sb.String()
}
