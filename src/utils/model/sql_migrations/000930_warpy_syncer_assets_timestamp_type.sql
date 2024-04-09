-- +migrate Down

-- +migrate Up
alter table warpy_syncer_assets alter column timestamp type varchar(255);