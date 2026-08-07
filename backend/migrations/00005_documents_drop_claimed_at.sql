-- +goose Up
ALTER TABLE documents DROP COLUMN claimed_at;

-- +goose Down
ALTER TABLE documents ADD COLUMN claimed_at timestamptz;
