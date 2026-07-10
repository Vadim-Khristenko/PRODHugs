package config

import (
	"fmt"
	"strings"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	HttpSrv    httpServer
	MetricsSrv metricsServer
	Postgres   postgres
	CORS       cors
	S3         s3
	Kafka      kafka
	Valkey     valkey
	JWT        jwt
	Telegram   telegram
	Matrix     matrix
	Notify     notify
}

type telegram struct {
	BotToken    string `env:"TELEGRAM_BOT_TOKEN"`
	BotUsername string `env:"TELEGRAM_BOT_USERNAME"`
}

// matrix holds the credentials for the dormant Matrix notification channel.
// When any field is empty the Matrix provider reports Enabled()==false and is
// skipped everywhere.
type matrix struct {
	HomeserverURL string `env:"MATRIX_HOMESERVER_URL"`
	UserID        string `env:"MATRIX_USER_ID"`
	AccessToken   string `env:"MATRIX_ACCESS_TOKEN"`
}

// notify holds cross-channel notification settings.
type notify struct {
	// DailyReminderHourUTC is the UTC hour (0-23) at which the daily-reward
	// reminder broadcast runs. Default 8 == 11:00 MSK (RF-centric user base;
	// a morning ping, never a late-evening one).
	DailyReminderHourUTC int `env:"DAILY_REMINDER_HOUR_UTC" env-default:"8"`
}

type httpServer struct {
	Addr string `env:"SERVER_ADDR" env-default:":8080"`
}

type metricsServer struct {
	Addr string `env:"METRICS_ADDR" env-default:":9090"`
}

type postgres struct {
	URL             string `env:"POSTGRES_URL" env-required:"true"`
	MaxConns        int32  `env:"POSTGRES_MAX_CONNS" env-default:"100"`
	MinConns        int32  `env:"POSTGRES_MIN_CONNS" env-default:"5"`
	MaxConnLifetime int    `env:"POSTGRES_MAX_CONN_LIFETIME" env-default:"3600"` // seconds
}

type cors struct {
	AllowOrigins []string `env:"CORS_ALLOW_ORIGINS" env-default:"http://localhost:3000,http://localhost:3001" env-separator:","`
}

type s3 struct {
	Region     string `env:"S3_REGION" env-default:"us-east-1"`
	Endpoint   string `env:"S3_ENDPOINT"`
	AccessKey  string `env:"S3_ACCESS_KEY"`
	SecretKey  string `env:"S3_SECRET_KEY"`
	BucketName string `env:"S3_BUCKET_NAME"`
}

type kafka struct {
	Addr          string `env:"KAFKA_ADDR"`
	ReaderGroupID string `env:"KAFKA_READER_GROUP_ID"`
	Topic         string `env:"KAFKA_TOPIC"`
}

type valkey struct {
	Addr     string `env:"VALKEY_ADDR"`
	Password string `env:"VALKEY_PASSWORD"`
}

type jwt struct {
	Secret                  string `env:"JWT_SECRET" env-required:"true"`
	AccessTokenDurationSec  int64  `env:"JWT_ACCESS_DURATION" env-default:"900"`     // 15 minutes
	RefreshTokenDurationSec int64  `env:"JWT_REFRESH_DURATION" env-default:"604800"` // 7 days
	CookieSecure            bool   `env:"JWT_COOKIE_SECURE" env-default:"true"`
}

func New() (*Config, error) {
	var cfg Config

	// Read .env file
	// If failed to read file, will try ReadEnv
	if err := cleanenv.ReadConfig(".env", &cfg); err == nil {
		if err := validateSecurityConfig(&cfg); err != nil {
			return nil, err
		}
		return &cfg, nil
	}

	// Read env
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}

	if err := validateSecurityConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validateSecurityConfig(cfg *Config) error {
	secret := strings.TrimSpace(cfg.JWT.Secret)
	if len(secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}

	weakSecrets := map[string]struct{}{
		"change-me":  {},
		"changeme":   {},
		"jwt-secret": {},
		"secret":     {},
		"hugs-as-a-service-super-secret-jwt-key-2026": {},
	}
	if _, weak := weakSecrets[strings.ToLower(secret)]; weak {
		return fmt.Errorf("JWT_SECRET uses a known weak/default value")
	}

	return nil
}
