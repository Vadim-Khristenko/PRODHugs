package models

import (
	"time"

	"github.com/google/uuid"
)

// Hug status constants
const (
	HugStatusPending   = "pending"
	HugStatusCompleted = "completed"
	HugStatusDeclined  = "declined"
	HugStatusExpired   = "expired"
	HugStatusCancelled = "cancelled"
)

// Hug type constants
const (
	HugTypeStandard = "standard"
	HugTypeBear     = "bear"
	HugTypeGroup    = "group"
	HugTypeWarm     = "warm"
	HugTypeSoul     = "soul"
	HugTypeAgressive = "agressive"
)

type Hug struct {
	ID         uuid.UUID
	GiverID    uuid.UUID
	ReceiverID uuid.UUID
	Status     string
	HugType    string
	Comment    *string
	StreakTier string
	CreatedAt  time.Time
	AcceptedAt *time.Time
}

type HugFeedItem struct {
	ID                  uuid.UUID
	GiverID             uuid.UUID
	ReceiverID          uuid.UUID
	GiverUsername       string
	ReceiverUsername    string
	GiverGender         *string
	GiverDisplayName    *string
	ReceiverDisplayName *string
	HugType             string
	HasComment          bool
	StreakTier          string
	CreatedAt           time.Time
}

type HugActivityItem struct {
	Timestamp time.Time
	Count     int64
}

type MutualHugStats struct {
	Total    int64
	Given    int64
	Received int64
}

type HugCooldown struct {
	UserAID              uuid.UUID // LEAST of the pair
	UserBID              uuid.UUID // GREATEST of the pair
	LastHugAt            time.Time
	CooldownSeconds      int32
	DeclineCooldownUntil *time.Time
}

// New models for pending hug inbox
type PendingHugInboxItem struct {
	ID               uuid.UUID
	GiverID          uuid.UUID
	ReceiverID       uuid.UUID
	GiverUsername    string
	GiverGender      *string
	GiverDisplayName *string
	HugType          string
	Comment          *string
	CreatedAt        time.Time
}

type OutgoingPendingHug struct {
	ID                  uuid.UUID
	GiverID             uuid.UUID
	ReceiverID          uuid.UUID
	ReceiverUsername    string
	ReceiverGender      *string
	ReceiverDisplayName *string
	HugType             string
	Comment             *string
	CreatedAt           time.Time
}

// HugDetail is the full hug info returned by the detail endpoint.
type HugDetail struct {
	ID                  uuid.UUID
	GiverID             uuid.UUID
	ReceiverID          uuid.UUID
	GiverUsername       string
	ReceiverUsername    string
	GiverGender         *string
	GiverDisplayName    *string
	ReceiverDisplayName *string
	Status              string
	HugType             string
	Comment             *string
	StreakTier          string
	CreatedAt           time.Time
	AcceptedAt          *time.Time
}

type SlotInfo struct {
	TotalSlots   int32
	UsedSlots    int32
	NextSlotCost *int32 // nil if at max (5)
}

// SlotCost returns the cost for the Nth slot (1-indexed). Slot 1 is free.
// Slots 2-5 cost 10, 20, 30, 40 respectively.
func SlotCost(slotNumber int32) int32 {
	if slotNumber <= 1 {
		return 0
	}
	return (slotNumber - 1) * 10
}

const MaxHugSlots = 5
