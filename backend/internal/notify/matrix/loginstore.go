package matrix

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"

	"github.com/google/uuid"
)

const loginTokenTTL = 5 * time.Minute

// LoginSessionStatus represents the state of a Matrix login session.
type LoginSessionStatus string

const (
	LoginSessionPending       LoginSessionStatus = "pending"
	LoginSessionAuthenticated LoginSessionStatus = "authenticated"
	LoginSessionFailed        LoginSessionStatus = "failed"
)

// loginSession represents a pending or completed Matrix login session.
type loginSession struct {
	BotToken   string
	Status     LoginSessionStatus
	UserID     uuid.UUID // set when authenticated
	FailReason string    // set when failed
	ExpiresAt  time.Time
}

// LoginStore manages Matrix login sessions. Thread-safe.
// Two tokens per session:
//   - botToken: embedded in the "/login <botToken>" command, consumed by the bot
//   - pollToken: returned to the frontend for polling the result
//
// Unlike the Telegram store there is no separate UserInfo: the Matrix identity
// is the sender's matrix user id, which the bot resolves directly.
type LoginStore struct {
	mu sync.Mutex
	// keyed by pollToken
	sessions map[string]*loginSession
	// maps botToken -> pollToken for bot-side lookup
	botIndex map[string]string
}

// NewLoginStore creates a new in-memory Matrix login session store.
func NewLoginStore() *LoginStore {
	return &LoginStore{
		sessions: make(map[string]*loginSession),
		botIndex: make(map[string]string),
	}
}

// CreateSession starts a new login session. Returns (botToken, pollToken, error).
// The botToken is embedded in the "/login <botToken>" command shown to the user.
// The pollToken is returned to the frontend for polling.
func (s *LoginStore) CreateSession() (botToken string, pollToken string, err error) {
	botToken, err = generateLoginToken()
	if err != nil {
		return "", "", err
	}
	pollToken, err = generateLoginToken()
	if err != nil {
		return "", "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[pollToken] = &loginSession{
		BotToken:  botToken,
		Status:    LoginSessionPending,
		ExpiresAt: time.Now().Add(loginTokenTTL),
	}
	s.botIndex[botToken] = pollToken

	return botToken, pollToken, nil
}

// ConsumeBotToken is called by the bot when it receives "/login <botToken>".
// Returns the pollToken if valid and not expired. One-shot: removes from botIndex.
func (s *LoginStore) ConsumeBotToken(botToken string) (pollToken string, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pollToken, exists := s.botIndex[botToken]
	if !exists {
		return "", false
	}
	delete(s.botIndex, botToken)

	session, exists := s.sessions[pollToken]
	if !exists || time.Now().After(session.ExpiresAt) {
		return "", false
	}

	return pollToken, true
}

// AuthenticateSession marks a login session as successfully authenticated.
// Called by the bot after resolving (or creating) the user.
func (s *LoginStore) AuthenticateSession(pollToken string, userID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[pollToken]
	if !exists {
		return
	}
	session.Status = LoginSessionAuthenticated
	session.UserID = userID
}

// FailSession marks a login session as failed with a reason.
func (s *LoginStore) FailSession(pollToken string, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[pollToken]
	if !exists {
		return
	}
	session.Status = LoginSessionFailed
	session.FailReason = reason
}

// PollResult holds the result of polling a login session.
type PollResult struct {
	Status     LoginSessionStatus
	UserID     uuid.UUID
	FailReason string
}

// PollSession checks the current state of a login session.
// Returns (result, found). If the session is terminal (authenticated or failed),
// it is deleted from the store.
func (s *LoginStore) PollSession(pollToken string) (PollResult, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[pollToken]
	if !exists {
		return PollResult{}, false
	}

	if time.Now().After(session.ExpiresAt) {
		delete(s.sessions, pollToken)
		return PollResult{}, false
	}

	result := PollResult{
		Status:     session.Status,
		UserID:     session.UserID,
		FailReason: session.FailReason,
	}

	// Clean up terminal sessions
	if session.Status == LoginSessionAuthenticated || session.Status == LoginSessionFailed {
		delete(s.sessions, pollToken)
	}

	return result, true
}

func generateLoginToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
