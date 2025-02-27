-- +migrate Down

-- +migrate Up

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION notify_l2_interaction() RETURNS trigger AS $$
DECLARE
	is_queue_full boolean; 
	is_forwarder_listening boolean;
   	is_too_big boolean;
    src_tx_id text;
	payload text;
BEGIN
	-- Notify only upon L2 changes
	IF NEW.source != 'redstone-sequencer' THEN
		RETURN NEW;
	END IF;

	-- Skip if there's a risk pg_notify would fail
	SELECT pg_notification_queue_usage() > 0.9 INTO is_queue_full;
	IF is_queue_full THEN
		-- pg_notify would fail upon full queue, so let's avoid this situation
		RETURN NEW;
	END IF;

	-- Skip if there's no forwarder listening
	SELECT EXISTS(SELECT pid FROM pg_stat_activity WHERE query='listen "interactions"') INTO is_forwarder_listening;
	IF NOT is_forwarder_listening THEN
		-- Forwarder is down, it will get this interaction when it comes back up
		RETURN NEW;
	END IF;

    -- Get the source tx id
    SELECT contracts.src_tx_id FROM contracts WHERE contracts.contract_id = NEW.contract_id INTO src_tx_id;

	-- Neglect big interactions
	SELECT jsonb_build_object(
            'contractId', NEW.contract_id,
			'interaction', NEW.interaction,
            'srcTxId', src_tx_id
		)::TEXT INTO payload;
	SELECT octet_length(payload) > 7999 INTO is_too_big;

	IF NOT is_too_big THEN
		PERFORM pg_notify('interactions', payload);
	END IF;

	RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd