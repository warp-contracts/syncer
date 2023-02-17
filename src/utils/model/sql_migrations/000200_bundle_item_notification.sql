-- +migrate Down
DROP FUNCTION IF EXISTS notify_pending_bundle_item;
-- +migrate Up

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION notify_pending_bundle_item() RETURNS trigger AS $$
DECLARE
	is_uploading boolean;
	payload jsonb;
BEGIN
	payload = jsonb_build_object(
		'tx', NEW.transaction,
		'id', NEW.interaction_id
	);

	UPDATE bundle_items 
	SET state = 'UPLOADING'::bundle_state 
	WHERE intearction_id = NEW.intearction_id
	AND state = 'PENDING'::bundle_state
	RETURNING TRUE INTO is_uploading;

	IF is_uploading THEN
		PERFORM pg_notify('bundle_items_pending', payload::TEXT);
	END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd