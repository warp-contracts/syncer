-- +migrate Down
ALTER TABLE sync_state DROP COLUMN IF EXISTS additional_data;
-- +migrate Up
ALTER TABLE sync_state ADD COLUMN IF NOT EXISTS additional_data jsonb DEFAULT NULL;
