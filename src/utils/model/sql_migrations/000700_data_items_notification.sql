-- +migrate Down
DROP FUNCTION IF EXISTS notify_pending_data_item;

-- +migrate Up

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION notify_pending_data_item() RETURNS trigger AS $$
DECLARE
	is_queue_full boolean; 
	is_listening boolean;
	is_uploading boolean;
	is_too_big boolean;
	payload text;
BEGIN
	-- Skip if there's a risk pg_notify would fail
	SELECT pg_notification_queue_usage() > 0.95 INTO is_queue_full;
	IF is_queue_full THEN
		-- pg_notify would fail upon full queue, so let's avoid this situation
		-- This bundle item WON'T GET LOST, it will be picked up by the polling job
		RETURN NEW;
	END IF;

	-- Skip if there's no listener
	SELECT EXISTS(SELECT pid FROM pg_stat_activity WHERE query='listen "data_items_pending"') INTO is_listening;
	IF NOT is_listening THEN
		RETURN NEW;
	END IF;

    -- Update state to UPLOADING
	UPDATE data_items 
	SET state = 'UPLOADING'::bundle_state 
	WHERE data_item_id = NEW.data_item_id
	AND state = 'PENDING'::bundle_state
	RETURNING TRUE INTO is_uploading;

	IF NOT is_uploading THEN
		-- TX got selected by the polling mechanism, we're done
		RETURN NEW;
	END IF;

    -- Create the notification
	payload = jsonb_build_object(
			'id', NEW.data_item_id
	)::TEXT;
	
    SELECT LENGTH(payload) > 7999 INTO is_too_big;
	IF is_too_big THEN
        RETURN NEW;
	END IF;
	
    PERFORM pg_notify('data_items_pending', payload::TEXT);

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd