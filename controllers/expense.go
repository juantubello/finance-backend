package controllers

import (
	"finance-backend/config"
	"finance-backend/models"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gorm.io/gorm"
)

// ExpenseService define la interfaz para operaciones con gastos
type ExpenseService interface {
	GetExpenses(c *gin.Context)
	GetExpensesSummary(c *gin.Context)
	formatAmount(amount float64) string
	getDB() (*gorm.DB, error)
}

// BaseExpense contiene la lógica compartida
type BaseExpense struct{}

// formatAmount formatea montos en pesos argentinos
func (e *BaseExpense) formatAmount(amount float64) string {
	printer := message.NewPrinter(language.Spanish)
	return printer.Sprintf("$%.2f", amount)
}

// getDB obtiene la conexión a la base de datos
func (e *BaseExpense) getDB() (*gorm.DB, error) {
	transactionsTable := config.GetEnv("TRANSACTION_TABLE")
	db, ok := config.DBs[transactionsTable]
	if !ok {
		return nil, fmt.Errorf("database not available")
	}
	return db, nil
}

// ExpenseController implementa ExpenseService
type ExpenseController struct {
	*BaseExpense // Embedding para heredar métodos
}

// NewExpenseController crea una nueva instancia del controlador
func NewExpenseController() *ExpenseController {
	return &ExpenseController{
		BaseExpense: &BaseExpense{},
	}
}

// FormattedExpenseResponse representa la respuesta formateada
type FormattedExpenseResponse struct {
	models.Expenses
	FormattedAmount string `json:"formatted_amount"`
}

// GetExpenses obtiene los gastos filtrados por fecha
func (ec *ExpenseController) GetExpenses(c *gin.Context) {
	year := c.Query("year")
	month := c.Query("month")
	datePattern := fmt.Sprintf("%s-%s%%", year, month)

	db, err := ec.getDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var expenses []models.Expenses
	if err := db.Where("date_time LIKE ?", datePattern).Find(&expenses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	formatted := make([]FormattedExpenseResponse, len(expenses))
	for i, exp := range expenses {
		formatted[i] = FormattedExpenseResponse{
			Expenses:        exp,
			FormattedAmount: ec.formatAmount(exp.Amount),
		}
	}

	c.JSON(http.StatusOK, formatted)
}

// SummaryResponse representa la respuesta del resumen
type TypeSummary struct {
	Type           string  `json:"type"`
	Total          float64 `json:"total"`
	FormattedTotal string  `json:"formatted_total"`
}

type ExpensesSummaryResponse struct {
	Total          float64       `json:"total"`
	FormattedTotal string        `json:"formatted_total"`
	Period         string        `json:"period"`
	TypesSummary   []TypeSummary `json:"types_summary"`
}

// GetExpensesSummary obtiene el resumen de gastos por categoría,
// hereda de ExpenseController para obtener los metodos base de obtener
// base de datos y formateo a pesos
func (ec *ExpenseController) GetExpensesSummary(c *gin.Context) {
	year := c.Query("year")
	month := c.Query("month")
	datePattern := fmt.Sprintf("%s-%s%%", year, month)
	period := fmt.Sprintf("%s-%s", month, year)

	db, err := ec.getDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Consulta para obtener el resumen por tipo
	var typeSummaries []struct {
		Type  string
		Total float64
	}

	if err := db.Model(&models.Expenses{}).
		Select("type, sum(amount) as total").
		Where("date_time LIKE ?", datePattern).
		Group("type").
		Find(&typeSummaries).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Calcular el total general
	var total float64
	formattedTypeSummaries := make([]TypeSummary, len(typeSummaries))

	for i, ts := range typeSummaries {
		total += ts.Total
		formattedTypeSummaries[i] = TypeSummary{
			Type:           ts.Type,
			Total:          ts.Total,
			FormattedTotal: ec.formatAmount(ts.Total),
		}
	}

	// Construir la respuesta final
	response := ExpensesSummaryResponse{
		Total:          total,
		FormattedTotal: ec.formatAmount(total),
		Period:         period,
		TypesSummary:   formattedTypeSummaries,
	}

	c.JSON(http.StatusOK, response)
}
