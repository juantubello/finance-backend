package controllers

import (
	"finance-backend/config"
	"finance-backend/models"
	"finance-backend/services"
	"fmt"
	"log"
	"net/http"

	"strconv"
	"strings"
	"time"
	"unicode"

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

func (ec *ExpenseController) SyncCurrentMonthExpenses(c *gin.Context) {

	now := time.Now()
	//Format 01 for month (02 is for current day)
	month := now.Format("01")
	//Format "2006" for current year
	year := now.Format("2006")

	datePattern := fmt.Sprintf("%s-%s%%", year, month)
	spreadsheetID := config.GetEnv("GS_SPREADSHEET_ID")
	sheetName := config.GetEnv("GS_SHEET_ID")

	// Create google sheet instance
	sheetsReader, err := services.NewGoogleSheetsReader(spreadsheetID)
	if err != nil {
		log.Fatalf("Error trying to create a new google reader instance at SyncExpensesByMonth(): %v", err)
	}

	// Read data sheet
	sheetRange := "GastosMesActual!A:Z" // Lee todas las columnas
	data, err := sheetsReader.ReadSheet(sheetName, sheetRange)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(data) <= 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "spreadsheet is empty"})
		return
	}

	uuidsFromSheet, err := ParseExpenseSheetDataToMap(data)

	fmt.Println(uuidsFromSheet)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

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

	var expensesToDelete, expensesToInsert []models.Expenses

	uuidsFromDatabase := make(map[string]models.Expenses) // Mapa clave: UUID (string), valor: ExpenseSheet

	for _, row := range expenses {
		uuidStr := toString(row.UUID) // Clave del mapa
		uuidsFromDatabase[uuidStr] = models.Expenses{
			ID:          row.ID,
			UUID:        uuidStr,
			DateTime:    toString(row.DateTime),
			Description: toString(row.Description),
			Amount:      parseAmount(row.Amount),
			Type:        toString(row.Type),
		}

		//Use current loop to validate the uuids
		if _, exists := uuidsFromSheet[uuidStr]; !exists {
			expensesToDelete = append(expensesToDelete, row)
		}
	}

	for _, row := range uuidsFromSheet {
		if _, exists := uuidsFromDatabase[row.UUID]; !exists {
			expensesToInsert = append(expensesToInsert, row)
		}
	}

	fmt.Println("Rows to be deleted:")
	fmt.Println(expensesToDelete)
	fmt.Println("Rows to be inserted:")
	fmt.Println(expensesToInsert)
	c.JSON(http.StatusOK, "{status: ok}")
}

// GetExpenses obtiene los gastos filtrados por fecha
func (ec *ExpenseController) SyncExpensesHistorical(c *gin.Context) {
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

func ParseExpenseSheetDataToMap(data [][]interface{}) (map[string]models.Expenses, error) {
	const (
		DateTime    int8 = 0
		Amount      int8 = 1
		Description int8 = 2
		Type        int8 = 3
		UUID        int8 = 4
	)

	expensesMap := make(map[string]models.Expenses) // Mapa clave: UUID (string), valor: ExpenseSheet

	for i, row := range data {
		if i == 0 { // Saltar encabezados
			continue
		}

		uuidStr := toString(row[UUID]) // Clave del mapa
		expensesMap[uuidStr] = models.Expenses{
			UUID:        uuidStr,
			DateTime:    toString(row[DateTime]),
			Description: toString(row[Description]),
			Amount:      parseAmount(row[Amount]),
			Type:        toString(row[Type]),
		}
	}

	return expensesMap, nil
}

func toString(v interface{}) string { return fmt.Sprintf("%v", v) }
func parseAmount(amountStr interface{}) float64 {
	str := fmt.Sprintf("%v", amountStr)

	// Clean the string (remove $, commas, etc.)
	var cleaned strings.Builder
	hasDecimal := false
	for _, r := range str {
		switch {
		case r == '-' && cleaned.Len() == 0:
			cleaned.WriteRune(r)
		case unicode.IsDigit(r):
			cleaned.WriteRune(r)
		case r == '.' && !hasDecimal:
			cleaned.WriteRune(r)
			hasDecimal = true
		}
	}

	// If empty, return 0.0
	if cleaned.Len() == 0 {
		return 0.0
	}

	// Try parsing, return 0.0 if it fails
	amount, err := strconv.ParseFloat(cleaned.String(), 64)
	if err != nil {
		return 0.0
	}
	return amount
}
