-- +goose Up
-- +goose StatementBegin
-- Mirror the telegram_id unique partial index: enforce that a Matrix account
-- links to at most one user (integrity, backing the app-level IsMatrixIDTaken
-- check against races) and make that lookup an index hit instead of a scan.
CREATE UNIQUE INDEX users_matrix_id_key ON users (matrix_id) WHERE matrix_id IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS users_matrix_id_key;
-- +goose StatementEnd
