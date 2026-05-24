package user

import (
	"context"
	"errors"
	"go-service-template/internal/errorz"
	"go-service-template/internal/models"
	v1 "go-service-template/internal/transport/http/v1"
	"regexp"
)

var (
	hasLetter  = regexp.MustCompile(`[a-zA-Z]`)
	hasDigit   = regexp.MustCompile(`[0-9]`)
	hasSpecial = regexp.MustCompile(`[^a-zA-Z0-9\s]`)
)

func (h *UserHandler) RegisterUser(ctx context.Context, req v1.RegisterUserRequestObject) (v1.RegisterUserResponseObject, error) {
	if !hasLetter.MatchString(req.Body.Password) {
		return v1.RegisterUser400JSONResponse{
			BadRequestJSONResponse: v1.BadRequestJSONResponse{
				Code:    v1.WEAKPASSWORD,
				Message: "password must contain at least one letter",
			},
		}, nil
	}
	if !hasDigit.MatchString(req.Body.Password) {
		return v1.RegisterUser400JSONResponse{
			BadRequestJSONResponse: v1.BadRequestJSONResponse{
				Code:    v1.WEAKPASSWORD,
				Message: "password must contain at least one digit",
			},
		}, nil
	}
	if !hasSpecial.MatchString(req.Body.Password) {
		return v1.RegisterUser400JSONResponse{
			BadRequestJSONResponse: v1.BadRequestJSONResponse{
				Code:    v1.WEAKPASSWORD,
				Message: "password must contain at least one special character",
			},
		}, nil
	}

	var gender *string
	if req.Body.Gender != nil {
		g := string(*req.Body.Gender)
		gender = &g
	}

	input := &models.CreateUser{
		Username:       req.Body.Username,
		Password:       req.Body.Password,
		HashedPassword: req.Body.Password,
		Role:           "user",
		Gender:         gender,
	}
	u, accessToken, refreshToken, err := h.svc.Create(ctx, input)
	if err != nil {
		if errors.Is(err, errorz.ErrUserAlreadyExists) {
			return v1.RegisterUser409JSONResponse{
				ConflictJSONResponse: v1.ConflictJSONResponse{
					Message: "user with the given username already exists",
					Code:    v1.USERALREADYEXISTS,
				},
			}, nil
		}
		return nil, err
	}

	cookie := makeRefreshCookie(refreshToken, h.jwtManager.RefreshTokenDuration(), h.cookieSecure)
	cookieStr := cookie.String()

	return v1.RegisterUser201JSONResponse{
		Body: v1.AuthResponse{
			User:  toV1User(u),
			Token: accessToken,
		},
		Headers: v1.RegisterUser201ResponseHeaders{
			SetCookie: cookieStr,
		},
	}, nil
}
