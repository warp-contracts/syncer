-- +migrate Down
DROP TABLE IF EXISTS syncer_state;

-- +migrate Up
DROP TABLE IF EXISTS syncer_state;

CREATE TABLE IF NOT EXISTS sync_state
(
    name synced_component PRIMARY KEY,
    finished_block_height BIGINT NOT NULL,
    finished_block_hash TEXT NOT NULL,
    CONSTRAINT check_block_height CHECK (finished_block_height > 422250)
);

INSERT INTO sync_state(name, finished_block_height, finished_block_hash) VALUES ('Interactions', 1161063, 'xPr-wXT3-0J6OQgnbCNgkid5H-rmaBwvZLtRhd3OAFo_stBTC-Af5kr3IVBnmxQb' );
INSERT INTO sync_state(name, finished_block_height, finished_block_hash) VALUES ('Contracts', 1161063, 'xPr-wXT3-0J6OQgnbCNgkid5H-rmaBwvZLtRhd3OAFo_stBTC-Af5kr3IVBnmxQb' );