package user

import (
	"context"
	"go-service-template/internal/db/sqlc/storage"
	"go-service-template/internal/models"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
)

type repo interface {
	Create(ctx context.Context, input *models.CreateUser) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByTelegramID(ctx context.Context, telegramID int64) (*models.User, error)
	UpdateSettings(ctx context.Context, id uuid.UUID, gender *string, displayName *string, tag *string) (*models.User, error)
	AdminUpdateTag(ctx context.Context, id uuid.UUID, tag *string) (*models.User, error)
	AdminUpdateSpecialTag(ctx context.Context, id uuid.UUID, specialTag *string) (*models.User, error)
	GetTelegramID(ctx context.Context, userID uuid.UUID) (*int64, error)
	SetTelegramID(ctx context.Context, userID uuid.UUID, telegramID int64) (*models.User, error)
	ClearTelegramID(ctx context.Context, userID uuid.UUID) (*models.User, error)
	GetByMatrixID(ctx context.Context, matrixID string) (*models.User, error)
	SetMatrixLink(ctx context.Context, userID uuid.UUID, matrixID, roomID string) error
	ClearMatrixLink(ctx context.Context, userID uuid.UUID) error
	IsTelegramIDTaken(ctx context.Context, telegramID int64, excludeUserID uuid.UUID) (bool, error)
	UpdatePassword(ctx context.Context, id uuid.UUID, hashedPassword string) error
	BanUser(ctx context.Context, id uuid.UUID) (*models.User, error)
	UnbanUser(ctx context.Context, id uuid.UUID) (*models.User, error)
	CountUsers(ctx context.Context) (int64, error)
	CountBannedUsers(ctx context.Context) (int64, error)
	ListUsersAdmin(ctx context.Context, limit, offset int32) ([]*models.AdminUser, error)
	AdminUpdateUsername(ctx context.Context, id uuid.UUID, username string) (*models.User, error)
	AdminUpdateGender(ctx context.Context, id uuid.UUID, gender *string) (*models.User, error)
	AdminUpdatePassword(ctx context.Context, id uuid.UUID, hashedPassword string) error
	AdminUpdateDisplayName(ctx context.Context, id uuid.UUID, displayName *string) (*models.User, error)
	SearchUsersAdmin(ctx context.Context, query string, limit, offset int32) ([]*models.AdminUser, error)
	AdminDeleteUser(ctx context.Context, id uuid.UUID) error
	ClearExpiredPromotions(ctx context.Context) (int64, error)
	ListVIPUsers(ctx context.Context) ([]*models.User, error)
	AdminUpdateCaptchaType(ctx context.Context, id uuid.UUID, captchaType string) (*models.User, error)
	AdminClearPromotion(ctx context.Context, id uuid.UUID) (*models.User, error)
	PromoteUser(ctx context.Context, id uuid.UUID, promotedUntil time.Time, message *string, bid int32) (*models.User, error)
	SetVipCooldown(ctx context.Context, id uuid.UUID, cooldownUntil time.Time, remainingSeconds int32) (*models.User, error)
	UpdateVipBudget(ctx context.Context, id uuid.UUID, remainingSeconds int32) (*models.User, error)

	CreateSudokuCaptcha(ctx context.Context, userID uuid.UUID, puzzle []byte, solution []byte, expiresAt time.Time) (storage.SudokuCaptcha, error)
	GetSudokuCaptcha(ctx context.Context, id uuid.UUID) (storage.SudokuCaptcha, error)
	IncrementSudokuErrors(ctx context.Context, id uuid.UUID) (storage.SudokuCaptcha, error)
	MarkSudokuPassed(ctx context.Context, id uuid.UUID) (storage.SudokuCaptcha, error)
	DeleteSudokuCaptcha(ctx context.Context, id uuid.UUID) error
	CreateCasinoCaptcha(ctx context.Context, userID uuid.UUID, expiresAt time.Time) (storage.CasinoCaptcha, error)
	GetCasinoCaptcha(ctx context.Context, id uuid.UUID) (storage.CasinoCaptcha, error)
	MarkCasinoPassed(ctx context.Context, id uuid.UUID) (storage.CasinoCaptcha, error)
	DeleteCasinoCaptcha(ctx context.Context, id uuid.UUID) error
	SetCaptchaCooldown(ctx context.Context, userID uuid.UUID, cooldownUntil time.Time) error
}

type balanceRepo interface {
	AdminSetBalance(ctx context.Context, userID uuid.UUID, amount int32) (*models.Balance, error)
	DeductBalance(ctx context.Context, userID uuid.UUID, delta int32) (*models.Balance, error)
}

type transactor interface {
	RunInTx(ctx context.Context, fn func(context.Context) error) error
}

type refreshTokenRepo interface {
	SaveRefreshToken(ctx context.Context, jti string, userID uuid.UUID, expiresAtUnix int64) error
	IsRefreshTokenActive(ctx context.Context, jti string) (bool, error)
	RevokeRefreshToken(ctx context.Context, jti string) error
	RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error
}

type jwtManager interface {
	GenerateAccessToken(userID uuid.UUID, role string) (string, int64, error)
	GenerateRefreshToken(userID uuid.UUID) (string, string, int64, error)
	GenerateCaptchaToken(userID uuid.UUID) (string, error)
	ParseCaptchaToken(tokenString string) (uuid.UUID, error)
}

type telegramLinkStore interface {
	GenerateToken(userID uuid.UUID) (string, error)
}

type matrixLinkStore interface {
	GenerateToken(userID uuid.UUID) (string, error)
}

type matrixLoginStore interface {
	CreateSession() (botToken string, pollToken string, err error)
}

type announcementRepo interface {
	GetActiveForUser(ctx context.Context, userID uuid.UUID) (*models.Announcement, error)
	GetActive(ctx context.Context) (*models.Announcement, error)
	Create(ctx context.Context, message string, createdBy uuid.UUID) (*models.Announcement, error)
	Deactivate(ctx context.Context, id uuid.UUID) error
	Dismiss(ctx context.Context, announcementID, userID uuid.UUID) error
}

// AnnouncementCallback is called when an announcement is created or removed.
type AnnouncementCallback func(announcement *models.Announcement)
type AnnouncementRemovedCallback func(id uuid.UUID)
type PromotionUpdatedCallback func()

type service struct {
	repo                 repo
	balanceRepo          balanceRepo
	refreshTokenRepo     refreshTokenRepo
	jwtManager           jwtManager
	telegramLinkStore    telegramLinkStore
	matrixLinkStore      matrixLinkStore
	matrixLoginStore     matrixLoginStore
	announcementRepo     announcementRepo
	botUsername          string
	telegramBotToken     string
	matrixBotUserID      string
	matrixLoginBotUserID string
	tx                   transactor
	rng                  *rand.Rand

	onAnnouncementCreated AnnouncementCallback
	onAnnouncementRemoved AnnouncementRemovedCallback
	onPromotionUpdated    PromotionUpdatedCallback
}

func New(repo repo, jwtManager jwtManager, opts ...func(*service)) *service {
	s := &service{
		repo:       repo,
		jwtManager: jwtManager,
		rng:        rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0)),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// WithBalanceRepo sets the balance repository for admin operations.
func WithBalanceRepo(br balanceRepo) func(*service) {
	return func(s *service) {
		s.balanceRepo = br
	}
}

func WithTransactor(tx transactor) func(*service) {
	return func(s *service) {
		s.tx = tx
	}
}

func WithRefreshTokenRepo(rtr refreshTokenRepo) func(*service) {
	return func(s *service) {
		s.refreshTokenRepo = rtr
	}
}

func WithAnnouncementRepo(ar announcementRepo) func(*service) {
	return func(s *service) {
		s.announcementRepo = ar
	}
}

// SetTelegramLinkStore configures the Telegram link store and bot username
// for deep-link token generation. Called after construction to break circular deps.
func (s *service) SetTelegramLinkStore(ls telegramLinkStore, botUsername string) {
	s.telegramLinkStore = ls
	s.botUsername = botUsername
}

// SetTelegramBotToken configures the Telegram bot token used to verify Login
// Widget payloads. Called after construction to break circular deps.
func (s *service) SetTelegramBotToken(token string) {
	s.telegramBotToken = token
}

// SetMatrixLinkStore configures the Matrix link store and bot user id for
// link-token generation. Called after construction to break circular deps.
func (s *service) SetMatrixLinkStore(ls matrixLinkStore, botUserID string) {
	s.matrixLinkStore = ls
	s.matrixBotUserID = botUserID
}

// SetMatrixLoginStore configures the Matrix login store and bot user id used by
// the Matrix bot-login flow. Called after construction to break circular deps.
func (s *service) SetMatrixLoginStore(ls matrixLoginStore, botUserID string) {
	s.matrixLoginStore = ls
	s.matrixLoginBotUserID = botUserID
}

func (s *service) SetAnnouncementCreatedCallback(cb AnnouncementCallback) {
	s.onAnnouncementCreated = cb
}

func (s *service) SetAnnouncementRemovedCallback(cb AnnouncementRemovedCallback) {
	s.onAnnouncementRemoved = cb
}

func (s *service) SetPromotionUpdatedCallback(cb PromotionUpdatedCallback) {
	s.onPromotionUpdated = cb
}
