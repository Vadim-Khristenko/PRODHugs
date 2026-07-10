package telegram

import "testing"

func TestParseAction(t *testing.T) {
	domain, verb, arg, ok := parseAction("hug.accept:1a2b")
	if !ok || domain != "hug" || verb != "accept" || arg != "1a2b" {
		t.Fatalf("got %q/%q/%q ok=%v", domain, verb, arg, ok)
	}
	if _, _, _, ok := parseAction("garbage"); ok {
		t.Fatal("garbage should not parse")
	}
	if _, _, _, ok := parseAction("hug.accept:"); ok {
		t.Fatal("empty arg should not parse")
	}
}
