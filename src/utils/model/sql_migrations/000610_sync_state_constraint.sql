-- +migrate Down

-- +migrate Up
ALTER TABLE sync_state DROP CONSTRAINT check_block_height;