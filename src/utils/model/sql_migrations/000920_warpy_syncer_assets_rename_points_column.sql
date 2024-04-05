-- +migrate Down

-- +migrate Up
alter table warpy_syncer_assets rename column points to assets;