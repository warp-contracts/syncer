-- +migrate Down
ALTER TABLE warpy_syncer_assets DROP COLUMN IF EXISTS asset_factor;

-- +migrate Up
ALTER TABLE warpy_syncer_assets ADD COLUMN IF NOT EXISTS asset_factor double precision not null default 1;
