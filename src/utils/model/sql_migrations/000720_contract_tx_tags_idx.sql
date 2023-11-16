-- +migrate Down
DROP INDEX IF EXISTS idx_contract_tx_tags_gin;

-- +migrate Up
CREATE INDEX IF NOT EXISTS idx_contract_tx_tags_gin
    ON contracts USING gin ((contract_tx->'tags') jsonb_path_ops)
WHERE type != 'error';