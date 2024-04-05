-- +migrate Down

-- +migrate Up
ALTER TYPE synced_component RENAME VALUE 'WarpyChainer' TO 'WarpySyncer';
