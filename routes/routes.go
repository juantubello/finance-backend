package routes

import (
	expenses "finance-backend/controllers/expenses" // 👈 Importá el paquete correcto

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	// Usá el constructor del paquete expenses
	expenseController := expenses.NewExpenseController()
	// Crea una instancia del controlador
	//expenseCtrl := controllers.NewExpenseController()
	r.GET("/expenses", expenseController.GetExpenses)
	r.GET("/expenses/summary", expenseController.GetExpensesSummary)
	r.GET("/expenses/sync/month", expenseController.SyncCurrentMonthExpenses)
	r.GET("/expenses/sync/historical", expenseController.SyncExpensesHistorical)
}
