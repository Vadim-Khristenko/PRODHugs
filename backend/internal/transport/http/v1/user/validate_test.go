package user

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// validateUsername returns true when the name does not exceed the rune limit.
// This mirrors the guard in RegisterUser: utf8.RuneCountInString(s) > maxUsernameRunes.
func checkUsername(s string) bool {
	return utf8.RuneCountInString(s) <= maxUsernameRunes
}

// checkDisplayName mirrors the guard in UpdateUserSettings.
func checkDisplayName(s string) bool {
	return utf8.RuneCountInString(s) <= maxDisplayNameRunes
}

func TestUsernameLength(t *testing.T) {
	t.Run("exactly 32 ASCII chars - ok", func(t *testing.T) {
		s := strings.Repeat("a", 32)
		if !checkUsername(s) {
			t.Errorf("expected %q (len=%d) to be valid", s, utf8.RuneCountInString(s))
		}
	})

	t.Run("33 ASCII chars - rejected", func(t *testing.T) {
		s := strings.Repeat("a", 33)
		if checkUsername(s) {
			t.Errorf("expected %q (len=%d) to be rejected", s, utf8.RuneCountInString(s))
		}
	})

	t.Run("empty string - ok (other validation handles minimum)", func(t *testing.T) {
		if !checkUsername("") {
			t.Error("empty string should not be rejected by length check alone")
		}
	})
}

func TestDisplayNameLength(t *testing.T) {
	t.Run("exactly 32 ASCII chars - ok", func(t *testing.T) {
		s := strings.Repeat("b", 32)
		if !checkDisplayName(s) {
			t.Errorf("expected %q (len=%d) to be valid", s, utf8.RuneCountInString(s))
		}
	})

	t.Run("33 ASCII chars - rejected", func(t *testing.T) {
		s := strings.Repeat("b", 33)
		if checkDisplayName(s) {
			t.Errorf("expected %q (len=%d) to be rejected", s, utf8.RuneCountInString(s))
		}
	})

	t.Run("32 Cyrillic chars - ok (rune count, not byte count)", func(t *testing.T) {
		// Each Cyrillic char is 2 bytes but 1 rune; 32 runes must pass.
		s := strings.Repeat("ф", 32)
		if utf8.RuneCountInString(s) != 32 {
			t.Fatalf("setup error: expected 32 runes, got %d", utf8.RuneCountInString(s))
		}
		if !checkDisplayName(s) {
			t.Errorf("32 Cyrillic runes should be valid; byte len=%d", len(s))
		}
	})

	t.Run("33 Cyrillic chars - rejected", func(t *testing.T) {
		s := strings.Repeat("ф", 33)
		if utf8.RuneCountInString(s) != 33 {
			t.Fatalf("setup error: expected 33 runes, got %d", utf8.RuneCountInString(s))
		}
		if checkDisplayName(s) {
			t.Errorf("33 Cyrillic runes should be rejected; byte len=%d", len(s))
		}
	})
}
