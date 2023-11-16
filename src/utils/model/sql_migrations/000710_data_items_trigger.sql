-- +migrate Down
DROP TRIGGER IF EXISTS data_items_notify_insert ON data_items;

-- +migrate Up

-- +migrate StatementBegin
DO $$ BEGIN IF NOT EXISTS (
    SELECT
        1
    FROM
        pg_trigger
    WHERE
        tgname = 'data_items_notify_insert'
) THEN CREATE TRIGGER data_items_notify_insert
AFTER
INSERT
    ON data_items FOR EACH ROW EXECUTE PROCEDURE notify_pending_data_item();
END IF;
END $$;
-- +migrate StatementEnd