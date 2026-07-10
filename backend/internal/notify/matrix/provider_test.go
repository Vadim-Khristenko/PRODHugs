package matrix

import (
	"strings"
	"testing"

	"go-service-template/internal/notify"
)

func TestEnabledRequiresAllThree(t *testing.T) {
	cases := []struct {
		hs, uid, tok string
		want         bool
	}{
		{"https://m.org", "@bot:m.org", "tok", true},
		{"", "@bot:m.org", "tok", false},
		{"https://m.org", "", "tok", false},
		{"https://m.org", "@bot:m.org", "", false},
	}
	for _, c := range cases {
		p := New(c.hs, c.uid, c.tok, nil)
		if p.Enabled() != c.want {
			t.Fatalf("Enabled(%q,%q,%q)=%v want %v", c.hs, c.uid, c.tok, p.Enabled(), c.want)
		}
	}
}

func TestBuildContentBodyOnly(t *testing.T) {
	msg := notify.Message{
		Body: notify.New().Bold("Аня").Text(" хочет обнять").Build(),
		Buttons: [][]notify.Button{{
			{Label: "Обнять 🤗", Action: "hug.accept:x"},
			{Label: "Отклонить", Action: "hug.decline:x"},
		}},
	}
	c := buildContent(msg)
	if c["msgtype"] != "m.notice" {
		t.Fatalf("msgtype = %v", c["msgtype"])
	}
	html := c["formatted_body"].(string)
	if !strings.Contains(html, "<strong>Аня</strong>") {
		t.Fatalf("formatted_body missing bold: %q", html)
	}
	// Buttons are placed as m.reaction annotations by Send, not degraded into
	// the body — so the labels must NOT appear in the rendered content.
	if strings.Contains(html, "Обнять 🤗") || strings.Contains(html, "Отклонить") {
		t.Fatalf("button labels leaked into formatted_body: %q", html)
	}
	body := c["body"].(string)
	if strings.Contains(body, "Обнять 🤗") || strings.Contains(body, "Отклонить") {
		t.Fatalf("button labels leaked into plain body: %q", body)
	}
}

func TestNextTxnMonotonic(t *testing.T) {
	p := New("https://m.org", "@bot:m.org", "tok", nil)
	a, b := p.nextTxn(), p.nextTxn()
	if a == b {
		t.Fatalf("txn ids not unique: %q == %q", a, b)
	}
}
