package balance

import (
	"finance-backend/models"
	"fmt"
	"net/http"
	"strconv"
	"time"

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
		Balance                  float64 `json:"balance"`
		FormattedBalance         string  `json:"formatted_balance"`
		TotalExpenses            float64 `json:"total_expenses"`
		FormattedExpenses        string  `json:"formatted_expenses"`
		TotalIncomes             float64 `json:"total_incomes"`
		FormattedIncomes         string  `json:"formatted_incomes"`
		FormattedMonthlyIncome   string  `json:"formatted_monthly_income"`
		FormattedMonthlyExpenses string  `json:"formatted_monthly_expenses"`
		FormattedMonthlyCardsARS string  `json:"formatted_monthly_cards_ars"`
		FormattedMonthlyCardsUSD string  `json:"formatted_monthly_cards_usd"`
	}

	// Response structs for db query
	var totalExpenses []struct {
		Total float64
	}

	var sumMonthlyExpenses []struct {
		Total float64
	}

	var sumMonthlyIncome []struct {
		Total float64
	}

	var totalIncome []struct {
		Total float64
	}

	monthStr, yearStr := getCurrentMonthAndYear()

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid year"})
		return
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid month"})
		return
	}

	dateFilter := fmt.Sprintf("%04d-%02d", year, month)

	var incomes []models.Incomes
	var expenses []models.Expenses

	db, err := ec.GetDatabaseInstance("TRANSACTION_DB")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cardsDB, err := ec.GetDatabaseInstance("CARDS_DB")
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

	// ---------- Monthly expenses ----------

	query := db.Model(&expenses).Select("sum(amount) as total").Where("strftime('%Y-%m', date) = ?", dateFilter)
	err = query.Find(&sumMonthlyExpenses).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// ---------- Monthly income ----------

	new_month_format := monthStr

	if monthStr[0] == '0' {
		new_month_format = monthStr[1:]
	} else {
		new_month_format = monthStr
	}

	datePatternOld := fmt.Sprintf("%%/%s/%s%%", new_month_format, yearStr)

	query2 := db.Model(&incomes).Distinct().Where("date_time LIKE ?", datePatternOld)

	if err := query2.Select("sum(amount) as total").Find(&sumMonthlyIncome).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// ---------- Monthly card spent ----------
	var resumes []models.Resume
	// Response structs for db query
	var totalMonthltyCards []struct {
		TotalArs float64
		TotalUsd float64
	}

	queryCards := cardsDB.Model(&resumes).Select("sum(total_ars) as TotalArs, sum(total_usd) as TotalUsd").Where("strftime('%Y-%m', resume_date) = ?", dateFilter)
	err = queryCards.Find(&totalMonthltyCards).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	expensesAmount := totalExpenses[0].Total
	incomesAmount := totalIncome[0].Total
	currentBalance := incomesAmount - expensesAmount

	balance := Balance{
		Balance:                  currentBalance,
		FormattedBalance:         ec.FormatAmount(currentBalance),
		TotalExpenses:            expensesAmount,
		FormattedExpenses:        ec.FormatAmount(expensesAmount),
		TotalIncomes:             incomesAmount,
		FormattedIncomes:         ec.FormatAmount(incomesAmount),
		FormattedMonthlyIncome:   ec.FormatAmount(sumMonthlyIncome[0].Total),
		FormattedMonthlyExpenses: ec.FormatAmount(sumMonthlyExpenses[0].Total),
		FormattedMonthlyCardsARS: ec.FormatAmount(totalMonthltyCards[0].TotalArs),
		FormattedMonthlyCardsUSD: ec.FormatAmount(totalMonthltyCards[0].TotalUsd),
	}

	c.JSON(http.StatusOK, balance)
}

func getCurrentMonthAndYear() (string, string) {
	now := time.Now()
	month := fmt.Sprintf("%02d", int(now.Month())) // fuerza dos dígitos
	year := fmt.Sprintf("%d", now.Year())
	return month, year
}
