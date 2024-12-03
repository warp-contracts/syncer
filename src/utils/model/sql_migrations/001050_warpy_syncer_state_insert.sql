-- +migrate Down
DELETE FROM sync_state WHERE name = 'WarpySyncerSei';

-- +migrate Up
INSERT INTO sync_state(name, finished_block_height, finished_block_hash) VALUES ('WarpySyncerSei', 118464710, '0xA92EAB828555840D6BC4E14F87176FBAEEDF598DEE6F794877856B9C2B938A90' );