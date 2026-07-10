-- +goose Up
-- +goose StatementBegin
-- telegram_blocked_at: set on a Telegram 403 send; the notifier skips the
-- user until it is cleared. Cleared when the bot receives any update from
-- them (implies they re-enabled us).
ALTER TABLE users ADD COLUMN telegram_blocked_at TIMESTAMPTZ;
-- daily_reminder_sent_at: last time the daily-reward reminder pinged this
-- user; dedupes within a UTC day.
ALTER TABLE users ADD COLUMN daily_reminder_sent_at TIMESTAMPTZ;
-- matrix_id: the user's Matrix address; NULL until they link Matrix. Enables
-- the dormant Matrix channel without a schema change later.
ALTER TABLE users ADD COLUMN matrix_id TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN matrix_id;
ALTER TABLE users DROP COLUMN daily_reminder_sent_at;
ALTER TABLE users DROP COLUMN telegram_blocked_at;
-- +goose StatementEnd
