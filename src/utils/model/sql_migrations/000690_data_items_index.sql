-- +migrate Down
DROP INDEX IF EXISTS idx_data_items_state;
DROP INDEX IF EXISTS idx_data_items_poller_uploading_updated_at;
DROP INDEX IF EXISTS idx_data_items_poller_pending_state;
DROP INDEX IF EXISTS idx_data_items_poller_malformed_updated_at;
DROP INDEX IF EXISTS idx_data_items_checker_uploaded_height;
DROP INDEX IF EXISTS idx_data_items_checker_checking_updated_at;

-- +migrate Up

-- Checker getting oldest items that are being checked.
-- This is for retrying checking
CREATE INDEX IF NOT EXISTS idx_data_items_checker_checking_updated_at
    ON data_items USING btree
    (updated_at DESC NULLS FIRST)
    INCLUDE(data_item_id)
    WHERE state = 'CHECKING'::bundle_state;

-- Chegger getting uploaded items that are ready to be checked
CREATE INDEX IF NOT EXISTS idx_data_items_checker_uploaded_height
    ON data_items USING btree
    (block_height ASC NULLS LAST)
    INCLUDE(data_item_id)
    WHERE state = 'UPLOADED'::bundle_state;

-- Getting newest malformed items. Not used in any component.
CREATE INDEX IF NOT EXISTS idx_data_items_poller_malformed_updated_at
    ON data_items USING brin
    (updated_at)
    WHERE state = 'MALFORMED'::bundle_state;

-- Getting pending items that are ready to be uploaded
CREATE INDEX IF NOT EXISTS idx_data_items_poller_pending_state
    ON data_items USING btree
    (updated_at ASC NULLS LAST)
    WHERE state = 'PENDING'::bundle_state;

-- Getting oldest items that are being uploaded. This is for retrying uploading
CREATE INDEX IF NOT EXISTS idx_data_items_poller_uploading_updated_at
    ON data_items USING btree
    (updated_at DESC NULLS FIRST)
    WHERE state = 'UPLOADING'::bundle_state;

-- Four counting items in each state, not used in any component
CREATE INDEX IF NOT EXISTS idx_data_items_state
    ON data_items USING btree
    (state ASC NULLS LAST);
