package user

import (
	"context"
	"fmt"

	matrixnotify "go-service-template/internal/notify/matrix"
	v1 "go-service-template/internal/transport/http/v1"
)

// InitMatrixLogin creates a new Matrix login session and returns the command the
// user must send to the bot along with the bot user id and a poll token.
func (h *UserHandler) InitMatrixLogin(ctx context.Context, req v1.InitMatrixLoginRequestObject) (v1.InitMatrixLoginResponseObject, error) {
	command, botUserID, pollToken, err := h.svc.InitMatrixLogin()
	if err != nil {
		return v1.InitMatrixLogin503JSONResponse{
			Code:    v1.MATRIXLOGINFAILED,
			Message: "Matrix login is not configured",
		}, nil
	}

	return v1.InitMatrixLogin200JSONResponse{
		Command:   command,
		BotUserId: botUserID,
		PollToken: pollToken,
	}, nil
}

// PollMatrixLogin checks the status of a Matrix login session.
func (h *UserHandler) PollMatrixLogin(ctx context.Context, req v1.PollMatrixLoginRequestObject) (v1.PollMatrixLoginResponseObject, error) {
	if h.matrixLoginStore == nil {
		return v1.PollMatrixLogin404JSONResponse{
			NotFoundJSONResponse: v1.NotFoundJSONResponse{
				Code:    v1.MATRIXLOGINFAILED,
				Message: "Matrix login is not configured",
			},
		}, nil
	}

	result, found := h.matrixLoginStore.PollSession(req.Body.PollToken)
	if !found {
		return v1.PollMatrixLogin404JSONResponse{
			NotFoundJSONResponse: v1.NotFoundJSONResponse{
				Code:    v1.MATRIXLOGINFAILED,
				Message: "Login session not found or expired",
			},
		}, nil
	}

	switch result.Status {
	case matrixnotify.LoginSessionPending:
		return v1.PollMatrixLogin202JSONResponse{
			Status: "pending",
		}, nil

	case matrixnotify.LoginSessionAuthenticated:
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

		return v1.PollMatrixLogin200JSONResponse{
			Body: v1.AuthResponse{
				User:  toV1User(u),
				Token: accessToken,
			},
			Headers: v1.PollMatrixLogin200ResponseHeaders{
				SetCookie: &cookieStr,
			},
		}, nil

	case matrixnotify.LoginSessionFailed:
		return v1.PollMatrixLogin403JSONResponse{
			ForbiddenJSONResponse: v1.ForbiddenJSONResponse{
				Code:    v1.MATRIXLOGINFAILED,
				Message: result.FailReason,
			},
		}, nil

	default:
		return v1.PollMatrixLogin404JSONResponse{
			NotFoundJSONResponse: v1.NotFoundJSONResponse{
				Code:    v1.MATRIXLOGINFAILED,
				Message: "Unknown session state",
			},
		}, nil
	}
}
