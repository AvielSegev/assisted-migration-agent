CREATE TABLE IF NOT EXISTS configuration (
    id INTEGER PRIMARY KEY DEFAULT 1,
    agent_mode VARCHAR DEFAULT 'disconnected',
    CHECK (id = 1)
);

-- Inventory storage table
CREATE TABLE IF NOT EXISTS inventory (
    id INTEGER PRIMARY KEY DEFAULT 1,
    data BLOB NOT NULL,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now(),
    CHECK (id = 1)
);

-- VM TABLE
CREATE TABLE IF NOT EXISTS vms (
    id VARCHAR(200) PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    state VARCHAR(50) NOT NULL,
    datacenter VARCHAR(200) NOT NULL,
    cluster VARCHAR(200) NOT NULL,
    disk_size VARCHAR(50) NOT NULL,
    memory VARCHAR(50) NOT NULL,
    issues VARCHAR[],
    inspection_state VARCHAR(100),
    inspection_error VARCHAR(100),
    inspection_result BLOB
);
