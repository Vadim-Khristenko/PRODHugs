package telegram

import "strings"

// parseAction splits a callback token "<domain>.<verb>:<arg>" into its parts.
// Returns ok=false for anything that doesn't match that shape.
func parseAction(s string) (domain, verb, arg string, ok bool) {
	colon := strings.IndexByte(s, ':')
	if colon < 0 {
		return "", "", "", false
	}
	head := s[:colon]
	arg = s[colon+1:]
	dot := strings.IndexByte(head, '.')
	if dot < 0 {
		return "", "", "", false
	}
	domain = head[:dot]
	verb = head[dot+1:]
	if domain == "" || verb == "" || arg == "" {
		return "", "", "", false
	}
	return domain, verb, arg, true
}
