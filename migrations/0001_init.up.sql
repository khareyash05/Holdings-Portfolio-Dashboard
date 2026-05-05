CREATE TABLE IF NOT EXISTS exchanges (
    short_name VARCHAR(64) PRIMARY KEY,
    currency   VARCHAR(8)  NOT NULL
);

CREATE TABLE IF NOT EXISTS stocks (
    ticker              VARCHAR(64) PRIMARY KEY,
    name                TEXT        NOT NULL,
    exchange_short_name VARCHAR(64) NOT NULL,
    country             VARCHAR(64) NOT NULL,
    currency            VARCHAR(8)  NOT NULL,
    sector              VARCHAR(64) NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_stocks_exchange ON stocks (exchange_short_name);

CREATE TABLE IF NOT EXISTS holdings (
    id                 BIGSERIAL    PRIMARY KEY,
    ticker             VARCHAR(64)  NOT NULL,
    quantity           NUMERIC(18,4) NOT NULL,
    bought_price_local NUMERIC(18,4) NOT NULL,
    bought_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_holdings_ticker ON holdings (ticker);
