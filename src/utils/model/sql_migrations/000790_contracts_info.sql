-- +migrate Down
DROP TABLE IF EXISTS contract_helper;
DROP TABLE IF EXISTS contracts_info;

-- +migrate Up
ALTER TABLE IF EXISTS contract_helper
RENAME TO contracts_info;
