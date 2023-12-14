-- +migrate Down
DROP TABLE IF EXISTS contract_helper;

-- +migrate Up
DROP TABLE IF EXISTS contract_helper;

CREATE TABLE IF NOT EXISTS contract_helper AS
SELECT contract_id,
       min(sort_key) as min_sk
FROM interactions
WHERE contract_id <> ''
GROUP BY contract_id;
ALTER TABLE contract_helper ADD PRIMARY KEY (contract_id);
