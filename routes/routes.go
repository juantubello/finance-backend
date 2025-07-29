package routes

import (
	"finance-backend/controllers/balance"
	"finance-backend/controllers/cards"
	"finance-backend/controllers/expenses"
	"finance-backend/controllers/incomes"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {

	expenseController := expenses.NewExpenseController()
	r.GET("/expenses", expenseController.GetExpenses)
	r.GET("/expenses/recent", expenseController.GetExpenses)
	r.GET("/expenses/summary", expenseController.GetExpensesSummary)
	r.GET("/expenses/sync/month", expenseController.SyncCurrentMonthExpenses)
	r.GET("/expenses/sync/historical", expenseController.SyncExpensesHistorical)

	incomeController := incomes.NewIncomeController()
	r.GET("/incomes", incomeController.GetIncomes)
	r.GET("/incomes/sync/month", incomeController.SyncCurrentMonthIncomes)
	r.GET("/incomes/sync/historical", incomeController.SyncIncomesHistorical)

	balanceController := balance.NewBalanceController()
	r.GET("/balance", balanceController.GetBalance)

	cardController := cards.NewCardsController()
	r.GET("/cards/sync/resumes", cardController.SyncResumes)
	r.GET("/cards/expenses", cardController.GetCardsExpenses)
	r.GET("/cards/subscriptions", cardController.GetSubscriptionSummary)
}
