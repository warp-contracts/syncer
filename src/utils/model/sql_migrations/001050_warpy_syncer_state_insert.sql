-- +migrate Down
DELETE FROM sync_state WHERE name = 'WarpySyncerSei';

-- +migrate Up
INSERT INTO sync_state(name, finished_block_height, finished_block_hash) VALUES ('WarpySyncerSei', 118327937, '0x21E3809F16C15CA04D9A0F347384A1A6ADA91CEDBFBC5EC590174CA3D8BA2D80' );