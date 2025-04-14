-- +goose Up
CREATE TABLE IF NOT EXISTS queue (
    id VARCHAR(36) PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    locked_at TIMESTAMP,
    attempts INT NOT NULL DEFAULT 0
);

-- +goose Down
DROP TABLE queue;

