package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

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

type Paste struct {
	ID              string
	Code            string
	Language        string
	Stdin           string
	Stdout          string
	Stderr          string
	Status          PasteStatus
	ExecutionTimeMs int
	MemoryUsageKb   int
	CreatedAt       time.Time
	UpdatedAt       time.Time
	BackEnd         string
}

type SubmitRequest struct {
	Code     string `json:"code" binding:"required"`
	Language string `json:"language" binding:"required"`
	Run      bool   `json:"run"`
	Stdin    string `json:"stdin"`
	BackEnd  string `json:"backend"`
}

type ExecutionTask struct {
	PasteID  string `json:"paste_id"`
	Code     string `json:"code"`
	Language string `json:"language"`
	Stdin    string `json:"stdin"`
}

// begin

var (
	// 使用 map 模拟数据库存储 Paste
	pastes = make(map[string]*Paste)
	// 使用读写锁保证并发访问 map 的安全
	pastesMutex = sync.RWMutex{}
)

// savePaste 将 Paste 保存到内存存储
func savePaste(p *Paste) {
	pastesMutex.Lock()         // 获取写锁
	defer pastesMutex.Unlock() // 函数结束时释放写锁
	p.UpdatedAt = time.Now()
	pastes[p.ID] = p
}

// getPasteByID 从内存存储中获取 Paste
func getPasteByID(id string) (*Paste, bool) {
	pastesMutex.RLock()         // 获取读锁
	defer pastesMutex.RUnlock() // 函数结束时释放读锁
	p, found := pastes[id]
	// 返回副本以避免外部修改影响存储 (对于指针类型，这仍然是浅拷贝，但对 Paste 结构体本身是安全的)
	if found {
		// 创建一个副本返回，避免并发问题
		// copy := *p
		// return &copy, true
		// 或者直接返回指针，但调用者不应修改它 (在这个简单例子中暂时直接返回)
		return p, true
	}
	return nil, false
}

// end

func dispatchExecutionTask(task ExecutionTask) error {
	// 在实际应用中，这里会连接到 RabbitMQ/Redis/Kafka 等，并将 task 序列化后发布
	fmt.Printf(" MOCK QUEUE: Dispatching task for Paste ID %s\n", task.PasteID)
	fmt.Printf("   Language: %s\n", task.Language)
	fmt.Printf("   Stdin: %s\n", task.Stdin[:min(50, len(task.Code))]+"...")
	fmt.Printf("   Code: %s\n", task.Code[:min(50, len(task.Code))]+"...") // 打印部分代码
	// 模拟成功
	return nil
	// 实际例子:
	// taskJSON, err := json.Marshal(task)
	// if err != nil { return err }
	// return messageQueueClient.Publish("execution_tasks", taskJSON)
}

func handleSubmitPaste(c *gin.Context) {
	var req SubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body: " + err.Error(),
		})
	}

	paste := &Paste{
		ID:        uuid.NewString(),
		Code:      req.Code,
		Language:  req.Language,
		Stdin:     req.Stdin,
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if !req.Run {
		paste.Status = StatusCompleted
	}

	savePaste(paste)
	log.Printf("Created Paste with ID: %s", paste.ID)

	c.JSON(http.StatusAccepted, gin.H{
		"message":  "Created",
		"paste_id": paste.ID,
		"url":      fmt.Sprintf("/api/pastes/%s", paste.ID),
	})

	if !req.Run {
		return
	}

	task := ExecutionTask{
		PasteID:  paste.ID,
		Code:     paste.Code,
		Language: paste.Language,
		Stdin:    paste.Stdin,
	}

	if err := dispatchExecutionTask(task); err != nil {
		// 如果分发失败，这通常是一个内部错误
		log.Printf("Error dispatching task for Paste ID %s: %v", paste.ID, err)
		paste.Status = StatusUnknownError
		savePaste(paste) // 更新状态
	}
}

func handleGetPaste(c *gin.Context) {
	pasteID := c.Param("id")

	paste, found := getPasteByID(pasteID)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Paste not found",
		})
		return
	}
	c.JSON(http.StatusOK, paste)
}

func handleGetLanguages(c *gin.Context) {
	supportedLanguages := []string{
		"c++20",
	}
	c.JSON(http.StatusOK, gin.H{
		"languages": supportedLanguages,
	})
}

func main() {
	router := gin.Default()
	api := router.Group("/api")
	{
		api.POST("/pastes", handleSubmitPaste)
		api.GET("/pastes/:id", handleGetPaste)
		api.GET("/languages", handleGetLanguages)
	}

	port := "8080"
	log.Printf("Statr API server on post %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
