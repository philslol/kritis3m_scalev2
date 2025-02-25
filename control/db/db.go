package db

const schemaSQL = `
-- Keep existing ENUM types
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

-- Version Management Table (unchanged)
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

-- Transactions Table (unchanged)
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status transaction_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    created_by TEXT NOT NULL,
    description TEXT,
    metadata JSONB
);

-- Change log (unchanged)
CREATE TABLE IF NOT EXISTS change_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID REFERENCES transactions(id),
    table_name TEXT NOT NULL,
    record_id TEXT NOT NULL,
    operation operation_type NOT NULL,
    old_data JSONB,
    new_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL
);

-- Modified Nodes Table
CREATE TABLE IF NOT EXISTS nodes (
    id SERIAL UNIQUE,
    serial_number TEXT NOT NULL CHECK (char_length(serial_number) <= 50),
    network_index INTEGER NOT NULL,
    locality TEXT,
    last_seen TIMESTAMPTZ,
    version_set_id UUID REFERENCES version_sets(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL,
    PRIMARY KEY (serial_number, version_set_id)
);

-- Modified Endpoint Configurations
CREATE TABLE IF NOT EXISTS endpoint_configs (
    id SERIAL UNIQUE,
    name TEXT NOT NULL,
    mutual_auth BOOLEAN NOT NULL DEFAULT false,
    no_encryption BOOLEAN NOT NULL DEFAULT false,
    asl_key_exchange_method asl_key_exchange_method NOT NULL DEFAULT 'ASL_KEX_DEFAULT',
    cipher TEXT,
    version_set_id UUID REFERENCES version_sets(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL,
    PRIMARY KEY (name, version_set_id)
);

-- Modified Groups Table
CREATE TABLE IF NOT EXISTS groups (
    id SERIAL UNIQUE,
    name TEXT NOT NULL,
    log_level INTEGER NOT NULL DEFAULT 0,
    endpoint_config_name TEXT,
    legacy_config_name TEXT,
    version_set_id UUID REFERENCES version_sets(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL,
    PRIMARY KEY (name, version_set_id),
    FOREIGN KEY (endpoint_config_name, version_set_id)
        REFERENCES endpoint_configs(name, version_set_id),
    FOREIGN KEY (legacy_config_name, version_set_id)
        REFERENCES endpoint_configs(name, version_set_id)
);

-- Hardware Configurations
CREATE TABLE IF NOT EXISTS hardware_configs (
    id SERIAL PRIMARY KEY ,
    node_serial TEXT NOT NULL,
    device TEXT NOT NULL,
    ip_cidr INET NOT NULL,
    version_set_id UUID REFERENCES version_sets(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL,
    FOREIGN KEY (node_serial, version_set_id)
        REFERENCES nodes(serial_number, version_set_id)
);

-- Modified Proxies Table
CREATE TABLE IF NOT EXISTS proxies (
    id SERIAL UNIQUE,
    name TEXT NOT NULL,
    node_serial TEXT NOT NULL,
    group_name TEXT NOT NULL,
    state BOOLEAN NOT NULL DEFAULT true,
    proxy_type proxy_type NOT NULL,
    server_endpoint_addr TEXT NOT NULL,
    client_endpoint_addr TEXT NOT NULL,
    version_set_id UUID REFERENCES version_sets(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL,
    PRIMARY KEY (name, version_set_id),
    FOREIGN KEY (node_serial, version_set_id)
        REFERENCES nodes(serial_number, version_set_id),
    FOREIGN KEY (group_name, version_set_id)
        REFERENCES groups(name, version_set_id)
);

-- Version transitions (unchanged)
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

-- Indexes
CREATE INDEX IF NOT EXISTS idx_version_sets_state ON version_sets(state);
CREATE INDEX IF NOT EXISTS idx_version_transitions_status ON version_transitions(status);
CREATE INDEX IF NOT EXISTS idx_nodes_version ON nodes(version_set_id);
CREATE INDEX IF NOT EXISTS idx_proxies_version ON proxies(version_set_id);
CREATE INDEX IF NOT EXISTS idx_hwconfig_version ON hardware_configs(version_set_id);
CREATE INDEX IF NOT EXISTS idx_groups_version ON groups(version_set_id);
CREATE INDEX IF NOT EXISTS idx_endpoint_version ON endpoint_configs(version_set_id);
`
