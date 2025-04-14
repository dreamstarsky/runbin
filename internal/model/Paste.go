package model

import (
	"time"
)

type Paste struct {
	ID              string
	Code            string    `json:"code"`
	Language        string    `json:"language"`
	Stdin           string    `json:"stdin"`
	Stdout          string    `json:"stdout"`
	Stderr          string    `json:"stderr"`
	Status          PasteStatus `json:"status"`
	CompileLog      string    `json:"compile_log"`
	ExecutionTimeMs int       `json:"execution_time_ms"`
	MemoryUsageKb   int       `json:"memory_usage_kb"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	BackEnd         string    `json:"backend"`
}
