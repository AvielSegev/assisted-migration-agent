-- Initial schema for agent credentials storage
CREATE TABLE IF NOT EXISTS credentials (
    id INTEGER PRIMARY KEY DEFAULT 1,
    url VARCHAR NOT NULL,
    username VARCHAR NOT NULL,
    password VARCHAR NOT NULL,
    is_data_sharing_allowed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now(),
    CHECK (id = 1)
);
