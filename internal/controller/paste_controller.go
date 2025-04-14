package controller

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"runbin/internal/model"
	"runbin/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PasteHandler struct {
	repo     repository.PasteRepository
}

func NewPasteHandler(repo repository.PasteRepository) *PasteHandler {
	return &PasteHandler{
		repo:    repo,
	}
}

func (h *PasteHandler) SubmitPaste(c *gin.Context) {
	var req model.SubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	paste := &model.Paste{
		ID:        uuid.NewString(),
		Code:      req.Code,
		Language:  req.Language,
		Stdin:     req.Stdin,
		Status:    model.StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.repo.Save(paste); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Internal Server Error",
		})
		log.Printf("Paste save error: %v", err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":  "Created",
		"paste_id": paste.ID,
		"url":      fmt.Sprintf("/api/pastes/%s", paste.ID),
	})

	if req.Run {
		go h.repo.DispatchExecutionTask(paste.ID)
	}
}

func (h *PasteHandler) GetPaste(c *gin.Context) {
	pasteID := c.Param("id")
	paste, exists := h.repo.GetByID(pasteID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Paste not found"})
		return
	}
	c.JSON(http.StatusOK, paste)
}

func (h *PasteHandler) GetLanguages(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"languages": []string{"c++20"},
	})
}
