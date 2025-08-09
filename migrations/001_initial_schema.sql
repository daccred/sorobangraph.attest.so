-- Create database schema for Stellar ingester

-- Ledgers table
CREATE TABLE IF NOT EXISTS ledgers (
    sequence BIGINT PRIMARY KEY,
    hash VARCHAR(64) NOT NULL,
    previous_hash VARCHAR(64) NOT NULL,
    transaction_count INTEGER NOT NULL DEFAULT 0,
    operation_count INTEGER NOT NULL DEFAULT 0,
    closed_at TIMESTAMP NOT NULL,
    total_coins BIGINT,
    fee_pool BIGINT,
    base_fee INTEGER,
    base_reserve INTEGER,
    max_tx_set_size INTEGER,
    protocol_version INTEGER,
    ledger_header JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_ledgers_closed_at ON ledgers(closed_at DESC);

-- Transactions table
CREATE TABLE IF NOT EXISTS transactions (
    id VARCHAR(255) PRIMARY KEY,
    hash VARCHAR(64) NOT NULL UNIQUE,
    ledger BIGINT NOT NULL,
    index INTEGER NOT NULL,
    source_account VARCHAR(56) NOT NULL,
    fee_paid BIGINT NOT NULL,
    operation_count INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    memo_type VARCHAR(20),
    memo_value TEXT,
    successful BOOLEAN NOT NULL DEFAULT true,
    envelope_xdr BYTEA,
    result_xdr BYTEA,
    result_meta_xdr BYTEA,
    FOREIGN KEY (ledger) REFERENCES ledgers(sequence) ON DELETE CASCADE
);

CREATE INDEX idx_transactions_hash ON transactions(hash);
CREATE INDEX idx_transactions_ledger ON transactions(ledger DESC);
CREATE INDEX idx_transactions_source_account ON transactions(source_account);
CREATE INDEX idx_transactions_created_at ON transactions(created_at DESC);

-- Operations table
CREATE TABLE IF NOT EXISTS operations (
    id VARCHAR(255) PRIMARY KEY,
    transaction_id VARCHAR(255) NOT NULL,
    index INTEGER NOT NULL,
    type VARCHAR(50) NOT NULL,
    source_account VARCHAR(56),
    details JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE CASCADE
);

CREATE INDEX idx_operations_transaction_id ON operations(transaction_id);
CREATE INDEX idx_operations_type ON operations(type);
CREATE INDEX idx_operations_source_account ON operations(source_account);

-- Contract events table (for Soroban)
CREATE TABLE IF NOT EXISTS contract_events (
    id VARCHAR(255) PRIMARY KEY,
    contract_id VARCHAR(64),
    ledger BIGINT NOT NULL,
    transaction_hash VARCHAR(64) NOT NULL,
    event_type VARCHAR(20) NOT NULL,
    topics JSONB,
    data JSONB,
    in_successful_tx BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (ledger) REFERENCES ledgers(sequence) ON DELETE CASCADE
);

CREATE INDEX idx_contract_events_contract_id ON contract_events(contract_id);
CREATE INDEX idx_contract_events_ledger ON contract_events(ledger DESC);
CREATE INDEX idx_contract_events_transaction_hash ON contract_events(transaction_hash);
CREATE INDEX idx_contract_events_event_type ON contract_events(event_type);

-- Ingestion state table (tracks progress)
CREATE TABLE IF NOT EXISTS ingestion_state (
    id INTEGER PRIMARY KEY DEFAULT 1,
    last_ledger BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT single_row CHECK (id = 1)
);

-- Insert default ingestion state
INSERT INTO ingestion_state (id, last_ledger, updated_at) 
VALUES (1, 0, NOW()) 
ON CONFLICT (id) DO NOTHING;

-- Accounts table (optional, for tracking account states)
CREATE TABLE IF NOT EXISTS accounts (
    account_id VARCHAR(56) PRIMARY KEY,
    sequence BIGINT NOT NULL,
    balance BIGINT NOT NULL DEFAULT 0,
    buying_liabilities BIGINT DEFAULT 0,
    selling_liabilities BIGINT DEFAULT 0,
    num_subentries INTEGER DEFAULT 0,
    num_sponsoring INTEGER DEFAULT 0,
    num_sponsored INTEGER DEFAULT 0,
    last_modified_ledger BIGINT,
    thresholds JSONB,
    flags INTEGER,
    home_domain VARCHAR(255),
    master_weight INTEGER,
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_accounts_last_modified ON accounts(last_modified_ledger DESC);

-- Assets table (optional, for tracking assets)
CREATE TABLE IF NOT EXISTS assets (
    id SERIAL PRIMARY KEY,
    asset_type VARCHAR(20) NOT NULL,
    asset_code VARCHAR(12),
    asset_issuer VARCHAR(56),
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(asset_type, asset_code, asset_issuer)
);

CREATE INDEX idx_assets_code ON assets(asset_code);
CREATE INDEX idx_assets_issuer ON assets(asset_issuer);

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at
CREATE TRIGGER update_ingestion_state_updated_at BEFORE UPDATE ON ingestion_state 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_accounts_updated_at BEFORE UPDATE ON accounts 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();