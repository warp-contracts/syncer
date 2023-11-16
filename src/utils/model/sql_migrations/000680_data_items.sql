-- +migrate Down
DROP TABLE IF EXISTS data_items;

-- +migrate Up
CREATE TABLE IF NOT EXISTS data_items (
    -- Id as in the data item
    data_item_id TEXT NOT NULL PRIMARY KEY,

    -- State of the bundle
    state bundle_state NOT NULL,

    -- Service used to bundle the data item
    service bundling_service NOT NULL,

    -- Response from the bundler service
    response jsonb,

    -- Block height upon which interaction was bundled. Used to trigger verification later
    block_height bigint,

    -- Time of the last update to this row
    updated_at timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Time of creation
    created_at timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Data item that will be bundled
    data_item bytea,

    -- Block height
    CONSTRAINT block_height_for_uploaded CHECK (
        state <> 'UPLOADED' :: bundle_state
        OR state = 'UPLOADED' :: bundle_state
        AND block_height IS NOT NULL
    ),

    CONSTRAINT block_height_for_on_arweave CHECK (
        state <> 'ON_ARWEAVE' :: bundle_state
        OR state = 'ON_ARWEAVE' :: bundle_state
        AND block_height IS NOT NULL
    ),

    CONSTRAINT block_height_for_checking CHECK (
        state <> 'CHECKING' :: bundle_state
        OR state = 'CHECKING' :: bundle_state
        AND block_height IS NOT NULL
    ),

    -- Data item
    CONSTRAINT data_item_for_pending CHECK (
        state <> 'PENDING' :: bundle_state
        OR state = 'PENDING' :: bundle_state
        AND data_item IS NOT NULL
    ),
    CONSTRAINT data_item_for_uploading CHECK (
        state <> 'UPLOADING' :: bundle_state
        OR state = 'UPLOADING' :: bundle_state
        AND data_item IS NOT NULL
    ),
    CONSTRAINT data_item_for_uploaded CHECK (
        state <> 'UPLOADED' :: bundle_state
        OR state = 'UPLOADED' :: bundle_state
        AND data_item IS NOT NULL
    ),

    -- Response
    CONSTRAINT response_for_uploaded CHECK (
        state <> 'UPLOADED' :: bundle_state
        OR state = 'UPLOADED' :: bundle_state
        AND response IS NOT NULL
    ),
    CONSTRAINT response_for_on_bundler CHECK (
        state <> 'ON_ARWEAVE' :: bundle_state
        OR state = 'ON_ARWEAVE' :: bundle_state
        AND response IS NOT NULL
    ),
    CONSTRAINT response_format CHECK (
        response IS NULL
        OR response ? 'id' :: text
    )
);