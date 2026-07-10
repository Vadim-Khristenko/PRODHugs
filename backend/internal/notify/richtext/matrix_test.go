package richtext

import "testing"

func TestRenderMatrixHTML(t *testing.T) {
	doc := New().Bold("Аня").Text(" обняла ").Italic("тебя").Line().
		Spoiler("сюрприз").Build()
	got := RenderMatrixHTML(doc)
	want := `<strong>Аня</strong> обняла <em>тебя</em><br><span data-mx-spoiler>сюрприз</span>`
	if got != want {
		t.Fatalf("\n got: %q\nwant: %q", got, want)
	}
}

func TestRenderPlain(t *testing.T) {
	doc := New().Bold("Аня").Text(" обняла ").Italic("тебя").Line().
		Quote("привет").Build()
	got := RenderPlain(doc)
	want := "Аня обняла тебя\n> привет"
	if got != want {
		t.Fatalf("\n got: %q\nwant: %q", got, want)
	}
}
