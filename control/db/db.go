package db

const schemaSQL = `

CREATE TYPE proxy_type AS ENUM ('forward', 'reverse', 'tlstls');
CREATE TYPE version_state AS ENUM ('draft', 'pending_deployment', 'active', 'disabled');
CREATE TYPE transaction_status AS ENUM ('pending', 'active', 'failed', 'rollback');
CREATE TYPE operation_type AS ENUM ('INSERT', 'UPDATE', 'DELETE', 'ADD');
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


-- Version Management Table with constraint
CREATE TABLE IF NOT EXISTS version_sets (
                                            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                                            name TEXT NOT NULL,
                                            description TEXT,
                                            state version_state NOT NULL DEFAULT 'draft',
                                            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                            activated_at TIMESTAMPTZ,
                                            disabled_at TIMESTAMPTZ,
                                            created_by TEXT NOT NULL,
                                            metadata JSONB
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

-- Nodes Table
CREATE TABLE IF NOT EXISTS nodes (
    id SERIAL PRIMARY KEY,
    serial_number TEXT UNIQUE NOT NULL CHECK (char_length(serial_number) <= 50),
    network_index INTEGER NOT NULL,
    locality TEXT,
    last_seen TIMESTAMPTZ,
    version_set_id UUID REFERENCES version_sets(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL
);

-- Endpoint Configurations
CREATE TABLE IF NOT EXISTS endpoint_configs (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    mutual_auth BOOLEAN NOT NULL DEFAULT false,
    no_encryption BOOLEAN NOT NULL DEFAULT false,
    asl_key_exchange_method asl_key_exchange_method NOT NULL DEFAULT 'ASL_KEX_DEFAULT',
    cipher TEXT,
    version_set_id UUID REFERENCES version_sets(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL
);

-- Groups Table
CREATE TABLE IF NOT EXISTS groups (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    log_level INTEGER NOT NULL DEFAULT 0,
    endpoint_config_id INTEGER REFERENCES endpoint_configs(id) ON DELETE SET NULL,
    legacy_config_id INTEGER REFERENCES endpoint_configs(id) ON DELETE SET NULL,
    version_set_id UUID REFERENCES version_sets(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL
);

-- Hardware Configurations
CREATE TABLE IF NOT EXISTS hardware_configs (
    id SERIAL PRIMARY KEY,
    node_id INTEGER REFERENCES nodes(id) ON DELETE CASCADE,
    device TEXT NOT NULL,
    ip_cidr INET NOT NULL,
    version_set_id UUID REFERENCES version_sets(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL
);

-- Proxies Table
CREATE TABLE IF NOT EXISTS proxies (
    id SERIAL PRIMARY KEY,
    node_id INTEGER REFERENCES nodes(id) ON DELETE CASCADE,
    group_id INTEGER REFERENCES groups(id) ON DELETE CASCADE,
    state BOOLEAN NOT NULL DEFAULT true,
    proxy_type proxy_type NOT NULL,
    server_endpoint_addr TEXT NOT NULL,
    client_endpoint_addr TEXT NOT NULL,
    version_set_id UUID REFERENCES version_sets(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS version_transitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_version_id UUID REFERENCES version_sets(id),
    to_version_id UUID REFERENCES version_sets(id) NOT NULL,
    status transaction_status NOT NULL DEFAULT 'pending',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    created_by TEXT NOT NULL,
    metadata JSONB
);

-- Add indexes for version management
CREATE INDEX IF NOT EXISTS idx_version_sets_state ON version_sets(state);
CREATE INDEX IF NOT EXISTS idx_version_transitions_status ON version_transitions(status);
CREATE INDEX IF NOT EXISTS idx_nodes_version ON nodes(version_set_id);
CREATE INDEX IF NOT EXISTS idx_proxies_version ON proxies(version_set_id);
CREATE INDEX IF NOT EXISTS idx_hwconfig_version ON hardware_configs(version_set_id);
CREATE INDEX IF NOT EXISTS idx_groups_version ON groups(version_set_id);
CREATE INDEX IF NOT EXISTS idx_endpoint_version ON endpoint_configs(version_set_id);

`

const functionsSQL = `
-- Create function to create a new pending transaction
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

-- Create function to ensure single pending transaction
CREATE OR REPLACE FUNCTION ensure_single_pending_transaction()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'pending' THEN
        IF EXISTS (
            SELECT 1 FROM transactions 
            WHERE status = 'pending' 
            AND id != NEW.id
        ) THEN
            RAISE EXCEPTION 'Only one pending transaction is allowed at a time.';
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

-- Create function to log changes
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

-- Create triggers for change logging
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

-- Create function to handle transaction rollback
CREATE OR REPLACE FUNCTION handle_transaction_rollback()
RETURNS TRIGGER AS $$
DECLARE
    change_record RECORD;
    update_query TEXT;
    insert_query TEXT;
BEGIN
    IF NEW.status = 'rollback' AND OLD.status = 'pending' THEN
        -- Process each change in reverse order
        FOR change_record IN (
            SELECT * FROM change_log 
            WHERE transaction_id = NEW.id 
            ORDER BY created_at DESC
        ) LOOP
            CASE change_record.operation
                WHEN 'INSERT' THEN
                    -- For INSERT, we DELETE
                    EXECUTE format('DELETE FROM %I WHERE id = $1::%s',
                        change_record.table_name,
                        CASE 
                            WHEN change_record.table_name = 'nodes' THEN 'INTEGER'
                            ELSE 'TEXT'
                        END
                    ) USING change_record.record_id;

                WHEN 'UPDATE' THEN
                    -- For UPDATE, we restore old values
                    SELECT string_agg(
                        format('%I = ($2->>%L)::%s', 
                            key, key,
                            CASE jsonb_typeof(value)
                                WHEN 'boolean' THEN 'boolean'
                                WHEN 'number' THEN 'numeric'
                                ELSE 'TEXT'
                            END
                        ), ', '
                    ) INTO update_query
                    FROM jsonb_each(change_record.old_data)
                    WHERE key != 'id';

                    EXECUTE format('UPDATE %I SET %s WHERE id = $1::%s',
                        change_record.table_name, update_query,
                        CASE 
                            WHEN change_record.table_name = 'nodes' THEN 'INTEGER'
                            ELSE 'TEXT'
                        END
                    ) USING change_record.record_id, change_record.old_data;

                WHEN 'DELETE' THEN
                    -- For DELETE, we re-INSERT
                    SELECT format(
                        'INSERT INTO %I (%s) VALUES (%s)',
                        change_record.table_name,
                        string_agg(quote_ident(key), ', '),
                        string_agg('($1->>' || quote_literal(key) || ')::' ||
                            CASE jsonb_typeof(value)
                                WHEN 'boolean' THEN 'boolean'
                                WHEN 'number' THEN 'numeric'
                                ELSE 'TEXT'
                            END, ', ')
                    )
                    INTO insert_query
                    FROM jsonb_each(change_record.old_data);

                    EXECUTE insert_query USING change_record.old_data;
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

-- Create trigger for transaction rollback
CREATE TRIGGER trigger_transaction_rollback
    AFTER UPDATE ON transactions
    FOR EACH ROW
    EXECUTE FUNCTION handle_transaction_rollback();

-- Create function to rollback a transaction
CREATE OR REPLACE FUNCTION rollback_transaction()
RETURNS void AS $$
DECLARE
    current_transaction_id UUID;
BEGIN
    -- Fetch the current transaction ID
    SELECT id INTO current_transaction_id 
    FROM transactions 
    WHERE status = 'pending' 
    ORDER BY created_at DESC 
    LIMIT 1;

    -- Check if a transaction exists
    IF current_transaction_id IS NOT NULL THEN
        -- Update the transaction status to 'rollback'
        UPDATE transactions 
        SET status = 'rollback',
            completed_at = NOW()
        WHERE id = current_transaction_id;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Create function to complete a transaction
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
`
