package notify

import (
	"context"

	"github.com/google/uuid"
)

// AddressResolver returns a user's per-channel identities.
type AddressResolver interface {
	ResolveAddress(ctx context.Context, userID uuid.UUID) (Address, error)
}

// RefStore persists and retrieves SentRefs so messages can be edited after
// their originating event changes state.
type RefStore interface {
	SaveRef(ctx context.Context, kind string, eventID uuid.UUID, ref SentRef) error
	GetRefs(ctx context.Context, kind string, eventID uuid.UUID) ([]SentRef, error)
}

// BlockMarker records a Telegram block for a user.
type BlockMarker interface {
	MarkTelegramBlocked(ctx context.Context, userID uuid.UUID) error
}
