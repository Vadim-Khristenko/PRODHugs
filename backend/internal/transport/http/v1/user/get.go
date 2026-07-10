package user

import (
	"context"
	"errors"
	"go-service-template/internal/errorz"
	"go-service-template/internal/transport/http/middleware"
	v1 "go-service-template/internal/transport/http/v1"

	"github.com/google/uuid"
)

func (h *UserHandler) Login(ctx context.Context, req v1.LoginRequestObject) (v1.LoginResponseObject, error) {
	u, accessToken, refreshToken, err := h.svc.Login(ctx, req.Body.Username, req.Body.Password)
	if err != nil {
		if errors.Is(err, errorz.ErrUserNotFound) || errors.Is(err, errorz.ErrInvalidCredentials) {
			return v1.Login401JSONResponse{
				UnauthorizedJSONResponse: v1.UnauthorizedJSONResponse{
					Message: "invalid username or password",
					Code:    v1.INVALIDCREDENTIALS,
				},
			}, nil
		}
		if errors.Is(err, errorz.ErrUserBanned) {
			return v1.Login403JSONResponse{
				ForbiddenJSONResponse: v1.ForbiddenJSONResponse{
					Message: "Ваш аккаунт заблокирован",
					Code:    v1.USERBANNED,
				},
			}, nil
		}
		return nil, err
	}

	cookie := makeRefreshCookie(refreshToken, h.jwtManager.RefreshTokenDuration(), h.cookieSecure)
	cookieStr := cookie.String()

	return v1.Login200JSONResponse{
		Body: v1.AuthResponse{
			User:  toV1User(u),
			Token: accessToken,
		},
		Headers: v1.Login200ResponseHeaders{
			SetCookie: &cookieStr,
		},
	}, nil
}

func (h *UserHandler) GetCurrentUser(ctx context.Context, req v1.GetCurrentUserRequestObject) (v1.GetCurrentUserResponseObject, error) {
	userID := ctx.Value(middleware.UserIDContextKey).(uuid.UUID)

	u, err := h.svc.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, errorz.ErrUserNotFound) {
			return v1.GetCurrentUser404JSONResponse{
				NotFoundJSONResponse: v1.NotFoundJSONResponse{
					Message: "user not found",
					Code:    v1.USERNOTFOUND,
				},
			}, nil
		}
		return nil, err
	}

	resp := v1.GetCurrentUser200JSONResponse(toV1User(u))
	return resp, nil
}

func (h *UserHandler) GetVIPUsers(ctx context.Context, req v1.GetVIPUsersRequestObject) (v1.GetVIPUsersResponseObject, error) {
	users, err := h.svc.ListVIPUsers(ctx)
	if err != nil {
		return nil, err
	}

	result := make(v1.GetVIPUsers200JSONResponse, len(users))
	for i, u := range users {
		result[i] = toV1UserListItem(u)
	}

	return result, nil
}
