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

-- Function to ensure only one pending transaction exists
CREATE OR REPLACE FUNCTION ensure_single_pending_transaction()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'pending' THEN
        IF EXISTS (
            SELECT 1 FROM transactions 
            WHERE status = 'pending' 
            AND id != NEW.id
        ) THEN
            RAISE EXCEPTION 'Only one pending transaction is allowed at a time';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to enforce single pending transaction
CREATE TRIGGER enforce_single_pending_transaction
    BEFORE INSERT OR UPDATE ON transactions
    FOR EACH ROW
    EXECUTE FUNCTION ensure_single_pending_transaction();

CREATE OR REPLACE FUNCTION create_new_pending_transaction()
RETURNS UUID AS $$
DECLARE
    new_transaction_id UUID;
BEGIN
    INSERT INTO transactions (
        status,
        created_by,
        description
    ) VALUES (
        'pending',
        COALESCE(current_setting('app.current_user', TRUE), 'system'),
        'Auto-created pending transaction'
    ) RETURNING id INTO new_transaction_id;
    
    RETURN new_transaction_id;
END;
$$ LANGUAGE plpgsql;

-- Modified change logging function to always use current pending transaction
CREATE OR REPLACE FUNCTION log_changes()
RETURNS TRIGGER AS $$
DECLARE
    current_transaction_id UUID;
BEGIN
    -- Get the current pending transaction, or create a new one if none exists
    SELECT id INTO current_transaction_id
    FROM transactions
    WHERE status = 'pending'
    LIMIT 1;
    
    IF current_transaction_id IS NULL THEN
        current_transaction_id := create_new_pending_transaction();
    END IF;

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
        current_transaction_id,
        TG_TABLE_NAME,
        CASE 
            WHEN TG_OP = 'DELETE' THEN OLD.id::TEXT 
            ELSE NEW.id::TEXT 
        END,
        TG_OP::operation_type,
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

-- Modified rollback function to handle reversing operations
CREATE OR REPLACE FUNCTION handle_transaction_rollback()
RETURNS TRIGGER AS $$
DECLARE
    change_record RECORD;
BEGIN
    -- Only proceed if the status is changing to 'rollback'
    IF NEW.status = 'rollback' AND OLD.status != 'rollback' THEN
        -- Process changes in reverse order
        FOR change_record IN (
            SELECT * FROM change_log 
            WHERE transaction_id = NEW.id 
            ORDER BY created_at DESC
        ) LOOP
            CASE change_record.operation
                WHEN 'INSERT' THEN
                    -- Delete the inserted record
                    EXECUTE format('DELETE FROM %I WHERE id = $1::' || 
                        CASE 
                            WHEN change_record.table_name = 'nodes' THEN 'INTEGER'
                            ELSE 'TEXT'
                        END,
                        change_record.table_name
                    ) USING change_record.record_id;
                
                WHEN 'UPDATE' THEN
                    -- Restore the old data
                    EXECUTE format(
                        'UPDATE %I SET %s WHERE id = $1::' || 
                        CASE 
                            WHEN change_record.table_name = 'nodes' THEN 'INTEGER'
                            ELSE 'TEXT'
                        END,
                        change_record.table_name,
                        (SELECT string_agg(format('%I = ($2->%L)::%s', 
                            key, 
                            key,
                            CASE jsonb_typeof(value)
                                WHEN 'boolean' THEN 'boolean'
                                WHEN 'number' THEN 'numeric'
                                ELSE 'text'
                            END
                        ), ', ')
                        FROM jsonb_each(change_record.old_data)
                        WHERE key != 'id')
                    ) USING change_record.record_id, change_record.old_data;
                
                WHEN 'DELETE' THEN
                    -- Reinsert the deleted record
                    EXECUTE format(
                        'INSERT INTO %I (%s) VALUES (%s)',
                        change_record.table_name,
                        (SELECT string_agg(quote_ident(key), ', ')
                        FROM jsonb_each(change_record.old_data)),
                        (SELECT string_agg('($1->' || quote_literal(key) || ')::'
                            || CASE jsonb_typeof(value)
                                WHEN 'boolean' THEN 'boolean'
                                WHEN 'number' THEN 'numeric'
                                ELSE 'text'
                               END,
                            ', ')
                        FROM jsonb_each(change_record.old_data))
                    ) USING change_record.old_data;
            END CASE;
        END LOOP;

        -- Mark transaction as failed
        NEW.status := 'failed';
        NEW.completed_at := NOW();
        
        -- Create a new pending transaction
        PERFORM create_new_pending_transaction();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- Function to complete the current pending transaction and create a new one
CREATE OR REPLACE FUNCTION complete_transaction()
RETURNS UUID AS $$
DECLARE
    current_transaction_id UUID;
    new_transaction_id UUID;
BEGIN
    -- Get the current pending transaction
    SELECT id INTO current_transaction_id
    FROM transactions
    WHERE status = 'pending'
    LIMIT 1;
    
    IF current_transaction_id IS NULL THEN
        RAISE EXCEPTION 'No pending transaction found';
    END IF;

    -- Mark the current transaction as active and completed
    UPDATE transactions 
    SET status = 'active',
        completed_at = NOW()
    WHERE id = current_transaction_id;
    
    -- Create a new pending transaction
    new_transaction_id := create_new_pending_transaction();
    
    RETURN new_transaction_id;
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


-- Create trigger for transaction rollback
CREATE TRIGGER trigger_transaction_rollback
    AFTER UPDATE ON transactions
    FOR EACH ROW
    EXECUTE FUNCTION handle_transaction_rollback();


-- Create a function to rollback a transaction
CREATE OR REPLACE FUNCTION rollback_transaction()
RETURNS VOID AS $$
DECLARE
    current_transaction_id UUID;
BEGIN
    -- Get the current pending transaction
    SELECT id INTO current_transaction_id
    FROM transactions
    WHERE status = 'pending'
    LIMIT 1;
    
    IF current_transaction_id IS NULL THEN
        RAISE EXCEPTION 'No pending transaction found';
    END IF;

    -- Update the transaction status to trigger the rollback
    UPDATE transactions 
    SET status = 'rollback',
        completed_at = NOW()
    WHERE id = current_transaction_id;
END;
$$ LANGUAGE plpgsql;

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
