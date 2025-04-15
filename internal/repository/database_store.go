package repository

import (
	"context"
	"database/sql"
	"fmt"
	"runbin/internal/model"
	"time"

	_ "github.com/lib/pq"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(connStr string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) Save(p *model.Paste) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO pastes (
			id, code, created_at, status,
			language, stdin, stdout, stderr,
			execution_time_ms, memory_usage_kb, updated_at, backend,
			compile_log
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		p.ID, p.Code, p.CreatedAt, p.Status,
		p.Language, p.Stdin, p.Stdout, p.Stderr,
		p.ExecutionTimeMs, p.MemoryUsageKb, p.UpdatedAt, p.BackEnd, p.CompileLog)

	return err
}

func (s *PostgresStore) GetByID(id string) (*model.Paste, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var p model.Paste
	err := s.db.QueryRowContext(ctx,
		`SELECT 
			id, code, created_at, status,
			language, stdin, stdout, stderr,
			execution_time_ms, memory_usage_kb, updated_at, backend, 
			compile_log
		FROM pastes WHERE id = $1`, id).Scan(
		&p.ID,
		&p.Code,
		&p.CreatedAt,
		&p.Status,
		&p.Language,
		&p.Stdin,
		&p.Stdout,
		&p.Stderr,
		&p.ExecutionTimeMs,
		&p.MemoryUsageKb,
		&p.UpdatedAt,
		&p.BackEnd,
		&p.CompileLog)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false
		}
		return nil, false
	}
	return &p, true
}

func (s *PostgresStore) Close() error {
	return s.db.Close()
}

func (s *PostgresStore) DispatchExecutionTask(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO queue (id) VALUES ($1)`,
		id)
	return err
}

func (s *PostgresStore) GetTask(ctx context.Context) (*model.Paste, error) {

	// 原子性地删除并获取队列中最旧的任务ID
	var taskID string
	err := s.db.QueryRowContext(ctx,
		`DELETE FROM queue 
		WHERE ctid = (
			SELECT ctid FROM queue 
			ORDER BY created_at 
			FOR UPDATE SKIP LOCKED 
			LIMIT 1
		)
		RETURNING id`).Scan(&taskID)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // 没有任务时返回nil
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// 获取完整的任务数据
	p, ok := s.GetByID(taskID)

	if !ok {
		return nil, fmt.Errorf("failed to get task details: %w", err)
	}

	return p, nil
}

func (s *PostgresStore) Update(p *model.Paste) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Always update the UpdatedAt timestamp on an update operation
	p.UpdatedAt = time.Now()

	_, err := s.db.ExecContext(ctx,
		`UPDATE pastes SET
			status = $1,
			stdout = $2,
			stderr = $3,
			execution_time_ms = $4,
			memory_usage_kb = $5,
			updated_at = $6,  
			backend = $7,
			compile_log = $8
		WHERE id = $9; `,
		p.Status,
		p.Stdout,
		p.Stderr,
		p.ExecutionTimeMs,
		p.MemoryUsageKb,
		p.UpdatedAt,
		p.BackEnd,
		p.CompileLog,
		p.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to execute update for paste with id %s: %w", p.ID, err)
	}

	return nil
}
