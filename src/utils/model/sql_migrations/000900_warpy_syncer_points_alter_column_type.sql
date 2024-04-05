-- +migrate Down

-- +migrate Up
alter table warpy_syncer_points alter column points type FLOAT;