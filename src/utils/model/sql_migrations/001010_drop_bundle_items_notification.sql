-- +migrate Down

-- +migrate Up
DROP TRIGGER IF EXISTS bundle_items_notify_insert ON bundle_items;
