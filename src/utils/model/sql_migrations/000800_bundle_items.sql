-- +migrate Down
ALTER TABLE bundle_items DROP COLUMN IF EXISTS service;

-- +migrate Up
ALTER TABLE bundle_items ADD COLUMN IF NOT EXISTS service bundling_service;

