package user

import (
	"context"

	"go-service-template/internal/notify"
	"go-service-template/internal/repository"

	"github.com/google/uuid"
)

// ResolveAddress returns the user's per-channel identities for the notify
// Router. A nil field means that channel isn't linked.
func (r *repo) ResolveAddress(ctx context.Context, userID uuid.UUID) (notify.Address, error) {
	q := repository.Queries(ctx, r.q)
	row, err := q.GetUserAddress(ctx, userID)
	if err != nil {
		return notify.Address{}, err
	}
	var a notify.Address
	if row.TelegramID.Valid {
		v := row.TelegramID.Int64
		a.TelegramID = &v
	}
	if row.MatrixID.Valid {
		v := row.MatrixID.String
		a.MatrixID = &v
	}
	if row.MatrixRoomID.Valid {
		v := row.MatrixRoomID.String
		a.MatrixRoom = &v
	}
	return a, nil
}
