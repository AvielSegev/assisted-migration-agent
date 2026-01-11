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

-- Sequence for VM inspection ordering
CREATE SEQUENCE IF NOT EXISTS vm_inspection_status_seq START 1;

-- VM Inspection status table
CREATE TABLE IF NOT EXISTS vm_inspection_status (
    "VM ID" VARCHAR PRIMARY KEY,
    status VARCHAR NOT NULL,
    error VARCHAR,
    sequence INTEGER DEFAULT nextval('vm_inspection_status_seq'),
    FOREIGN KEY ("VM ID") REFERENCES vinfo("VM ID")
);
