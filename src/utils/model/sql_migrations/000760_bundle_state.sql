-- +migrate Down
ALTER TYPE bundle_state DROP VALUE 'DUPLICATE';

-- +migrate Up
ALTER TYPE bundle_state ADD VALUE 'DUPLICATE';
