package models

import (
	"time"

	"github.com/google/uuid"
)

// PairIntimacy represents the raw intimacy data for a pair of users.
type PairIntimacy struct {
	UserAID     uuid.UUID
	UserBID     uuid.UUID
	RawScore    int
	LastHugAt   time.Time
	LastDecayAt time.Time
}

// IntimacyTier defines an intimacy level with its unlocks.
type IntimacyTier struct {
	Level             int
	Name              string
	MinScore          int
	CooldownReduction float64 // 0.0 to 0.5
	BonusCoins        int
	UnlockedHugTypes  []string
}

// IntimacyInfo is the computed intimacy data returned to clients.
type IntimacyInfo struct {
	RawScore             int
	Tier                 int
	TierName             string
	NextTierAt           *int // nil if max tier
	CooldownReductionPct int  // 0-50
	AvailableHugTypes    []string
	BonusCoins           int
}

// ConnectionItem represents a user's connection with intimacy info.
type ConnectionItem struct {
	UserID      uuid.UUID
	Username    string
	Gender      *string
	DisplayName *string
	Intimacy    IntimacyInfo
}

// Intimacy tier definitions
var IntimacyTiers = []IntimacyTier{
	{Level: 0, Name: "Незнакомцы", MinScore: 0, CooldownReduction: 0.0, BonusCoins: 0, UnlockedHugTypes: []string{HugTypeStandard}},
	{Level: 1, Name: "Знакомые", MinScore: 5, CooldownReduction: 0.10, BonusCoins: 0, UnlockedHugTypes: []string{HugTypeStandard}},
	{Level: 2, Name: "Приятели", MinScore: 15, CooldownReduction: 0.20, BonusCoins: 1, UnlockedHugTypes: []string{HugTypeStandard, HugTypeBear, HugTypeAgressive}},
	{Level: 3, Name: "Друзья", MinScore: 30, CooldownReduction: 0.30, BonusCoins: 1, UnlockedHugTypes: []string{HugTypeStandard, HugTypeBear, HugTypeAgressive, HugTypeGroup}},
	{Level: 4, Name: "Близкие", MinScore: 50, CooldownReduction: 0.40, BonusCoins: 2, UnlockedHugTypes: []string{HugTypeStandard, HugTypeBear, HugTypeAgressive, HugTypeGroup, HugTypeWarm}},
	{Level: 5, Name: "Родные души", MinScore: 80, CooldownReduction: 0.50, BonusCoins: 2, UnlockedHugTypes: []string{HugTypeStandard, HugTypeBear, HugTypeAgressive, HugTypeGroup, HugTypeWarm, HugTypeSoul}},
}

// ComputeTier returns the tier for a given raw score.
func ComputeTier(rawScore int) IntimacyTier {
	tier := IntimacyTiers[0]
	for _, t := range IntimacyTiers {
		if rawScore >= t.MinScore {
			tier = t
		} else {
			break
		}
	}
	return tier
}

// ComputeIntimacyInfo builds the full IntimacyInfo from a raw score.
func ComputeIntimacyInfo(rawScore int) IntimacyInfo {
	tier := ComputeTier(rawScore)
	info := IntimacyInfo{
		RawScore:             rawScore,
		Tier:                 tier.Level,
		TierName:             tier.Name,
		CooldownReductionPct: int(tier.CooldownReduction * 100),
		AvailableHugTypes:    tier.UnlockedHugTypes,
		BonusCoins:           tier.BonusCoins,
	}

	// Calculate next tier threshold
	if tier.Level < len(IntimacyTiers)-1 {
		nextMin := IntimacyTiers[tier.Level+1].MinScore
		info.NextTierAt = &nextMin
	}

	return info
}

// IsHugTypeUnlocked checks if a hug type is available at the given raw score.
func IsHugTypeUnlocked(rawScore int, hugType string) bool {
	tier := ComputeTier(rawScore)
	for _, ht := range tier.UnlockedHugTypes {
		if ht == hugType {
			return true
		}
	}
	return false
}

// LeaderboardPairEntry represents a pair in the public intimacy leaderboard.
type LeaderboardPairEntry struct {
	UserAID          uuid.UUID
	UserAUsername    string
	UserADisplayName *string
	UserBID          uuid.UUID
	UserBUsername    string
	UserBDisplayName *string
	RawScore         int
	Tier             int
	TierName         string
}

// ValidHugType checks if the string is a valid hug type.
func ValidHugType(hugType string) bool {
	switch hugType {
	case HugTypeStandard, HugTypeBear, HugTypeGroup, HugTypeWarm, HugTypeSoul, HugTypeAgressive:
		return true
	default:
		return false
	}
}
