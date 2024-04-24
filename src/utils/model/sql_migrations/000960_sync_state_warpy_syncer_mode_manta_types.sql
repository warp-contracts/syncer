-- +migrate Down

-- +migrate Up
ALTER TYPE synced_component ADD VALUE 'WarpySyncerMode';
ALTER TYPE synced_component ADD VALUE 'WarpySyncerManta';