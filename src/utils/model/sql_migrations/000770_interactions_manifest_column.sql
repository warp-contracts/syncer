-- +migrate Down
ALTER TABLE interactions DROP COLUMN IF EXISTS manifest;

-- +migrate Up
ALTER TABLE interactions ADD COLUMN IF NOT EXISTS manifest jsonb;