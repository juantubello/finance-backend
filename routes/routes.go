package routes

import (
	"finance-backend/controllers/expenses"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {

	expenseController := expenses.NewExpenseController()

	r.GET("/expenses", expenseController.GetExpenses)
	r.GET("/expenses/summary", expenseController.GetExpensesSummary)
	r.GET("/expenses/sync/month", expenseController.SyncCurrentMonthExpenses)
	r.GET("/expenses/sync/historical", expenseController.SyncExpensesHistorical)
}
