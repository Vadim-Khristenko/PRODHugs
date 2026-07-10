package user

import (
	"context"

	"go-service-template/internal/repository"

	"github.com/google/uuid"
)

// MarkTelegramBlocked records a Telegram 403 for this user; idempotent.
func (r *repo) MarkTelegramBlocked(ctx context.Context, userID uuid.UUID) error {
	q := repository.Queries(ctx, r.q)
	return q.MarkTelegramBlocked(ctx, userID)
}

// ClearTelegramBlocked lifts the block; called on any inbound update.
func (r *repo) ClearTelegramBlocked(ctx context.Context, userID uuid.UUID) error {
	q := repository.Queries(ctx, r.q)
	return q.ClearTelegramBlocked(ctx, userID)
}

// MarkDailyReminderSent stamps the last daily-reminder time for this user.
func (r *repo) MarkDailyReminderSent(ctx context.Context, userID uuid.UUID) error {
	q := repository.Queries(ctx, r.q)
	return q.MarkDailyReminderSent(ctx, userID)
}

// DailyReminderCandidate is the minimal slice the reminder cron needs.
type DailyReminderCandidate struct {
	ID          uuid.UUID
	TelegramID  int64
	Username    string
	DisplayName *string
	Gender      *string
}

// ListDailyReminderCandidates returns up to limit users eligible for a daily
// reward reminder (Telegram linked, not blocked, not banned, not reminded
// today, no reward claimed today).
func (r *repo) ListDailyReminderCandidates(ctx context.Context, limit int32) ([]DailyReminderCandidate, error) {
	q := repository.Queries(ctx, r.q)
	rows, err := q.ListDailyReminderCandidates(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]DailyReminderCandidate, 0, len(rows))
	for _, row := range rows {
		if !row.TelegramID.Valid {
			continue
		}
		c := DailyReminderCandidate{
			ID:         row.ID,
			TelegramID: row.TelegramID.Int64,
			Username:   row.Username,
		}
		if row.DisplayName.Valid {
			s := row.DisplayName.String
			c.DisplayName = &s
		}
		if row.Gender.Valid {
			s := row.Gender.String
			c.Gender = &s
		}
		out = append(out, c)
	}
	return out, nil
}
