-- +migrate Down

-- +migrate Up
alter table warpy_syncer_points rename to warpy_syncer_assets;