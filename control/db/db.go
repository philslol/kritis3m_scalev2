package db

// Schema definitions
const createTablesSQL = `
-- Create ENUM types for status fields and proxy types
CREATE TYPE transaction_status AS ENUM ('pending', 'active', 'failed', 'rollback');
CREATE TYPE operation_type AS ENUM ('INSERT', 'UPDATE', 'DELETE', 'ADD');
CREATE TYPE proxy_type AS ENUM ('FORWARD', 'REVERSE', 'TLSTLS');

CREATE TYPE asl_key_exchange_method AS ENUM (
    'ASL_KEX_DEFAULT',
    'ASL_KEX_CLASSIC_SECP256',
    'ASL_KEX_CLASSIC_SECP384',
    'ASL_KEX_CLASSIC_SECP521',
    'ASL_KEX_CLASSIC_X25519',
    'ASL_KEX_CLASSIC_X448',
    'ASL_KEX_PQC_MLKEM512',
    'ASL_KEX_PQC_MLKEM768',
    'ASL_KEX_PQC_MLKEM1024',
    'ASL_KEX_HYBRID_SECP256_MLKEM512',
    'ASL_KEX_HYBRID_SECP384_MLKEM768',
    'ASL_KEX_HYBRID_SECP256_MLKEM768',
    'ASL_KEX_HYBRID_SECP521_MLKEM1024',
    'ASL_KEX_HYBRID_SECP384_MLKEM1024',
    'ASL_KEX_HYBRID_X25519_MLKEM512',
    'ASL_KEX_HYBRID_X448_MLKEM768',
    'ASL_KEX_HYBRID_X25519_MLKEM768'
);


-- Transactions track all changes
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status transaction_status NOT NULL DEFAULT 'pending',  -- Use ENUM for status
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    created_by TEXT NOT NULL,
    description TEXT,
    metadata JSONB
);

-- Change log tracks all modifications
CREATE TABLE IF NOT EXISTS change_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID REFERENCES transactions(id),
    table_name TEXT NOT NULL,
    record_id TEXT NOT NULL,  -- Changed to TEXT to support UUID and INTEGER
    operation operation_type NOT NULL,  -- Use ENUM for operation
    old_data JSONB,
    new_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL
);

-- Nodes are the base entity
CREATE TABLE IF NOT EXISTS nodes (
    id SERIAL PRIMARY KEY,
    serial_number TEXT UNIQUE NOT NULL,
    network_index INTEGER NOT NULL,
    locality TEXT,
    last_seen TIMESTAMPTZ,
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL,
    CONSTRAINT valid_serial CHECK (serial_number ~ '^[A-Za-z0-9-]{1,50}$')  -- Added length constraint
);

-- EndpointConfigs define connection parameters
CREATE TABLE IF NOT EXISTS endpoint_configs (
    id SERIAL PRIMARY KEY,
    transaction_id UUID REFERENCES transactions(id),
    name TEXT NOT NULL,
    mutual_auth BOOLEAN NOT NULL DEFAULT false,
    no_encryption BOOLEAN NOT NULL DEFAULT false,
	asl_key_exchange_method asl_key_exchange_method NOT NULL DEFAULT 'ASL_KEX_DEFAULT',
    cipher TEXT,
    status transaction_status NOT NULL DEFAULT 'pending',  -- Use ENUM for status
    version INTEGER NOT NULL DEFAULT 1,
    previous_version_id INTEGER REFERENCES endpoint_configs(id) NULL,  -- Allow NULL for first version
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL
);

-- Groups organize configurations
CREATE TABLE IF NOT EXISTS groups (
    id SERIAL PRIMARY KEY,
    transaction_id UUID REFERENCES transactions(id),
    name TEXT NOT NULL,
    log_level INTEGER NOT NULL DEFAULT 0,
    endpoint_config_id INTEGER REFERENCES endpoint_configs(id),
    legacy_config_id INTEGER REFERENCES endpoint_configs(id),
    status transaction_status NOT NULL DEFAULT 'pending',  -- Use ENUM for status
    version INTEGER NOT NULL DEFAULT 1,
    previous_version_id INTEGER REFERENCES groups(id) NULL,  -- Allow NULL for first version
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL
);

-- Hardware configurations for nodes
CREATE TABLE IF NOT EXISTS hardware_configs (
    id SERIAL PRIMARY KEY,
    node_id INTEGER REFERENCES nodes(id),
    transaction_id UUID REFERENCES transactions(id),
    device TEXT NOT NULL,
    ip_cidr INET NOT NULL,  -- Use INET type for better CIDR handling
    status transaction_status NOT NULL DEFAULT 'pending',  -- Use ENUM for status
    version INTEGER NOT NULL DEFAULT 1,
    previous_version_id INTEGER REFERENCES hardware_configs(id) NULL,  -- Allow NULL for first version
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL
);

-- Proxies connect nodes to groups
CREATE TABLE IF NOT EXISTS proxies (
    id SERIAL PRIMARY KEY,
    transaction_id UUID REFERENCES transactions(id),
    node_id INTEGER REFERENCES nodes(id),
    group_id INTEGER REFERENCES groups(id),
    state BOOLEAN NOT NULL DEFAULT true,
    proxy_type proxy_type NOT NULL,  -- Use ENUM for proxy type
    server_endpoint_addr TEXT NOT NULL,
    client_endpoint_addr TEXT NOT NULL,
    status transaction_status NOT NULL DEFAULT 'pending',  -- Use ENUM for status
    version INTEGER NOT NULL DEFAULT 1,
    previous_version_id INTEGER REFERENCES proxies(id) NULL,  -- Allow NULL for first version
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL
);

-- Trigger function for change logging
CREATE OR REPLACE FUNCTION log_changes()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO change_log (
        transaction_id,
        table_name,
        record_id,
        operation,
        old_data,
        new_data,
        created_by
    )
    VALUES (
        COALESCE(current_setting('app.current_transaction_id', TRUE)::UUID, NULL),
        TG_TABLE_NAME,
        CASE 
            WHEN TG_OP = 'DELETE' THEN OLD.id::TEXT 
            ELSE NEW.id::TEXT 
        END,
        TG_OP::operation_type,  -- Cast to ENUM type
        CASE 
            WHEN TG_OP = 'DELETE' THEN TO_JSONB(OLD)
            WHEN TG_OP = 'UPDATE' THEN TO_JSONB(OLD)
            ELSE NULL 
        END,
        CASE 
            WHEN TG_OP = 'DELETE' THEN NULL
            ELSE TO_JSONB(NEW)
        END,
        COALESCE(current_setting('app.current_user', TRUE), 'system')
    );
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Triggers for change logging
CREATE TRIGGER log_nodes_changes
    AFTER INSERT OR UPDATE OR DELETE ON nodes
    FOR EACH ROW EXECUTE FUNCTION log_changes();

CREATE TRIGGER log_endpoint_configs_changes
    AFTER INSERT OR UPDATE OR DELETE ON endpoint_configs
    FOR EACH ROW EXECUTE FUNCTION log_changes();

CREATE TRIGGER log_groups_changes
    AFTER INSERT OR UPDATE OR DELETE ON groups
    FOR EACH ROW EXECUTE FUNCTION log_changes();

CREATE TRIGGER log_hardware_configs_changes
    AFTER INSERT OR UPDATE OR DELETE ON hardware_configs
    FOR EACH ROW EXECUTE FUNCTION log_changes();

CREATE TRIGGER log_proxies_changes
    AFTER INSERT OR UPDATE OR DELETE ON proxies
    FOR EACH ROW EXECUTE FUNCTION log_changes();

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_transaction_status ON transactions(status);
CREATE INDEX IF NOT EXISTS idx_change_log_transaction ON change_log(transaction_id);
CREATE INDEX IF NOT EXISTS idx_change_log_created_at ON change_log(created_at);
CREATE INDEX IF NOT EXISTS idx_nodes_valid_range ON nodes(valid_from, valid_to);
CREATE INDEX IF NOT EXISTS idx_hwconfig_node ON hardware_configs(node_id);
CREATE INDEX IF NOT EXISTS idx_hwconfig_transaction ON hardware_configs(transaction_id);
CREATE INDEX IF NOT EXISTS idx_proxy_node ON proxies(node_id);
CREATE INDEX IF NOT EXISTS idx_proxy_group ON proxies(group_id);
CREATE INDEX IF NOT EXISTS idx_group_endpoint ON groups(endpoint_config_id);
`
