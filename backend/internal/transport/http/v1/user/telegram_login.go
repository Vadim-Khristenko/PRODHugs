package user

import (
	"context"
	"fmt"

	"go-service-template/internal/telegram"
	v1 "go-service-template/internal/transport/http/v1"
)

// InitTelegramLogin creates a new Telegram login session and returns a bot deep-link URL.
func (h *UserHandler) InitTelegramLogin(ctx context.Context, req v1.InitTelegramLoginRequestObject) (v1.InitTelegramLoginResponseObject, error) {
	if h.loginStore == nil || h.botUsername == "" {
		return v1.InitTelegramLogin503JSONResponse{
			Code:    v1.TELEGRAMLOGINFAILED,
			Message: "Telegram login is not configured",
		}, nil
	}

	botToken, pollToken, err := h.loginStore.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("create login session: %w", err)
	}

	botURL := telegram.DeepLinkURL(h.botUsername, "login_"+botToken)

	return v1.InitTelegramLogin200JSONResponse{
		BotUrl:    botURL,
		PollToken: pollToken,
	}, nil
}

// PollTelegramLogin checks the status of a Telegram login session.
func (h *UserHandler) PollTelegramLogin(ctx context.Context, req v1.PollTelegramLoginRequestObject) (v1.PollTelegramLoginResponseObject, error) {
	if h.loginStore == nil {
		return v1.PollTelegramLogin404JSONResponse{
			NotFoundJSONResponse: v1.NotFoundJSONResponse{
				Code:    v1.TELEGRAMLOGINFAILED,
				Message: "Telegram login is not configured",
			},
		}, nil
	}

	result, found := h.loginStore.PollSession(req.Body.PollToken)
	if !found {
		return v1.PollTelegramLogin404JSONResponse{
			NotFoundJSONResponse: v1.NotFoundJSONResponse{
				Code:    v1.TELEGRAMLOGINFAILED,
				Message: "Login session not found or expired",
			},
		}, nil
	}

	switch result.Status {
	case telegram.LoginSessionPending:
		return v1.PollTelegramLogin202JSONResponse{
			Status: "pending",
		}, nil

	case telegram.LoginSessionAuthenticated:
		// Generate tokens for the authenticated user
		u, err := h.svc.GetByID(ctx, result.UserID)
		if err != nil {
			return nil, fmt.Errorf("get user: %w", err)
		}

		accessToken, _, err := h.jwtManager.GenerateAccessToken(u.ID, u.Role)
		if err != nil {
			return nil, fmt.Errorf("generate access token: %w", err)
		}

		refreshToken, jti, expUnix, err := h.jwtManager.GenerateRefreshToken(u.ID)
		if err != nil {
			return nil, fmt.Errorf("generate refresh token: %w", err)
		}

		if err := h.svc.SaveRefreshToken(ctx, jti, u.ID, expUnix); err != nil {
			return nil, fmt.Errorf("persist refresh token: %w", err)
		}

		cookie := makeRefreshCookie(refreshToken, h.jwtManager.RefreshTokenDuration(), h.cookieSecure)
		cookieStr := cookie.String()

		return v1.PollTelegramLogin200JSONResponse{
			Body: v1.AuthResponse{
				User:  toV1User(u),
				Token: accessToken,
			},
			Headers: v1.PollTelegramLogin200ResponseHeaders{
				SetCookie: &cookieStr,
			},
		}, nil

	case telegram.LoginSessionFailed:
		return v1.PollTelegramLogin403JSONResponse{
			ForbiddenJSONResponse: v1.ForbiddenJSONResponse{
				Code:    v1.TELEGRAMLOGINFAILED,
				Message: result.FailReason,
			},
		}, nil

	default:
		return v1.PollTelegramLogin404JSONResponse{
			NotFoundJSONResponse: v1.NotFoundJSONResponse{
				Code:    v1.TELEGRAMLOGINFAILED,
				Message: "Unknown session state",
			},
		}, nil
	}
}
