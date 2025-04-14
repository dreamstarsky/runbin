package router

import (
	"runbin/internal/controller"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(engine *gin.Engine, handler *controller.PasteHandler) {
	api := engine.Group("/api")
	{
		api.POST("/pastes", handler.SubmitPaste)
		api.GET("/pastes/:id", handler.GetPaste)
		api.GET("/languages", handler.GetLanguages)
	}
}
