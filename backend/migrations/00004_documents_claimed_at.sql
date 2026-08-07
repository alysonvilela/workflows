-- +goose Up
ALTER TABLE documents ADD COLUMN claimed_at timestamptz;

-- +goose Down
ALTER TABLE documents DROP COLUMN claimed_at;
