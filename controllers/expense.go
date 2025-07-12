package controllers

import (
	"finance-backend/config"
	"finance-backend/models"
	"finance-backend/services"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gorm.io/gorm"
)

// BaseExpense contiene la lógica compartida
type BaseExpense struct{}

// ExpenseController implementa BaseExpense para poder utilizar sus metodos
type ExpenseController struct {
	*BaseExpense // Embedding para heredar métodos
}

// Expenses response inherit Expenses model and add new json field for formatted amount
type FormattedExpenseResponse struct {
	models.Expenses
	FormattedAmount string `json:"formatted_amount"`
}

// Summary types (items)
type TypeSummary struct {
	Type           string  `json:"type"`
	Total          float64 `json:"total"`
	FormattedTotal string  `json:"formatted_total"`
}

// Summary overview (header)
type ExpensesSummaryResponse struct {
	Total          float64       `json:"total"`
	FormattedTotal string        `json:"formatted_total"`
	Period         string        `json:"period"`
	TypesSummary   []TypeSummary `json:"types_summary"`
}

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

// NewExpenseController crea una nueva instancia del controlador
func NewExpenseController() *ExpenseController {
	return &ExpenseController{
		BaseExpense: &BaseExpense{},
	}
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

// GetExpensesSummary obtiene el resumen de gastos por categoría,
// hereda de ExpenseController para obtener los metodos base de obtener
// base de datos y formateo a pesos
func (ec *ExpenseController) GetExpensesSummary(c *gin.Context) {
	year := c.Query("year")
	month := c.Query("month")
	datePattern := fmt.Sprintf("%s-%s%%", year, month)
	period := fmt.Sprintf("%s-%s", month, year)

	spreadsheetID := config.GetEnv("GS_SPREADSHEET_ID")
	sheetName := config.GetEnv("GS_SHEET_ID")

	// Crear instancia del lector
	sheetsReader, err := services.NewGoogleSheetsReader(spreadsheetID)
	if err != nil {
		log.Fatalf("Error al crear lector de Google Sheets: %v", err)
	}

	// Opción 1: Leer datos y procesarlos manualmente
	sheetRange := "Gastos!A:Z" // Lee todas las columnas
	data, err := sheetsReader.ReadSheet(sheetName, sheetRange)
	fmt.Println(data)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Se leyeron %d filas (incluyendo encabezados)", len(data))

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
