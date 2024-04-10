-- +migrate Down
DROP INDEX IF EXISTS warpy_syncer_assets_from_address_index;
DROP INDEX IF EXISTS warpy_syncer_assets_timestamp_index;

-- +migrate Up
CREATE INDEX IF NOT EXISTS warpy_syncer_assets_from_address_index ON warpy_syncer_assets (from_address);
CREATE INDEX IF NOT EXISTS warpy_syncer_assets_timestamp_index ON warpy_syncer_assets (timestamp DESC NULLS LAST);