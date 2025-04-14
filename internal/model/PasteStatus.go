package model

type PasteStatus string

const (
	StatusPending           PasteStatus = "pending"
	StatusRunning           PasteStatus = "running"
	StatusCompileError      PasteStatus = "compile error"
	StatusRuntimeError      PasteStatus = "runtime error"
	StatusTimeLimitExceed   PasteStatus = "time limit exceeded"
	StatusMemoryLimitExceed PasteStatus = "memory limit exceeded"
	StatusUnknownError      PasteStatus = "unknown error"
	StatusCompleted         PasteStatus = "completed"
)
