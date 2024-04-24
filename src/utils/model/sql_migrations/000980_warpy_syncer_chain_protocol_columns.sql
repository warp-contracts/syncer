-- +migrate Down
ALTER TABLE warpy_syncer_transactions DROP COLUMN IF EXISTS protocol;
ALTER TABLE warpy_syncer_assets DROP COLUMN IF EXISTS chain;

-- +migrate Up
ALTER TABLE warpy_syncer_transactions ADD COLUMN IF NOT EXISTS protocol VARCHAR(64);
ALTER TABLE warpy_syncer_assets ADD COLUMN IF NOT EXISTS chain VARCHAR(64);