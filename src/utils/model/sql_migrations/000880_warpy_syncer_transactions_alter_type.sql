-- +migrate Down

-- +migrate Up
ALTER TABLE warpy_syncer_transactions ALTER COLUMN block_timestamp TYPE BIGINT USING EXTRACT(EPOCH FROM block_timestamp);
ALTER TABLE warpy_syncer_transactions ALTER COLUMN sync_timestamp TYPE BIGINT USING EXTRACT(EPOCH FROM sync_timestamp);
