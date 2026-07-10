package user

import (
	"context"
	"time"

	"go-service-template/internal/db/sqlc/storage"
	"go-service-template/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type repo struct {
	q *storage.Queries
}

func New(db *pgxpool.Pool) *repo {
	return &repo{
		q: storage.New(db),
	}
}

func toModelUser(u storage.User) *models.User {
	var gender *string
	if u.Gender.Valid {
		gender = &u.Gender.String
	}
	var bannedAt *time.Time
	if u.BannedAt.Valid {
		bannedAt = &u.BannedAt.Time
	}
	var createdAt *time.Time
	if u.CreatedAt.Valid {
		createdAt = &u.CreatedAt.Time
	}
	var displayName *string
	if u.DisplayName.Valid {
		displayName = &u.DisplayName.String
	}
	var telegramID *int64
	if u.TelegramID.Valid {
		telegramID = &u.TelegramID.Int64
	}
	var matrixID *string
	if u.MatrixID.Valid {
		matrixID = &u.MatrixID.String
	}
	var tag *string
	if u.Tag.Valid {
		tag = &u.Tag.String
	}
	var specialTag *string
	if u.SpecialTag.Valid {
		specialTag = &u.SpecialTag.String
	}
	var captchaCooldownUntil *time.Time
	if u.CaptchaCooldownUntil.Valid {
		captchaCooldownUntil = &u.CaptchaCooldownUntil.Time
	}
	var promotedUntil *time.Time
	if u.PromotedUntil.Valid {
		promotedUntil = &u.PromotedUntil.Time
	}
	var promotionMessage *string
	if u.PromotionMessage.Valid {
		promotionMessage = &u.PromotionMessage.String
	}
	var vipCooldownUntil *time.Time
	if u.VipCooldownUntil.Valid {
		vipCooldownUntil = &u.VipCooldownUntil.Time
	}
	return &models.User{
		ID:                   u.ID,
		Username:             u.Username,
		Role:                 u.Role,
		HashedPassword:       u.Password,
		Gender:               gender,
		DisplayName:          displayName,
		Tag:                  tag,
		SpecialTag:           specialTag,
		TelegramID:           telegramID,
		MatrixID:             matrixID,
		BannedAt:             bannedAt,
		CreatedAt:            createdAt,
		CaptchaType:          u.CaptchaType,
		CaptchaCooldownUntil: captchaCooldownUntil,
		PromotedUntil:        promotedUntil,
		PromotionMessage:     promotionMessage,
		PromotionBid:         u.PromotionBid,
		VipRemainingSeconds:  u.VipRemainingSeconds,
		VipCooldownUntil:     vipCooldownUntil,
	}
}

func toModelUserFromByID(u storage.GetUserByIDRow) *models.User {
	var gender *string
	if u.Gender.Valid {
		gender = &u.Gender.String
	}
	var bannedAt *time.Time
	if u.BannedAt.Valid {
		bannedAt = &u.BannedAt.Time
	}
	var createdAt *time.Time
	if u.CreatedAt.Valid {
		createdAt = &u.CreatedAt.Time
	}
	var displayName *string
	if u.DisplayName.Valid {
		displayName = &u.DisplayName.String
	}
	var telegramID *int64
	if u.TelegramID.Valid {
		telegramID = &u.TelegramID.Int64
	}
	var matrixID *string
	if u.MatrixID.Valid {
		matrixID = &u.MatrixID.String
	}
	var tag *string
	if u.Tag.Valid {
		tag = &u.Tag.String
	}
	var specialTag *string
	if u.SpecialTag.Valid {
		specialTag = &u.SpecialTag.String
	}
	var captchaCooldownUntil *time.Time
	if u.CaptchaCooldownUntil.Valid {
		captchaCooldownUntil = &u.CaptchaCooldownUntil.Time
	}
	var promotedUntil *time.Time
	if u.PromotedUntil.Valid {
		promotedUntil = &u.PromotedUntil.Time
	}
	var promotionMessage *string
	if u.PromotionMessage.Valid {
		promotionMessage = &u.PromotionMessage.String
	}
	var vipCooldownUntil *time.Time
	if u.VipCooldownUntil.Valid {
		vipCooldownUntil = &u.VipCooldownUntil.Time
	}
	var avgResponseTime *float64
	if u.AvgResponseTime >= 0 {
		avgResponseTime = &u.AvgResponseTime
	}

	return &models.User{
		ID:                   u.ID,
		Username:             u.Username,
		Role:                 u.Role,
		HashedPassword:       u.Password,
		Gender:               gender,
		DisplayName:          displayName,
		Tag:                  tag,
		SpecialTag:           specialTag,
		TelegramID:           telegramID,
		MatrixID:             matrixID,
		BannedAt:             bannedAt,
		CreatedAt:            createdAt,
		CaptchaType:          u.CaptchaType,
		CaptchaCooldownUntil: captchaCooldownUntil,
		PromotedUntil:        promotedUntil,
		PromotionMessage:     promotionMessage,
		PromotionBid:         u.PromotionBid,
		VipRemainingSeconds:  u.VipRemainingSeconds,
		VipCooldownUntil:     vipCooldownUntil,
		Balance:              u.Balance,
		AvgResponseTime:      avgResponseTime,
	}
}

func toModelUserFromByUsername(u storage.GetUserByUsernameRow) *models.User {
	var gender *string
	if u.Gender.Valid {
		gender = &u.Gender.String
	}
	var bannedAt *time.Time
	if u.BannedAt.Valid {
		bannedAt = &u.BannedAt.Time
	}
	var createdAt *time.Time
	if u.CreatedAt.Valid {
		createdAt = &u.CreatedAt.Time
	}
	var displayName *string
	if u.DisplayName.Valid {
		displayName = &u.DisplayName.String
	}
	var telegramID *int64
	if u.TelegramID.Valid {
		telegramID = &u.TelegramID.Int64
	}
	var matrixID *string
	if u.MatrixID.Valid {
		matrixID = &u.MatrixID.String
	}
	var tag *string
	if u.Tag.Valid {
		tag = &u.Tag.String
	}
	var specialTag *string
	if u.SpecialTag.Valid {
		specialTag = &u.SpecialTag.String
	}
	var captchaCooldownUntil *time.Time
	if u.CaptchaCooldownUntil.Valid {
		captchaCooldownUntil = &u.CaptchaCooldownUntil.Time
	}
	var promotedUntil *time.Time
	if u.PromotedUntil.Valid {
		promotedUntil = &u.PromotedUntil.Time
	}
	var promotionMessage *string
	if u.PromotionMessage.Valid {
		promotionMessage = &u.PromotionMessage.String
	}
	var vipCooldownUntil *time.Time
	if u.VipCooldownUntil.Valid {
		vipCooldownUntil = &u.VipCooldownUntil.Time
	}
	var avgResponseTime *float64
	if u.AvgResponseTime >= 0 {
		avgResponseTime = &u.AvgResponseTime
	}

	return &models.User{
		ID:                   u.ID,
		Username:             u.Username,
		Role:                 u.Role,
		HashedPassword:       u.Password,
		Gender:               gender,
		DisplayName:          displayName,
		Tag:                  tag,
		SpecialTag:           specialTag,
		TelegramID:           telegramID,
		MatrixID:             matrixID,
		BannedAt:             bannedAt,
		CreatedAt:            createdAt,
		CaptchaType:          u.CaptchaType,
		CaptchaCooldownUntil: captchaCooldownUntil,
		PromotedUntil:        promotedUntil,
		PromotionMessage:     promotionMessage,
		PromotionBid:         u.PromotionBid,
		VipRemainingSeconds:  u.VipRemainingSeconds,
		VipCooldownUntil:     vipCooldownUntil,
		Balance:              u.Balance,
		AvgResponseTime:      avgResponseTime,
	}
}

func toModelUserFromByTelegramID(u storage.GetUserByTelegramIDRow) *models.User {
	var gender *string
	if u.Gender.Valid {
		gender = &u.Gender.String
	}
	var bannedAt *time.Time
	if u.BannedAt.Valid {
		bannedAt = &u.BannedAt.Time
	}
	var createdAt *time.Time
	if u.CreatedAt.Valid {
		createdAt = &u.CreatedAt.Time
	}
	var displayName *string
	if u.DisplayName.Valid {
		displayName = &u.DisplayName.String
	}
	var telegramID *int64
	if u.TelegramID.Valid {
		telegramID = &u.TelegramID.Int64
	}
	var matrixID *string
	if u.MatrixID.Valid {
		matrixID = &u.MatrixID.String
	}
	var tag *string
	if u.Tag.Valid {
		tag = &u.Tag.String
	}
	var specialTag *string
	if u.SpecialTag.Valid {
		specialTag = &u.SpecialTag.String
	}
	var captchaCooldownUntil *time.Time
	if u.CaptchaCooldownUntil.Valid {
		captchaCooldownUntil = &u.CaptchaCooldownUntil.Time
	}
	var promotedUntil *time.Time
	if u.PromotedUntil.Valid {
		promotedUntil = &u.PromotedUntil.Time
	}
	var promotionMessage *string
	if u.PromotionMessage.Valid {
		promotionMessage = &u.PromotionMessage.String
	}
	var vipCooldownUntil *time.Time
	if u.VipCooldownUntil.Valid {
		vipCooldownUntil = &u.VipCooldownUntil.Time
	}
	var avgResponseTime *float64
	if u.AvgResponseTime >= 0 {
		avgResponseTime = &u.AvgResponseTime
	}

	return &models.User{
		ID:                   u.ID,
		Username:             u.Username,
		Role:                 u.Role,
		HashedPassword:       u.Password,
		Gender:               gender,
		DisplayName:          displayName,
		Tag:                  tag,
		SpecialTag:           specialTag,
		TelegramID:           telegramID,
		MatrixID:             matrixID,
		BannedAt:             bannedAt,
		CreatedAt:            createdAt,
		CaptchaType:          u.CaptchaType,
		CaptchaCooldownUntil: captchaCooldownUntil,
		PromotedUntil:        promotedUntil,
		PromotionMessage:     promotionMessage,
		PromotionBid:         u.PromotionBid,
		VipRemainingSeconds:  u.VipRemainingSeconds,
		VipCooldownUntil:     vipCooldownUntil,
		Balance:              u.Balance,
		AvgResponseTime:      avgResponseTime,
	}
}

func toAdminUser(u storage.ListUsersAdminRow) *models.AdminUser {
	var gender *string
	if u.Gender.Valid {
		gender = &u.Gender.String
	}
	var bannedAt *time.Time
	if u.BannedAt.Valid {
		bannedAt = &u.BannedAt.Time
	}
	var createdAt *time.Time
	if u.CreatedAt.Valid {
		createdAt = &u.CreatedAt.Time
	}
	var displayName *string
	if u.DisplayName.Valid {
		displayName = &u.DisplayName.String
	}
	var lastVisitAt *time.Time
	if u.LastVisitAt.Valid {
		lastVisitAt = &u.LastVisitAt.Time
	}
	var tag *string
	if u.Tag.Valid {
		tag = &u.Tag.String
	}
	var specialTag *string
	if u.SpecialTag.Valid {
		specialTag = &u.SpecialTag.String
	}
	var captchaCooldownUntil *time.Time
	if u.CaptchaCooldownUntil.Valid {
		captchaCooldownUntil = &u.CaptchaCooldownUntil.Time
	}
	var promotedUntil *time.Time
	if u.PromotedUntil.Valid {
		promotedUntil = &u.PromotedUntil.Time
	}
	var promotionMessage *string
	if u.PromotionMessage.Valid {
		promotionMessage = &u.PromotionMessage.String
	}
	var vipCooldownUntil *time.Time
	if u.VipCooldownUntil.Valid {
		vipCooldownUntil = &u.VipCooldownUntil.Time
	}
	return &models.AdminUser{
		ID:                   u.ID,
		Username:             u.Username,
		Role:                 u.Role,
		Gender:               gender,
		DisplayName:          displayName,
		Tag:                  tag,
		SpecialTag:           specialTag,
		BannedAt:             bannedAt,
		CreatedAt:            createdAt,
		Balance:              u.Balance,
		LastVisitAt:          lastVisitAt,
		CaptchaType:          u.CaptchaType,
		CaptchaCooldownUntil: captchaCooldownUntil,
		PromotedUntil:        promotedUntil,
		PromotionMessage:     promotionMessage,
		PromotionBid:         u.PromotionBid,
		VipRemainingSeconds:  u.VipRemainingSeconds,
		VipCooldownUntil:     vipCooldownUntil,
	}
}

func toAdminUserFromSearch(u storage.SearchUsersAdminRow) *models.AdminUser {
	var gender *string
	if u.Gender.Valid {
		gender = &u.Gender.String
	}
	var bannedAt *time.Time
	if u.BannedAt.Valid {
		bannedAt = &u.BannedAt.Time
	}
	var createdAt *time.Time
	if u.CreatedAt.Valid {
		createdAt = &u.CreatedAt.Time
	}
	var displayName *string
	if u.DisplayName.Valid {
		displayName = &u.DisplayName.String
	}
	var lastVisitAt *time.Time
	if u.LastVisitAt.Valid {
		lastVisitAt = &u.LastVisitAt.Time
	}
	var tag *string
	if u.Tag.Valid {
		tag = &u.Tag.String
	}
	var specialTag *string
	if u.SpecialTag.Valid {
		specialTag = &u.SpecialTag.String
	}
	var captchaCooldownUntil *time.Time
	if u.CaptchaCooldownUntil.Valid {
		captchaCooldownUntil = &u.CaptchaCooldownUntil.Time
	}
	var promotedUntil *time.Time
	if u.PromotedUntil.Valid {
		promotedUntil = &u.PromotedUntil.Time
	}
	var promotionMessage *string
	if u.PromotionMessage.Valid {
		promotionMessage = &u.PromotionMessage.String
	}
	var vipCooldownUntil *time.Time
	if u.VipCooldownUntil.Valid {
		vipCooldownUntil = &u.VipCooldownUntil.Time
	}
	return &models.AdminUser{
		ID:                   u.ID,
		Username:             u.Username,
		Role:                 u.Role,
		Gender:               gender,
		DisplayName:          displayName,
		Tag:                  tag,
		SpecialTag:           specialTag,
		BannedAt:             bannedAt,
		CreatedAt:            createdAt,
		Balance:              u.Balance,
		LastVisitAt:          lastVisitAt,
		CaptchaType:          u.CaptchaType,
		CaptchaCooldownUntil: captchaCooldownUntil,
		PromotedUntil:        promotedUntil,
		PromotionMessage:     promotionMessage,
		PromotionBid:         u.PromotionBid,
		VipRemainingSeconds:  u.VipRemainingSeconds,
		VipCooldownUntil:     vipCooldownUntil,
	}
}

func toModelUserListItemFromVIP(row storage.ListVIPUsersRow) *models.User {
	var gender *string
	if row.Gender.Valid {
		gender = &row.Gender.String
	}
	var displayName *string
	if row.DisplayName.Valid {
		displayName = &row.DisplayName.String
	}
	var tag *string
	if row.Tag.Valid {
		tag = &row.Tag.String
	}
	var specialTag *string
	if row.SpecialTag.Valid {
		specialTag = &row.SpecialTag.String
	}
	var promotedUntil *time.Time
	if row.PromotedUntil.Valid {
		promotedUntil = &row.PromotedUntil.Time
	}
	var promotionMessage *string
	if row.PromotionMessage.Valid {
		promotionMessage = &row.PromotionMessage.String
	}
	var vipCooldownUntil *time.Time
	if row.VipCooldownUntil.Valid {
		vipCooldownUntil = &row.VipCooldownUntil.Time
	}

	var avgResponseTime *float64
	if row.AvgResponseTime >= 0 {
		avgResponseTime = &row.AvgResponseTime
	}

	return &models.User{
		ID:                  row.ID,
		Username:            row.Username,
		Role:                row.Role,
		Gender:              gender,
		DisplayName:         displayName,
		Tag:                 tag,
		SpecialTag:          specialTag,
		IsTelegramLinked:    row.IsTelegramLinked,
		PromotedUntil:       promotedUntil,
		PromotionMessage:    promotionMessage,
		PromotionBid:        row.PromotionBid,
		VipRemainingSeconds: row.VipRemainingSeconds,
		VipCooldownUntil:    vipCooldownUntil,
		IsRecentlyActive:    row.IsRecentlyActive,
		AvgResponseTime:     avgResponseTime,
	}
}

func (r *repo) ListVIPUsers(ctx context.Context) ([]*models.User, error) {
	rows, err := r.q.ListVIPUsers(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*models.User, len(rows))
	for i, row := range rows {
		result[i] = toModelUserListItemFromVIP(row)
	}
	return result, nil
}
