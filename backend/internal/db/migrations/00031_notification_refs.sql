-- +goose Up
-- +goose StatementBegin
-- Records the messages we delivered per (kind, event, provider) so the core
-- can edit them when the originating event changes state through another
-- channel (e.g. a hug accepted on the website edits the Telegram message).
CREATE TABLE notification_refs (
    kind        TEXT        NOT NULL,
    event_id    UUID        NOT NULL,
    provider    TEXT        NOT NULL,
    chat_ref    TEXT        NOT NULL,
    message_ref TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (kind, event_id, provider)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE notification_refs;
-- +goose StatementEnd
