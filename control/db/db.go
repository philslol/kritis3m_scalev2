package db

const schemaSQL = `
-- Create ENUM types
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

-- Transactions Table
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status transaction_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    created_by TEXT NOT NULL,
    description TEXT,
    metadata JSONB
);

-- Change Log
CREATE TABLE IF NOT EXISTS change_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID REFERENCES transactions(id) ON DELETE CASCADE,
    table_name TEXT NOT NULL,
    record_id TEXT NOT NULL,
    operation operation_type NOT NULL,
    old_data JSONB,
    new_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL
);

-- Nodes Table
CREATE TABLE IF NOT EXISTS nodes (
    id SERIAL PRIMARY KEY,
    serial_number TEXT UNIQUE NOT NULL CHECK (char_length(serial_number) <= 50),
    network_index INTEGER NOT NULL,
    locality TEXT,
    last_seen TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL
);

-- Endpoint Configurations
CREATE TABLE IF NOT EXISTS endpoint_configs (
    id SERIAL PRIMARY KEY,
    transaction_id UUID REFERENCES transactions(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    mutual_auth BOOLEAN NOT NULL DEFAULT false,
    no_encryption BOOLEAN NOT NULL DEFAULT false,
    asl_key_exchange_method asl_key_exchange_method NOT NULL DEFAULT 'ASL_KEX_DEFAULT',
    cipher TEXT,
    status transaction_status NOT NULL DEFAULT 'pending',
    version INTEGER NOT NULL DEFAULT 1,
    previous_version_id INTEGER REFERENCES endpoint_configs(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL
);

-- Groups Table
CREATE TABLE IF NOT EXISTS groups (
    id SERIAL PRIMARY KEY,
    transaction_id UUID REFERENCES transactions(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    log_level INTEGER NOT NULL DEFAULT 0,
    endpoint_config_id INTEGER REFERENCES endpoint_configs(id) ON DELETE SET NULL,
    legacy_config_id INTEGER REFERENCES endpoint_configs(id) ON DELETE SET NULL,
    status transaction_status NOT NULL DEFAULT 'pending',
    version INTEGER NOT NULL DEFAULT 1,
    previous_version_id INTEGER REFERENCES groups(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL
);

-- Hardware Configurations
CREATE TABLE IF NOT EXISTS hardware_configs (
    id SERIAL PRIMARY KEY,
    node_id INTEGER REFERENCES nodes(id) ON DELETE CASCADE,
    transaction_id UUID REFERENCES transactions(id) ON DELETE CASCADE,
    device TEXT NOT NULL,
    ip_cidr INET NOT NULL,
    status transaction_status NOT NULL DEFAULT 'pending',
    version INTEGER NOT NULL DEFAULT 1,
    previous_version_id INTEGER REFERENCES hardware_configs(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL
);

-- Proxies Table
CREATE TABLE IF NOT EXISTS proxies (
    id SERIAL PRIMARY KEY,
    transaction_id UUID REFERENCES transactions(id) ON DELETE CASCADE,
    node_id INTEGER REFERENCES nodes(id) ON DELETE CASCADE,
    group_id INTEGER REFERENCES groups(id) ON DELETE CASCADE,
    state BOOLEAN NOT NULL DEFAULT true,
    proxy_type proxy_type NOT NULL,
    server_endpoint_addr TEXT NOT NULL,
    client_endpoint_addr TEXT NOT NULL,
    status transaction_status NOT NULL DEFAULT 'pending',
    version INTEGER NOT NULL DEFAULT 1,
    previous_version_id INTEGER REFERENCES proxies(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL
);

-- Indexes for Performance
CREATE INDEX IF NOT EXISTS idx_transaction_status ON transactions(status);
CREATE INDEX IF NOT EXISTS idx_change_log_transaction ON change_log(transaction_id);
CREATE INDEX IF NOT EXISTS idx_change_log_created_at ON change_log(created_at);
CREATE INDEX IF NOT EXISTS idx_hwconfig_node ON hardware_configs(node_id);
CREATE INDEX IF NOT EXISTS idx_hwconfig_transaction ON hardware_configs(transaction_id);
CREATE INDEX IF NOT EXISTS idx_proxy_node ON proxies(node_id);
CREATE INDEX IF NOT EXISTS idx_proxy_group ON proxies(group_id);
CREATE INDEX IF NOT EXISTS idx_group_endpoint ON groups(endpoint_config_id);

`

const functionsSQL = `
DROP TRIGGER IF EXISTS enforce_single_pending_transaction ON transactions;
DROP TRIGGER IF EXISTS log_nodes_changes ON nodes;
DROP TRIGGER IF EXISTS log_endpoint_configs_changes ON endpoint_configs;
DROP TRIGGER IF EXISTS log_groups_changes ON groups;
DROP TRIGGER IF EXISTS log_hardware_configs_changes ON hardware_configs;
DROP TRIGGER IF EXISTS log_proxies_changes ON proxies;
DROP TRIGGER IF EXISTS trigger_transaction_rollback ON transactions;

DROP FUNCTION IF EXISTS create_new_pending_transaction();
DROP FUNCTION IF EXISTS ensure_single_pending_transaction();
DROP FUNCTION IF EXISTS log_changes();
DROP FUNCTION IF EXISTS handle_transaction_rollback();
DROP FUNCTION IF EXISTS rollback_transaction();
DROP FUNCTION IF EXISTS complete_transaction();

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

CREATE OR REPLACE FUNCTION log_changes()
RETURNS TRIGGER AS $$
DECLARE
    current_transaction_id UUID;
BEGIN
    SELECT id INTO current_transaction_id
    FROM transactions
    WHERE status = 'pending'
    LIMIT 1;
    
    IF current_transaction_id IS NULL THEN

        RAISE NOTICE 'new pending transaction';
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

CREATE OR REPLACE FUNCTION complete_transaction()
RETURNS UUID AS $$
DECLARE
    current_transaction_id UUID;
    new_transaction_id UUID;
BEGIN
    SELECT id INTO current_transaction_id
    FROM transactions
    WHERE status = 'pending'
    LIMIT 1;
    
    IF current_transaction_id IS NULL THEN
        RAISE EXCEPTION 'No pending transaction found';
    END IF;

    UPDATE transactions 
    SET status = 'active',
        completed_at = NOW()
    WHERE id = current_transaction_id;
    
    new_transaction_id := create_new_pending_transaction();
    RETURN new_transaction_id;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION handle_transaction_rollback() 
RETURNS TRIGGER AS $$
DECLARE
    change_record RECORD;
    column_definitions TEXT;
    update_statement TEXT;
BEGIN
    -- Ensure rollback is triggered only when transitioning to 'rollback' status
    IF NEW.status = 'rollback' AND OLD.status != 'rollback' THEN
        
        -- Process change_log records in reverse order
        FOR change_record IN (
            SELECT * FROM change_log 
            WHERE transaction_id = NEW.id 
            ORDER BY created_at DESC
        ) LOOP
    -- Disable triggers on the current table
    EXECUTE format('ALTER TABLE %I DISABLE TRIGGER ALL', change_record.table_name);

            
            CASE change_record.operation
                WHEN 'INSERT' THEN
                    -- Delete the inserted record
                    EXECUTE format(
                        'DELETE FROM %I WHERE id = $1', 
                        change_record.table_name
                    ) USING change_record.record_id::INTEGER;
                
                WHEN 'UPDATE' THEN
                    -- Generate column assignments dynamically with explicit type casting
                    column_definitions := (
                        SELECT string_agg(
                            format('%I = ($2->>%L)::%s', 
                                key, 
                                key,
                                CASE 
                                    WHEN key = 'last_seen' OR key = 'created_at' OR key = 'updated_at' THEN 'timestamptz'
                                    WHEN key = 'network_index' OR key = 'id' THEN 'integer'
                                    WHEN jsonb_typeof(value) = 'boolean' THEN 'boolean'
                                    WHEN jsonb_typeof(value) = 'number' THEN 'numeric'
                                    ELSE 'text'
                                END
                            ), ', '
                        )
                        FROM jsonb_each(change_record.old_data)
                        WHERE key != 'id' -- Exclude primary key
                    );

                    -- Construct dynamic UPDATE query
                    update_statement := format(
                        'UPDATE %I SET %s WHERE id = $1::INTEGER',
                        change_record.table_name, 
                        column_definitions
                    );

                    -- Debugging: Print query for inspection
                    RAISE NOTICE 'Executing rollback query: %', update_statement;

                    -- Execute the update
                    EXECUTE update_statement USING change_record.record_id::INTEGER, change_record.old_data;
                
                WHEN 'DELETE' THEN
                    -- Reinsert the deleted record with explicit casting
                    EXECUTE format(
                        'INSERT INTO %I (%s) VALUES (%s)',
                        change_record.table_name,
                        (SELECT string_agg(quote_ident(key), ', ') FROM jsonb_each(change_record.old_data)),
                        (SELECT string_agg(
                            '($1->>' || quote_literal(key) || ')::' ||
                            CASE 
                                WHEN key = 'last_seen' OR key = 'created_at' OR key = 'updated_at' THEN 'timestamptz'
                                WHEN key = 'network_index' OR key = 'id' THEN 'integer'
                                WHEN jsonb_typeof(value) = 'boolean' THEN 'boolean'
                                WHEN jsonb_typeof(value) = 'number' THEN 'numeric'
                                ELSE 'text'
                            END,
                        ', ') FROM jsonb_each(change_record.old_data))
                    ) USING change_record.old_data;
            END CASE;

    -- Re-enable triggers on the current table after processing
    EXECUTE format('ALTER TABLE %I ENABLE TRIGGER ALL', change_record.table_name);
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



CREATE TRIGGER trigger_transaction_rollback
    AFTER UPDATE ON transactions
    FOR EACH ROW
    EXECUTE FUNCTION handle_transaction_rollback();

CREATE OR REPLACE FUNCTION rollback_transaction()
RETURNS void AS $$
DECLARE
    current_transaction_id UUID;
BEGIN
    SELECT id INTO current_transaction_id 
    FROM transactions 
    WHERE status = 'pending' 
    ORDER BY created_at DESC 
    LIMIT 1;

    IF current_transaction_id IS NOT NULL THEN
	    EXECUTE format(
        'UPDATE transactions 
        SET status = %L, completed_at = NOW()::timestamptz
        WHERE id = %L',
        'rollback',
        current_transaction_id
    );
    END IF;
END;
$$ LANGUAGE plpgsql;
`
