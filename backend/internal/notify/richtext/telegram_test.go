package richtext

import "testing"

func TestRenderTelegramHTML(t *testing.T) {
	doc := New().Bold("Аня").Text(" обняла ").Italic("тебя").Line().
		Spoiler("сюрприз").Build()
	got := RenderTelegramHTML(doc)
	want := "<b>Аня</b> обняла <i>тебя</i>\n<tg-spoiler>сюрприз</tg-spoiler>"
	if got != want {
		t.Fatalf("\n got: %q\nwant: %q", got, want)
	}
}

func TestTelegramEscapesHostileNames(t *testing.T) {
	doc := New().Bold(`<b>x</b> & "q"`).Build()
	got := RenderTelegramHTML(doc)
	want := `<b>&lt;b&gt;x&lt;/b&gt; &amp; &quot;q&quot;</b>`
	if got != want {
		t.Fatalf("\n got: %q\nwant: %q", got, want)
	}
}

func TestTelegramMention(t *testing.T) {
	doc := New().Mention("Аня", "123456").Build()
	got := RenderTelegramHTML(doc)
	want := `<a href="tg://user?id=123456">Аня</a>`
	if got != want {
		t.Fatalf("\n got: %q\nwant: %q", got, want)
	}
}
