package user

import (
	"context"
	"go-service-template/internal/jwt"
	"go-service-template/internal/models"
	matrixnotify "go-service-template/internal/notify/matrix"
	userService "go-service-template/internal/service/user"
	"go-service-template/internal/telegram"
	v1 "go-service-template/internal/transport/http/v1"
	"time"

	"github.com/google/uuid"
)

type service interface {
	Create(ctx context.Context, input *models.CreateUser) (*models.User, string, string, error)
	Login(ctx context.Context, username string, password string) (*models.User, string, string, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	UpdateSettings(ctx context.Context, id uuid.UUID, gender *string, displayName *string, tag *string) (*models.User, error)
	ChangePassword(ctx context.Context, id uuid.UUID, oldPassword, newPassword string) error
	GenerateLinkToken(ctx context.Context, userID uuid.UUID) (string, string, error)
	UnlinkTelegram(ctx context.Context, userID uuid.UUID) (*models.User, error)
	GenerateMatrixLinkToken(ctx context.Context, userID uuid.UUID) (string, string, string, error)
	InitMatrixLogin() (command, botUserID, pollToken string, err error)
	LoginViaTelegramWidget(ctx context.Context, d userService.TelegramWidgetData) (*models.User, error)
	UnlinkMatrix(ctx context.Context, userID uuid.UUID) (*models.User, error)
	SaveRefreshToken(ctx context.Context, jti string, userID uuid.UUID, expiresAtUnix int64) error
	IsRefreshTokenActive(ctx context.Context, jti string) (bool, error)
	RevokeRefreshToken(ctx context.Context, jti string) error
	RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error
	PromoteUser(ctx context.Context, id uuid.UUID, bid int32, message *string) (*models.User, error)
	ListVIPUsers(ctx context.Context) ([]*models.User, error)
	GenerateSudokuCaptcha(ctx context.Context, userID uuid.UUID) (uuid.UUID, [][]int, error)
	VerifySudokuCell(ctx context.Context, captchaID uuid.UUID, userID uuid.UUID, row, col, value int) (*userService.CaptchaResult, error)
	CompleteSudoku(ctx context.Context, captchaID uuid.UUID, userID uuid.UUID) (string, error)
	GenerateCasinoCaptcha(ctx context.Context, userID uuid.UUID) (uuid.UUID, time.Time, error)
	SpinCasino(ctx context.Context, captchaID uuid.UUID, userID uuid.UUID) (*userService.CasinoSpinResult, error)
}

type UserHandler struct {
	svc          service
	jwtManager   *jwt.Manager
	cookieSecure bool
	loginStore   *telegram.LoginStore
	botUsername  string

	matrixLoginStore *matrixnotify.LoginStore
}

func New(svc service, jwtManager *jwt.Manager, cookieSecure bool) *UserHandler {
	return &UserHandler{svc: svc, jwtManager: jwtManager, cookieSecure: cookieSecure}
}

// SetTelegramLoginStore configures the login store and bot username for
// the Telegram login endpoints. Called after construction.
func (h *UserHandler) SetTelegramLoginStore(store *telegram.LoginStore, botUsername string) {
	h.loginStore = store
	h.botUsername = botUsername
}

// SetMatrixLoginStore configures the Matrix login store used by the Matrix
// login poll endpoint. Called after construction.
func (h *UserHandler) SetMatrixLoginStore(store *matrixnotify.LoginStore) {
	h.matrixLoginStore = store
}

func ptr[T any](v T) *T {
	return &v
}

func toV1User(u *models.User) v1.User {
	bal := int(u.Balance)
	user := v1.User{
		Id:                   u.ID,
		Username:             u.Username,
		Role:                 v1.UserRole(u.Role),
		DisplayName:          u.DisplayName,
		Tag:                  u.Tag,
		SpecialTag:           u.SpecialTag,
		TelegramId:           u.TelegramID,
		MatrixId:             u.MatrixID,
		CaptchaType:          v1.CaptchaType(u.CaptchaType),
		CaptchaCooldownUntil: u.CaptchaCooldownUntil,
		PromotedUntil:        u.PromotedUntil,
		PromotionMessage:     u.PromotionMessage,
		PromotionBid:         ptr(int(u.PromotionBid)),
		VipRemainingSeconds:  ptr(int(u.VipRemainingSeconds)),
		VipCooldownUntil:     u.VipCooldownUntil,
		IsRecentlyActive:     ptr(u.IsRecentlyActive),
		Balance:              &bal,
	}
	if u.Gender != nil {
		g := v1.Gender(*u.Gender)
		user.Gender = &g
	}
	return user
}

func toV1UserListItem(u *models.User) v1.UserListItem {
	var avgResponseTime *float32
	if u.AvgResponseTime != nil {
		val := float32(*u.AvgResponseTime)
		avgResponseTime = &val
	}

	item := v1.UserListItem{
		Id:                  u.ID,
		Username:            u.Username,
		Role:                u.Role,
		DisplayName:         u.DisplayName,
		Tag:                 u.Tag,
		SpecialTag:          u.SpecialTag,
		IsTelegramLinked:    u.IsTelegramLinked,
		AvgResponseTime:     avgResponseTime,
		PromotedUntil:       u.PromotedUntil,
		PromotionMessage:    u.PromotionMessage,
		PromotionBid:        ptr(int(u.PromotionBid)),
		VipRemainingSeconds: ptr(int(u.VipRemainingSeconds)),
		VipCooldownUntil:    u.VipCooldownUntil,
		IsRecentlyActive:    ptr(u.IsRecentlyActive),
	}
	if u.Gender != nil {
		g := v1.Gender(*u.Gender)
		item.Gender = &g
	}
	return item
}
