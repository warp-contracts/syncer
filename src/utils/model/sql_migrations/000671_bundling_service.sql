-- +migrate Down
DROP TYPE IF EXISTS bundling_service;

-- +migrate Up
-- +migrate StatementBegin
DO $$ BEGIN IF NOT EXISTS (
    SELECT
        1
    FROM
        pg_type
    WHERE
        typname = 'bundling_service'
) THEN CREATE TYPE bundling_service AS ENUM (
    'IRYS',
    'TURBO'
);

END IF;
END $$;

-- +migrate StatementEnd