package matrix

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"

	"github.com/google/uuid"
)

const linkTokenTTL = 5 * time.Minute

type linkEntry struct {
	UserID    uuid.UUID
	ExpiresAt time.Time
}

// LinkStore holds pending link tokens for Matrix account linking.
// Tokens expire after 5 minutes. Thread-safe.
type LinkStore struct {
	mu      sync.Mutex
	entries map[string]linkEntry
}

// NewLinkStore creates a new in-memory link token store.
func NewLinkStore() *LinkStore {
	return &LinkStore{
		entries: make(map[string]linkEntry),
	}
}

// GenerateToken creates a new opaque token that maps to the given userID.
// Returns a URL-safe base64 string (32 bytes of randomness).
func (s *LinkStore) GenerateToken(userID uuid.UUID) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(b)

	s.mu.Lock()
	s.entries[token] = linkEntry{
		UserID:    userID,
		ExpiresAt: time.Now().Add(linkTokenTTL),
	}
	s.mu.Unlock()

	return token, nil
}

// ConsumeToken looks up the token, returns the associated userID if valid
// and not expired, and deletes the entry (one-shot). Returns false if the
// token is unknown or expired.
func (s *LinkStore) ConsumeToken(token string) (uuid.UUID, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.entries[token]
	if !exists {
		return uuid.Nil, false
	}

	delete(s.entries, token)

	if time.Now().After(entry.ExpiresAt) {
		return uuid.Nil, false
	}

	return entry.UserID, true
}
