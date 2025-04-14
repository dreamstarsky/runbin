-- +goose Up
CREATE TABLE IF NOT EXISTS pastes (
    id VARCHAR(36) PRIMARY KEY,
    code TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    language VARCHAR(20),
    stdin TEXT,
    stdout TEXT,
    stderr TEXT,
    status VARCHAR(20) NOT NULL,
    execution_time_ms INTEGER,
    memory_usage_kb INTEGER,
    updated_at TIMESTAMP WITH TIME ZONE,
    backend VARCHAR(50)
);

-- +goose Down 
DROP TABLE pastes;