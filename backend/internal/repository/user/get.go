package user

import (
	"context"
	"database/sql"
	"errors"
	"go-service-template/internal/errorz"
	"go-service-template/internal/models"
	"go-service-template/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (r *repo) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	q := repository.Queries(ctx, r.q)

	u, err := q.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errorz.ErrUserNotFound
		}
		return nil, err
	}

	return toModelUserFromByUsername(u), nil
}

func (r *repo) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	q := repository.Queries(ctx, r.q)

	u, err := q.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errorz.ErrUserNotFound
		}
		return nil, err
	}

	return toModelUserFromByID(u), nil
}

func (r *repo) GetByTelegramID(ctx context.Context, telegramID int64) (*models.User, error) {
	q := repository.Queries(ctx, r.q)

	u, err := q.GetUserByTelegramID(ctx, pgtype.Int8{Int64: telegramID, Valid: true})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errorz.ErrUserNotFound
		}
		return nil, err
	}

	return toModelUserFromByTelegramID(u), nil
}

func (r *repo) GetByMatrixID(ctx context.Context, matrixID string) (*models.User, error) {
	q := repository.Queries(ctx, r.q)

	u, err := q.GetUserByMatrixID(ctx, pgtype.Text{String: matrixID, Valid: true})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errorz.ErrUserNotFound
		}
		return nil, err
	}

	return toModelUserFromByMatrixID(u), nil
}
