-- +migrate Down
DROP TABLE IF EXISTS warpy_syncer_points;

-- +migrate Up
CREATE TABLE warpy_syncer_points (
	tx_id TEXT PRIMARY KEY NOT NULL,
	from_address TEXT NOT NULL,
	points INT NOT NULL,
	timestamp INT NOT NULL,
	protocol VARCHAR(64) NOT NULL
);