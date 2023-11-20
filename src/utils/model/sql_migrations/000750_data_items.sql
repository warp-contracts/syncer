-- +migrate Down

-- +migrate Up
ALTER TABLE data_items ALTER COLUMN service DROP NOT NULL;
