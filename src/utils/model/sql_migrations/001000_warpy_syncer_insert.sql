-- +migrate Down
DELETE FROM sync_state WHERE name = 'WarpySyncerBsc';

-- +migrate Up
INSERT INTO sync_state(name, finished_block_height, finished_block_hash) VALUES ('WarpySyncerBsc', 41353910, '0x881367e3abc51a6c04a7513b0abb30a8f391aa45e35864828b4e3c29a2114b05' );