package routes

import (
	"finance-backend/controllers"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	r.GET("/expenses", controllers.GetExpenses)
}
