package user

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"go-service-template/internal/errorz"
	"go-service-template/internal/transport/http/middleware"
	v1 "go-service-template/internal/transport/http/v1"

	"github.com/google/uuid"
)

func (h *UserHandler) UpdateUserSettings(ctx context.Context, req v1.UpdateUserSettingsRequestObject) (v1.UpdateUserSettingsResponseObject, error) {
	userID := ctx.Value(middleware.UserIDContextKey).(uuid.UUID)

	var gender *string
	if req.Body.Gender != nil {
		g := string(*req.Body.Gender)
		gender = &g
	}

	// display_name: trim whitespace; treat empty string as clearing the name
	var displayName *string
	if req.Body.DisplayName != nil {
		dn := strings.TrimSpace(*req.Body.DisplayName)
		if dn == "" {
			// explicit null / empty → clear
			displayName = nil
		} else {
			if utf8.RuneCountInString(dn) > maxDisplayNameRunes {
				return v1.UpdateUserSettings400JSONResponse{
					BadRequestJSONResponse: v1.BadRequestJSONResponse{
						Code:    v1.NAMETOOLONG,
						Message: "display_name must not exceed 32 characters",
					},
				}, nil
			}
			displayName = &dn
		}
	}

	// tag: trim whitespace; treat empty string as clearing the tag
	var tag *string
	if req.Body.Tag != nil {
		t := strings.TrimSpace(*req.Body.Tag)
		if t == "" {
			tag = nil
		} else {
			tag = &t
		}
	}

	u, err := h.svc.UpdateSettings(ctx, userID, gender, displayName, tag)
	if err != nil {
		if errors.Is(err, errorz.ErrInsufficientBalance) {
			return v1.UpdateUserSettings400JSONResponse{
				BadRequestJSONResponse: v1.BadRequestJSONResponse{
					Code:    v1.INSUFFICIENTBALANCE,
					Message: "Недостаточно обнимань для смены тега (нужно 5)",
				},
			}, nil
		}
		return nil, err
	}

	resp := v1.UpdateUserSettings200JSONResponse(toV1User(u))
	return resp, nil
}

func (h *UserHandler) ChangePassword(ctx context.Context, req v1.ChangePasswordRequestObject) (v1.ChangePasswordResponseObject, error) {
	userID := ctx.Value(middleware.UserIDContextKey).(uuid.UUID)

	// Validate new password strength
	if !hasLetter.MatchString(req.Body.NewPassword) {
		return v1.ChangePassword400JSONResponse{
			BadRequestJSONResponse: v1.BadRequestJSONResponse{
				Code:    v1.WEAKPASSWORD,
				Message: "password must contain at least one letter",
			},
		}, nil
	}
	if !hasDigit.MatchString(req.Body.NewPassword) {
		return v1.ChangePassword400JSONResponse{
			BadRequestJSONResponse: v1.BadRequestJSONResponse{
				Code:    v1.WEAKPASSWORD,
				Message: "password must contain at least one digit",
			},
		}, nil
	}
	if !hasSpecial.MatchString(req.Body.NewPassword) {
		return v1.ChangePassword400JSONResponse{
			BadRequestJSONResponse: v1.BadRequestJSONResponse{
				Code:    v1.WEAKPASSWORD,
				Message: "password must contain at least one special character",
			},
		}, nil
	}

	err := h.svc.ChangePassword(ctx, userID, req.Body.OldPassword, req.Body.NewPassword)
	if err != nil {
		if errors.Is(err, errorz.ErrWrongPassword) {
			return v1.ChangePassword400JSONResponse{
				BadRequestJSONResponse: v1.BadRequestJSONResponse{
					Code:    v1.WRONGPASSWORD,
					Message: "current password is incorrect",
				},
			}, nil
		}
		return nil, err
	}

	return v1.ChangePassword200JSONResponse{
		Message: "password changed successfully",
	}, nil
}

func (h *UserHandler) PromoteUser(ctx context.Context, req v1.PromoteUserRequestObject) (v1.PromoteUserResponseObject, error) {
	userID := ctx.Value(middleware.UserIDContextKey).(uuid.UUID)

	u, err := h.svc.PromoteUser(ctx, userID, int32(req.Body.Bid), req.Body.Message)
	if err != nil {
		if errors.Is(err, errorz.ErrInsufficientBalance) {
			return v1.PromoteUser400JSONResponse{
				BadRequestJSONResponse: v1.BadRequestJSONResponse{
					Code:    v1.INSUFFICIENTBALANCE,
					Message: "Недостаточно монет",
				},
			}, nil
		}
		return nil, err
	}

	return v1.PromoteUser200JSONResponse(toV1UserListItem(u)), nil
}
