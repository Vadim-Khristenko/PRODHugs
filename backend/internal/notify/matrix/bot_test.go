package matrix

import "testing"

func TestParseLinkCommand(t *testing.T) {
	cases := []struct {
		in    string
		token string
		ok    bool
	}{
		{"link abc123", "abc123", true},
		{"LINK abc123", "abc123", true},
		{"  link   abc123  ", "abc123", true},
		{"link", "", false},
		{"link a b", "", false},
		{"hello abc123", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		token, ok := parseLinkCommand(c.in)
		if ok != c.ok || token != c.token {
			t.Fatalf("parseLinkCommand(%q) = (%q,%v), want (%q,%v)", c.in, token, ok, c.token, c.ok)
		}
	}
}
