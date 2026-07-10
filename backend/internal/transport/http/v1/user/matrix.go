package user

import (
	"context"
	"errors"

	"go-service-template/internal/errorz"
	"go-service-template/internal/transport/http/middleware"
	v1 "go-service-template/internal/transport/http/v1"

	"github.com/google/uuid"
)

func (h *UserHandler) CreateMatrixLinkToken(ctx context.Context, req v1.CreateMatrixLinkTokenRequestObject) (v1.CreateMatrixLinkTokenResponseObject, error) {
	userID := ctx.Value(middleware.UserIDContextKey).(uuid.UUID)

	token, command, botUserID, err := h.svc.GenerateMatrixLinkToken(ctx, userID)
	if err != nil {
		if errors.Is(err, errorz.ErrMatrixLinkingUnavailable) {
			return v1.CreateMatrixLinkToken503JSONResponse{
				Code:    v1.MATRIXLINKINGUNAVAILABLE,
				Message: "Matrix linking is not configured",
			}, nil
		}
		return nil, err
	}

	return v1.CreateMatrixLinkToken200JSONResponse{
		Token:     token,
		Command:   command,
		BotUserId: botUserID,
	}, nil
}

func (h *UserHandler) UnlinkMatrix(ctx context.Context, req v1.UnlinkMatrixRequestObject) (v1.UnlinkMatrixResponseObject, error) {
	userID := ctx.Value(middleware.UserIDContextKey).(uuid.UUID)

	u, err := h.svc.UnlinkMatrix(ctx, userID)
	if err != nil {
		return nil, err
	}

	return v1.UnlinkMatrix200JSONResponse(toV1User(u)), nil
}
