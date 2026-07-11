package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go-service-template/internal/errorz"
	"go-service-template/internal/models"
	"go-service-template/pkg/crypto"
)

// InitMatrixLogin creates a new Matrix login session and returns the command the
// user must send to the bot, the configured bot user id, and the poll token.
// Returns ErrMatrixLoginUnavailable if Matrix login is not configured.
func (s *service) InitMatrixLogin() (command, botUserID, pollToken string, err error) {
	if s.matrixLoginStore == nil || s.matrixLoginBotUserID == "" {
		return "", "", "", errorz.ErrMatrixLoginUnavailable
	}

	botToken, pollToken, err := s.matrixLoginStore.CreateSession()
	if err != nil {
		return "", "", "", fmt.Errorf("create matrix login session: %w", err)
	}

	command = "!login " + botToken
	return command, s.matrixLoginBotUserID, pollToken, nil
}

// LoginViaMatrix authenticates an existing user by Matrix ID or auto-registers a
// new one. Returns the user (without tokens — the caller mints tokens separately).
func (s *service) LoginViaMatrix(ctx context.Context, matrixID, roomID string) (*models.User, error) {
	// Try to find an existing user with this Matrix ID
	u, err := s.repo.GetByMatrixID(ctx, matrixID)
	if err == nil {
		if u.BannedAt != nil {
			return nil, errorz.ErrUserBanned
		}
		return u, nil
	}
	if !errors.Is(err, errorz.ErrUserNotFound) {
		return nil, fmt.Errorf("lookup by matrix ID: %w", err)
	}

	// No existing user — auto-register from the matrix localpart.
	base := sanitizeUsername(matrixLocalpart(matrixID))
	var username string
	if base != "" {
		username, err = s.findAvailableUsername(ctx, base)
		if err != nil {
			return nil, fmt.Errorf("generate username: %w", err)
		}
	} else {
		suffix, err := randomHex(4)
		if err != nil {
			return nil, fmt.Errorf("generate username: %w", err)
		}
		username = "user_" + suffix
	}

	randomPassword, err := generateRandomPassword()
	if err != nil {
		return nil, fmt.Errorf("generate random password: %w", err)
	}

	hash, err := crypto.GenerateHash(randomPassword)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	input := &models.CreateUser{
		Username:       username,
		Password:       randomPassword,
		HashedPassword: hash,
		Role:           "user",
	}

	u, err = s.repo.Create(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	if err := s.repo.SetMatrixLink(ctx, u.ID, matrixID, roomID); err != nil {
		return nil, fmt.Errorf("set matrix link: %w", err)
	}
	u.MatrixID = &matrixID

	return u, nil
}

// matrixLocalpart extracts the localpart from a Matrix user id of the form
// "@name:server". Returns the input unchanged if it is not in that form.
func matrixLocalpart(matrixID string) string {
	s := strings.TrimPrefix(matrixID, "@")
	if i := strings.IndexByte(s, ':'); i >= 0 {
		s = s[:i]
	}
	return s
}
