package user

import (
	"context"
	"fmt"
	"time"

	"go-service-template/internal/errorz"
	"go-service-template/internal/models"
	"go-service-template/pkg/crypto"

	"github.com/google/uuid"
)

const tagChangeCost = 5

func (s *service) UpdateSettings(ctx context.Context, id uuid.UUID, gender *string, displayName *string, tag *string) (*models.User, error) {
	// Setting a non-empty tag that differs from the current one costs coins.
	// Clearing (removing) a tag is free.
	if tag != nil && *tag != "" {
		currentUser, err := s.repo.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}

		// Only charge if the new tag is different from the current one.
		if currentUser.Tag == nil || *currentUser.Tag != *tag {
			var result *models.User
			err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
				bal, err := s.balanceRepo.DeductBalance(txCtx, id, tagChangeCost)
				if err != nil {
					return err
				}
				if bal == nil {
					return errorz.ErrInsufficientBalance
				}
				result, err = s.repo.UpdateSettings(txCtx, id, gender, displayName, tag)
				return err
			})
			if err != nil {
				return nil, err
			}
			return result, nil
		}
	}

	// No cost: tag is nil (not provided), empty (clearing), or unchanged.
	return s.repo.UpdateSettings(ctx, id, gender, displayName, tag)
}

func (s *service) GetTelegramID(ctx context.Context, userID uuid.UUID) (*int64, error) {
	return s.repo.GetTelegramID(ctx, userID)
}

// GenerateLinkToken creates a deep-link token for Telegram account linking.
// Returns the token and the full bot deep-link URL.
func (s *service) GenerateLinkToken(ctx context.Context, userID uuid.UUID) (string, string, error) {
	if s.telegramLinkStore == nil || s.botUsername == "" {
		return "", "", fmt.Errorf("telegram linking not configured")
	}

	token, err := s.telegramLinkStore.GenerateToken(userID)
	if err != nil {
		return "", "", fmt.Errorf("generate link token: %w", err)
	}

	botURL := fmt.Sprintf("https://t.me/%s?start=%s", s.botUsername, token)
	return token, botURL, nil
}

// UnlinkTelegram removes the Telegram ID from the user.
func (s *service) UnlinkTelegram(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	return s.repo.ClearTelegramID(ctx, userID)
}

// GenerateMatrixLinkToken creates a token for Matrix account linking.
// Returns the token, the command the user must send to the bot, and the
// configured bot user id.
func (s *service) GenerateMatrixLinkToken(ctx context.Context, userID uuid.UUID) (string, string, string, error) {
	if s.matrixLinkStore == nil || s.matrixBotUserID == "" {
		return "", "", "", errorz.ErrMatrixLinkingUnavailable
	}

	token, err := s.matrixLinkStore.GenerateToken(userID)
	if err != nil {
		return "", "", "", fmt.Errorf("generate matrix link token: %w", err)
	}

	command := "link " + token
	return token, command, s.matrixBotUserID, nil
}

// UnlinkMatrix removes the Matrix link from the user and returns the refreshed user.
func (s *service) UnlinkMatrix(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	if err := s.repo.ClearMatrixLink(ctx, userID); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, userID)
}

func (s *service) ChangePassword(ctx context.Context, id uuid.UUID, oldPassword, newPassword string) error {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	ok, err := crypto.ComparePasswordAndHash(oldPassword, u.HashedPassword)
	if err != nil {
		return fmt.Errorf("failed to compare password: %w", err)
	}
	if !ok {
		return errorz.ErrWrongPassword
	}

	hash, err := crypto.GenerateHash(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	return s.repo.UpdatePassword(ctx, id, hash)
}

func (s *service) SaveRefreshToken(ctx context.Context, jti string, userID uuid.UUID, expiresAtUnix int64) error {
	return s.refreshTokenRepo.SaveRefreshToken(ctx, jti, userID, expiresAtUnix)
}

func (s *service) IsRefreshTokenActive(ctx context.Context, jti string) (bool, error) {
	return s.refreshTokenRepo.IsRefreshTokenActive(ctx, jti)
}

func (s *service) RevokeRefreshToken(ctx context.Context, jti string) error {
	return s.refreshTokenRepo.RevokeRefreshToken(ctx, jti)
}

func (s *service) RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	return s.refreshTokenRepo.RevokeAllUserRefreshTokens(ctx, userID)
}

func (s *service) PromoteUser(ctx context.Context, id uuid.UUID, bid int32, message *string) (*models.User, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	var result *models.User
	err = s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		now := time.Now()
		isAlreadyPromoted := u.PromotedUntil != nil && u.PromotedUntil.After(now)

		var promotedUntil time.Time
		farFuture := now.Add(100 * 365 * 24 * time.Hour)

		if isAlreadyPromoted {
			// If already in top, pay the difference and keep the same timer
			cost := bid - u.PromotionBid
			if cost < 0 {
				cost = 0 // Allow updating message without paying more, but don't refund
			}
			promotedUntil = *u.PromotedUntil

			if cost > 0 {
				bal, err := s.balanceRepo.DeductBalance(txCtx, id, cost)
				if err != nil {
					return err
				}
				if bal == nil {
					return errorz.ErrInsufficientBalance
				}
			}
		} else {
			// First time or expired, pay full price and get 24h
			cost := bid
			promotedUntil = farFuture

			if cost > 0 {
				bal, err := s.balanceRepo.DeductBalance(txCtx, id, cost)
				if err != nil {
					return err
				}
				if bal == nil {
					return errorz.ErrInsufficientBalance
				}
			}
		}

		result, err = s.repo.PromoteUser(txCtx, id, promotedUntil, message, bid)
		return err
	})

	if err != nil {
		return nil, err
	}
	if s.onPromotionUpdated != nil {
		s.onPromotionUpdated()
	}
	return result, nil
}
