package balance

import (
	"finance-backend/models"
	"net/http"

	"github.com/gin-gonic/gin"

	transactions "finance-backend/controllers/base"
)

type BalanceController struct {
	*transactions.BaseController // Embed base to share base methods
}

func NewBalanceController() *BalanceController {
	return &BalanceController{
		BaseController: &transactions.BaseController{},
	}
}

type IncomeSyncResponse struct {
	DeletedRows        int              `json:"rows_deleted"`
	DeletedRowsDetail  []models.Incomes `json:"deleted_rows_detail"`
	InsertedRows       int              `json:"inserted_rows"`
	InsertedRowsDetail []models.Incomes `json:"inserted_rows_detail"`
}

type SyncIncomeData struct {
	HistoricalSync bool
	DatePattern    string
	DatePattern2   string
	SheetId        string
	SheetName      string
	SheetRange     string
}

// GetExpenses obtiene los gastos filtrados por fecha
func (ec *BalanceController) GetBalance(c *gin.Context) {

	type Balance struct {
		Balance           float64 `json:"balance"`
		FormattedBalance  string  `json:"formatted_balance"`
		TotalExpenses     float64 `json:"total_expenses"`
		FormattedExpenses string  `json:"formatted_expenses"`
		TotalIncomes      float64 `json:"total_incomes"`
		FormattedIncomes  string  `json:"formatted_incomes"`
	}

	// Response structs for db query
	var totalExpenses []struct {
		Total float64
	}

	var totalIncome []struct {
		Total float64
	}

	var incomes []models.Incomes
	var expenses []models.Expenses

	db, err := ec.GetDatabaseInstance("TRANSACTION_DB")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := db.Model(&expenses).
		Select("sum(amount) as total").
		Find(&totalExpenses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := db.Model(&incomes).
		Select("sum(amount) as total").
		Find(&totalIncome).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	expensesAmount := totalExpenses[0].Total
	incomesAmount := totalIncome[0].Total
	currentBalance := incomesAmount - expensesAmount

	balance := Balance{
		Balance:           currentBalance,
		FormattedBalance:  ec.FormatAmount(currentBalance),
		TotalExpenses:     expensesAmount,
		FormattedExpenses: ec.FormatAmount(expensesAmount),
		TotalIncomes:      incomesAmount,
		FormattedIncomes:  ec.FormatAmount(incomesAmount),
	}

	c.JSON(http.StatusOK, balance)
}
