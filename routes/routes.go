package routes

import (
	"finance-backend/controllers"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	// Crea una instancia del controlador
	expenseCtrl := controllers.NewExpenseController()
	r.GET("/expenses", expenseCtrl.GetExpenses)
	r.GET("/expenses/summary", expenseCtrl.GetExpensesSummary)
}
