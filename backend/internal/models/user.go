package models

import (
	"time"

	"github.com/google/uuid"
)

type CreateUser struct {
	Username       string
	Password       string
	HashedPassword string
	Role           string
	Gender         *string
}

type User struct {
	ID                   uuid.UUID
	Username             string
	Role                 string
	HashedPassword       string
	Gender               *string
	DisplayName          *string
	Tag                  *string
	SpecialTag           *string
	TelegramID           *int64
	MatrixID             *string
	BannedAt             *time.Time
	CreatedAt            *time.Time
	CaptchaType          string
	CaptchaCooldownUntil *time.Time
	PromotedUntil        *time.Time
	PromotionMessage     *string
	PromotionBid         int32
	VipRemainingSeconds  int32
	VipCooldownUntil     *time.Time
	IsRecentlyActive     bool
	IsTelegramLinked     bool
	AvgResponseTime      *float64
	Balance              int32
}

type AdminUser struct {
	ID                   uuid.UUID
	Username             string
	Role                 string
	Gender               *string
	DisplayName          *string
	Tag                  *string
	SpecialTag           *string
	BannedAt             *time.Time
	CreatedAt            *time.Time
	Balance              int32
	LastVisitAt          *time.Time // proxy for last user visit via refresh token
	CaptchaType          string
	CaptchaCooldownUntil *time.Time
	PromotedUntil        *time.Time
	PromotionMessage     *string
	PromotionBid         int32
	VipRemainingSeconds  int32
	VipCooldownUntil     *time.Time
}

type AdminStats struct {
	TotalUsers  int64
	BannedUsers int64
}

type BlockedUser struct {
	ID          uuid.UUID
	Username    string
	Gender      *string
	DisplayName *string
	Tag         *string
	SpecialTag  *string
	CreatedAt   time.Time
}

// PromotionCost returns the cost to promote a user for a given duration in hours.
func PromotionCost(hours int32) int32 {
	// 5 coins per hour
	return hours * 5
}
