-- +goose Up
-- +goose StatementBegin
-- matrix_room_id: the direct-message room the bot uses to deliver Matrix
-- notifications to this user. Set together with matrix_id when the user links
-- their Matrix account by DMing the bot. NULL until linked.
ALTER TABLE users ADD COLUMN matrix_room_id TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN matrix_room_id;
-- +goose StatementEnd
