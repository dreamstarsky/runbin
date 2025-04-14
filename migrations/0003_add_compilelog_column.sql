-- +goose Up
ALTER TABLE pastes ADD COLUMN IF NOT EXISTS compile_log TEXT NOT NULL DEFAULT '';

-- +goose Down 
ALTER TABLE pastes DROP COLUMN compile_log;
