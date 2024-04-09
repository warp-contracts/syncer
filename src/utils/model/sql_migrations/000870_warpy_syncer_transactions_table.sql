-- +migrate Down
DROP TABLE warpy_syncer_transactions;

-- +migrate Up
CREATE TABLE IF NOT EXISTS warpy_syncer_transactions (
	tx_id TEXT PRIMARY KEY NOT NULL,
	from_address TEXT NOT NULL,
	to_address TEXT NOT NULL,
	block_height BIGINT NOT NULL,
	block_timestamp TIMESTAMP NOT NULL,
	sync_timestamp TIMESTAMP NOT NULL,
	method_name VARCHAR(64) NOT NULL,
	chain VARCHAR(64) NOT NULL,
	input jsonb
);
