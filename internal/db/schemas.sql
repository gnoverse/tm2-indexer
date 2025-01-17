
CREATE TABLE IF NOT EXISTS validators (
    validator_addr VARCHAR(90)  PRIMARY KEY,
    validator_name VARCHAR(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS blocks (
    height      INT       UNIQUE PRIMARY KEY,
    time        TIMESTAMP NOT NULL,

    hash               VARCHAR(255) NOT NULL,

    num_txs            BIGINT      NOT NULL,
    total_txs          BIGINT      NOT NULL,
    app_version        VARCHAR(25) NOT NULL,

    data_hash            TEXT,
    last_commit_hash     VARCHAR(255),
    validators_hash      VARCHAR(255),
    next_validators_hash VARCHAR(255),
    consensus_hash       VARCHAR(255),
    app_hash             VARCHAR NOT NULL,
    last_results_hash    VARCHAR NOT NULL,

    proposer_address     VARCHAR(90) NOT NULL REFERENCES validators(validator_addr)
);

CREATE TABLE IF NOT EXISTS block_signatures (
    block_height   INT         NOT NULL REFERENCES blocks(height),
    validator_addr VARCHAR(90) NOT NULL REFERENCES validators(validator_addr),
    signed         BOOL        NOT NULL DEFAULT FALSE,
    PRIMARY KEY (block_height, validator_addr)
);

CREATE TABLE IF NOT EXISTS transactions (
    height       INT NOT NULL REFERENCES blocks(height),
    tx_index     INT NOT NULL,

    hash         TEXT         UNIQUE NOT NULL,
    gas_fee      INT          NOT NULL,
    gas_wanted   INT          NOT NULL,

    memo         TEXT,

    UNIQUE (height, tx_index),
    PRIMARY KEY (height, tx_index)
);

CREATE TABLE IF NOT EXISTS messages (
    height       INT          NOT NULL REFERENCES blocks(height),
    tx_hash      TEXT         NOT NULL REFERENCES transactions(hash),
    index        INT          NOT NULL,

    route        VARCHAR(255) NOT NULL,
    type         VARCHAR(255) NOT NULL,

    msg_raw      JSONB NOT NULL,

    UNIQUE (height, tx_hash, index),
    PRIMARY KEY (height, tx_hash, index)
);

-- TODO: create a table events for all the emits data from BlockResults

CREATE INDEX IF NOT EXISTS idx_blocks_time                     ON blocks(time);
CREATE INDEX IF NOT EXISTS idx_block_signatures_block_height   ON block_signatures(block_height);
CREATE INDEX IF NOT EXISTS idx_block_signatures_validator_addr ON block_signatures(validator_addr);


CREATE INDEX IF NOT EXISTS idx_transactions_height_tx_index ON transactions(height, tx_index);
CREATE INDEX IF NOT EXISTS idx_transactions_hash            ON transactions(hash);
CREATE INDEX IF NOT EXISTS idx_messages_height_tx_hash      ON messages(height, tx_hash);
CREATE INDEX IF NOT EXISTS idx_messages_tx_hash_index       ON messages(tx_hash, index);

-- gave read only access to grafana user for dashboards
-- GRANT USAGE ON SCHEMA public to grafana;
-- GRANT SELECT ON ALL TABLES IN SCHEMA public TO grafana;
