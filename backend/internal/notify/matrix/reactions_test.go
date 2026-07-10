package matrix

import "testing"

func TestReactionForActionRoundTrip(t *testing.T) {
	cases := []struct {
		action string
		key    string
		verb   string
	}{
		{"hug.accept:1234", reactAccept, "accept"},
		{"hug.decline:abcd", reactDecline, "decline"},
	}
	for _, c := range cases {
		key, ok := reactionForAction(c.action)
		if !ok || key != c.key {
			t.Fatalf("reactionForAction(%q) = (%q,%v), want (%q,true)", c.action, key, ok, c.key)
		}
		verb, ok := verbForReaction(key)
		if !ok || verb != c.verb {
			t.Fatalf("verbForReaction(%q) = (%q,%v), want (%q,true)", key, verb, ok, c.verb)
		}
	}
}

func TestReactionForActionUnmapped(t *testing.T) {
	if _, ok := reactionForAction("hug.pick:bear"); ok {
		t.Fatal("reactionForAction(hug.pick:...) should not map")
	}
	if _, ok := verbForReaction("👍"); ok {
		t.Fatal("verbForReaction(arbitrary) should not map")
	}
}
