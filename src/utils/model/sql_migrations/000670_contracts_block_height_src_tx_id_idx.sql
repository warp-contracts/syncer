-- +migrate Down
DROP INDEX IF EXISTS contracts_block_height_src_tx_id_index;

-- +migrate Up
CREATE INDEX IF NOT EXISTS contracts_block_height_src_tx_id_index ON contracts (block_height, src_tx_id);