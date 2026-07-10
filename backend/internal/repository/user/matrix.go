package user

import (
	"context"

	storage "go-service-template/internal/db/sqlc/storage"
	"go-service-template/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// SetMatrixLink records the user's Matrix identity and the DM room used to
// deliver notifications.
func (r *repo) SetMatrixLink(ctx context.Context, userID uuid.UUID, matrixID, roomID string) error {
	q := repository.Queries(ctx, r.q)
	return q.SetMatrixLink(ctx, storage.SetMatrixLinkParams{
		ID:           userID,
		MatrixID:     pgtype.Text{String: matrixID, Valid: true},
		MatrixRoomID: pgtype.Text{String: roomID, Valid: true},
	})
}

// ClearMatrixLink unlinks the user's Matrix account.
func (r *repo) ClearMatrixLink(ctx context.Context, userID uuid.UUID) error {
	q := repository.Queries(ctx, r.q)
	return q.ClearMatrixLink(ctx, userID)
}

// IsMatrixIDTaken reports whether another user already linked this Matrix id.
func (r *repo) IsMatrixIDTaken(ctx context.Context, matrixID string, excludeUserID uuid.UUID) (bool, error) {
	q := repository.Queries(ctx, r.q)
	return q.IsMatrixIDTaken(ctx, storage.IsMatrixIDTakenParams{
		MatrixID: pgtype.Text{String: matrixID, Valid: true},
		ID:       excludeUserID,
	})
}
