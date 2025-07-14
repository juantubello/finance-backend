package controllers

import (
	"finance-backend/config"
	"finance-backend/models"
	"finance-backend/services"
	"fmt"
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

type ExpenseSyncResponse struct {
	DeletedRows        int               `json:"rows_deleted"`
	DeletedRowsDetail  []models.Expenses `json:"deleted_rows_detail"`
	InsertedRows       int               `json:"inserted_rows"`
	InsertedRowsDetail []models.Expenses `json:"inserted_rows_detail"`
}

type SyncExpenseData struct {
	HistoricalSync bool
	DatePattern    string
	DatePattern2   string
	SheetId        string
	SheetName      string
	SheetRange     string
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

func (e *BaseExpense) formatDate(dateStr string) string {
	// Try to parse as  ISO 8601: "2025-07-01T10:03:03"
	t, err := time.Parse("2006-01-02T15:04:05", dateStr)
	if err != nil {
		// If fails, return original formar (assume is already ok)
		return dateStr
	}

	// Return format "10/7/2025 17:16:31"
	return t.Format("2/1/2006 15:04:05")
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

	new_month_format := month

	// -> '' is for bytes and runes  -> "" is for strings
	//month[1:]
	//Esto es slicing de strings. Devuelve una subcadena que comienza en la posición 1 hasta el final.
	//Por ejemplo:
	//"07"[1:] → "7"
	//"11"[1:] → "1" (esto probablemente no lo querés si el mes es válido)

	if month[0] == '0' {
		new_month_format = month[1:]
	} else {
		new_month_format = month
	}

	datePatternNew := fmt.Sprintf("%%/%s/%s%%", new_month_format, year)

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

	var expensesNewFormat []models.Expenses
	if err := db.Where("date_time LIKE ?", datePatternNew).Find(&expensesNewFormat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(expensesNewFormat) > 0 {
		// the 3 dots ... means we are appending a slice
		expenses = append(expenses, expensesNewFormat...)
	}

	formatted := make([]FormattedExpenseResponse, len(expenses))
	for i, exp := range expenses {
		formatted[i] = FormattedExpenseResponse{
			Expenses: models.Expenses{
				ID:          exp.ID,
				UUID:        exp.UUID,
				Description: exp.Description,
				Amount:      exp.Amount,
				Type:        exp.Type,
				DateTime:    ec.formatDate(exp.DateTime), // acá el cambio
			},
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

	new_month_format := month

	if month[0] == '0' {
		new_month_format = month[1:]
	} else {
		new_month_format = month
	}

	datePattern2 := fmt.Sprintf("%%/%s/%s%%", new_month_format, year)

	spreadsheetID := config.GetEnv("GS_SPREADSHEET_ID")
	sheetName := config.GetEnv("GS_SHEET_ID")
	sheetRange := "GastosMesActual!A:Z" // Lee todas las columnas

	syncParameters := SyncExpenseData{
		HistoricalSync: false,
		DatePattern:    datePattern,
		DatePattern2:   datePattern2,
		SheetId:        spreadsheetID,
		SheetName:      sheetName,
		SheetRange:     sheetRange,
	}

	expensesInserted, expensesDeleted, err := SyncData(syncParameters)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create json response
	response := ExpenseSyncResponse{
		DeletedRows:        len(expensesDeleted),
		DeletedRowsDetail:  expensesDeleted,
		InsertedRows:       len(expensesInserted),
		InsertedRowsDetail: expensesInserted,
	}

	c.JSON(http.StatusOK, response)
}

func (ec *ExpenseController) SyncExpensesHistorical(c *gin.Context) {

	spreadsheetID := config.GetEnv("GS_SPREADSHEET_ID")
	sheetName := config.GetEnv("GS_SHEET_ID")
	sheetRange := "Gastos!A:Z" // Lee todas las columnas

	syncParameters := SyncExpenseData{
		HistoricalSync: true,
		SheetId:        spreadsheetID,
		SheetName:      sheetName,
		SheetRange:     sheetRange,
	}

	expensesInserted, expensesDeleted, err := SyncData(syncParameters)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create json response
	response := ExpenseSyncResponse{
		DeletedRows:        len(expensesDeleted),
		DeletedRowsDetail:  expensesDeleted,
		InsertedRows:       len(expensesInserted),
		InsertedRowsDetail: expensesInserted,
	}

	c.JSON(http.StatusOK, response)

}

func SyncData(parameters SyncExpenseData) (expensesInserted []models.Expenses, expensesDeleted []models.Expenses, Error error) {

	ec := NewExpenseController()

	// Create google sheet instance
	sheetsReader, err := services.NewGoogleSheetsReader(parameters.SheetId)
	if err != nil {
		return nil, nil, fmt.Errorf("error trying to create a new google reader instance at SyncExpensesByMonth(): %w", err)
	}

	// Read data sheet
	data, err := sheetsReader.ReadSheet(parameters.SheetName, parameters.SheetRange)

	if err != nil {
		return nil, nil, fmt.Errorf("error at SyncData() on ReadSheet: %w", err)
	}
	if len(data) <= 0 {
		return nil, nil, fmt.Errorf("no data found on spreadsheet at SyncData() ReadSheet(): %w", err)
	}

	uuidsFromSheet, err := ExpenseSheetDataToMap(data)
	if err != nil {
		return nil, nil, fmt.Errorf("error trying to parse sheet data to map at ExpenseSheetDataToMap ): %w", err)
	}

	db, err := ec.getDB()
	if err != nil {
		return nil, nil, fmt.Errorf("error trying to connect to database at getDB() ): %w", err.Error())
	}

	var expenses []models.Expenses

	if !parameters.HistoricalSync {

		if err := db.Where("date_time LIKE ?", parameters.DatePattern).Find(&expenses).Error; err != nil {
			return nil, nil, fmt.Errorf("error trying to fetch expenses data ): %w", err.Error())
		}

		if parameters.DatePattern2 != "" {
			var expensesNewFormat []models.Expenses
			if err := db.Where("date_time LIKE ?", parameters.DatePattern2).Find(&expensesNewFormat).Error; err != nil {
				return nil, nil, fmt.Errorf("error trying to fetch expenses data with new pattern ): %w", err.Error())
			}

			if len(expensesNewFormat) > 0 {
				// the 3 dots ... means we are appending a slice
				expenses = append(expenses, expensesNewFormat...)
			}
		}
	}

	if parameters.HistoricalSync {
		if err := db.Find(&expenses).Error; err != nil {
			return nil, nil, fmt.Errorf("error trying to fetch all expenses ): %w", err.Error())
		}
	}

	uuidsFromDataBase, err := ExpenseDatabaseDataToMap(expenses)
	if err != nil {
		return nil, nil, fmt.Errorf("error trying to parse database records to map ): %w", err.Error())
	}

	expensesToInsert := GetExpensesToInsert(uuidsFromSheet, uuidsFromDataBase)
	expensesToDelete := GetExpensesToDelete(uuidsFromSheet, uuidsFromDataBase)

	//Handle records insertions
	if len(expensesToInsert) > 0 {
		db.Create(&expensesToInsert)
	}

	//Handle records deletions, it will delete by primary key ID
	if len(expensesToDelete) > 0 {
		db.Delete(&expensesToDelete)
	}

	return expensesToInsert, expensesToDelete, nil
}

func ExpenseSheetDataToMap(data [][]interface{}) (map[string]models.Expenses, error) {
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

func GetExpensesToInsert(sheetData map[string]models.Expenses, databaseData map[string]models.Expenses) (expensesToInsert []models.Expenses) {
	for _, row := range sheetData {
		if _, exists := databaseData[row.UUID]; !exists {
			expensesToInsert = append(expensesToInsert, row)
		}
	}
	return expensesToInsert
}

func GetExpensesToDelete(sheetData map[string]models.Expenses, databaseData map[string]models.Expenses) (expensesToDelete []models.Expenses) {
	for _, row := range databaseData {
		if _, exists := sheetData[row.UUID]; !exists {
			expensesToDelete = append(expensesToDelete, row)
		}
	}
	return expensesToDelete
}

func ExpenseDatabaseDataToMap(data []models.Expenses) (map[string]models.Expenses, error) {

	expensesMap := make(map[string]models.Expenses, len(data)) // Mapa clave: UUID (string), valor: ExpenseSheet

	for _, row := range data {
		uuidStr := toString(row.UUID) // Clave del mapa
		expensesMap[uuidStr] = models.Expenses{
			ID:          row.ID,
			UUID:        uuidStr,
			DateTime:    toString(row.DateTime),
			Description: toString(row.Description),
			Amount:      parseAmount(row.Amount),
			Type:        toString(row.Type),
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
