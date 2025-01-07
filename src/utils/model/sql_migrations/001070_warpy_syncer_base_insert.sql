-- +migrate Down
DELETE FROM sync_state WHERE name = 'WarpySyncerBase';

-- +migrate Up
INSERT INTO sync_state(name, finished_block_height, finished_block_hash) VALUES ('WarpySyncerBase', 24307450, '0xb758290fa1ab94c595afe984d9534fede934d368882ef322f65b9cf542a1f431' );