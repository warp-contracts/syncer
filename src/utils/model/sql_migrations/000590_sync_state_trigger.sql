-- +migrate Down
DROP TRIGGER IF EXISTS sync_state_fill_updated_at ON sync_state;

-- +migrate Up

-- +migrate StatementBegin
DO $$ BEGIN IF NOT EXISTS (
    SELECT
        1
    FROM
        pg_trigger
    WHERE
        tgname = 'sync_state_fill_updated_at'
) THEN CREATE TRIGGER sync_state_fill_updated_at
BEFORE
UPDATE
    ON sync_state FOR EACH ROW EXECUTE PROCEDURE fill_updated_at();
END IF;
END $$;
-- +migrate StatementEnd