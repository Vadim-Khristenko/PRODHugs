package notification

import (
	"context"

	"go-service-template/internal/db/sqlc/storage"
	"go-service-template/internal/notify"
	"go-service-template/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type repo struct{ q *storage.Queries }

func New(db *pgxpool.Pool) *repo { return &repo{q: storage.New(db)} }

func (r *repo) SaveRef(ctx context.Context, kind string, eventID uuid.UUID, ref notify.SentRef) error {
	q := repository.Queries(ctx, r.q)
	return q.SaveNotificationRef(ctx, storage.SaveNotificationRefParams{
		Kind:       kind,
		EventID:    eventID,
		Provider:   ref.Provider,
		ChatRef:    ref.ChatRef,
		MessageRef: ref.MessageRef,
	})
}

func (r *repo) GetRefs(ctx context.Context, kind string, eventID uuid.UUID) ([]notify.SentRef, error) {
	q := repository.Queries(ctx, r.q)
	rows, err := q.GetNotificationRefs(ctx, storage.GetNotificationRefsParams{Kind: kind, EventID: eventID})
	if err != nil {
		return nil, err
	}
	out := make([]notify.SentRef, 0, len(rows))
	for _, row := range rows {
		out = append(out, notify.SentRef{Provider: row.Provider, ChatRef: row.ChatRef, MessageRef: row.MessageRef})
	}
	return out, nil
}
