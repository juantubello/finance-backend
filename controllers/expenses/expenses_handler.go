package expenses

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

	transactions "finance-backend/controllers/base"
)

type ExpenseController struct {
	*transactions.BaseController // Embed base to share base methods
}

func NewExpenseController() *ExpenseController {
	return &ExpenseController{
		BaseController: &transactions.BaseController{},
	}
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

func (ec *ExpenseController) GetExpenses(c *gin.Context) {
	type FormattedExpenseResponse struct {
		models.Expenses
		FormattedAmount string `json:"formatted_amount"`
	}

	yearStr := c.Query("year")
	monthStr := c.Query("month")

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

	db, err := ec.GetDatabaseInstance("TRANSACTION_DB")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var expenses []models.Expenses

	err = db.
		Where("strftime('%Y-%m', date) = ?", dateFilter).
		Group("date").
		Order("date DESC").
		Find(&expenses).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
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
				DateTime:    ec.FormatDate(exp.DateTime),
			},
			FormattedAmount: ec.FormatAmount(exp.Amount),
		}
	}

	c.JSON(http.StatusOK, gin.H{"Expenses": formatted})
}

func (ec *ExpenseController) GetExpensesSummary(c *gin.Context) {

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

	// The following block converts the "exclude" parameter into a slice of strings usable by GORM.
	// This allows filtering excluded types coming in the URL as a single string, for example:
	// ?exclude=[Rent and utilities, Other]
	// 1. Retrieves the full string using c.Query("exclude").
	// 2. Removes the brackets "[" and "]" using strings.Trim(..., "[]").
	// 3. Splits the string by commas using strings.Split(..., ",") to separate each type.
	// 4. Trims extra spaces with strings.TrimSpace(...) on each element.
	// The final result is []string{"Rent and utilities", "Other"}, ready to use in a WHERE NOT IN clause.

	rawExclude := c.Query("exclude")
	exclude := strings.Split(strings.Trim(rawExclude, "[]"), ",")
	for i := range exclude {
		exclude[i] = strings.TrimSpace(exclude[i])
	}

	yearStr := c.Query("year")
	monthStr := c.Query("month")

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

	db, err := ec.GetDatabaseInstance("TRANSACTION_DB")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Response structs for db query
	var typeSummaries []struct {
		Type  string
		Total float64
	}

	if err := db.Model(&models.Expenses{}).
		Select("type, sum(amount) as total").
		Where("strftime('%Y-%m', date) = ?", dateFilter).
		Where("type NOT IN ?", exclude).
		Group("type").
		Find(&typeSummaries).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Calculate total
	var total float64
	formattedTypeSummaries := make([]TypeSummary, len(typeSummaries))

	for i, ts := range typeSummaries {
		total += ts.Total
		formattedTypeSummaries[i] = TypeSummary{
			Type:           ts.Type,
			Total:          ts.Total,
			FormattedTotal: ec.FormatAmount(ts.Total),
		}
	}

	// Construir la respuesta final
	response := ExpensesSummaryResponse{
		Total:          total,
		FormattedTotal: ec.FormatAmount(total),
		Period:         dateFilter,
		TypesSummary:   formattedTypeSummaries,
	}

	c.JSON(http.StatusOK, gin.H{"ExpensesSummary": response})
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
		return nil, nil, fmt.Errorf("error at SyncData() on ReadSheet")
	}
	if len(data) <= 0 {
		return nil, nil, fmt.Errorf("no data found on spreadsheet at SyncData() ReadSheet()")
	}

	uuidsFromSheet, err := expenseSheetDataToMap(data)
	if err != nil {
		return nil, nil, fmt.Errorf("error trying to parse sheet data to map at ExpenseSheetDataToMap )")
	}

	db, err := ec.GetDatabaseInstance("TRANSACTION_DB")
	if err != nil {
		return nil, nil, fmt.Errorf("error trying to connect to database at getDB()")
	}

	var expenses []models.Expenses

	if !parameters.HistoricalSync {

		query := db.Distinct().Where("date_time LIKE ?", parameters.DatePattern)

		if parameters.DatePattern2 != "" {
			query = query.Or("date_time LIKE ?", parameters.DatePattern2)
		}

		if err := query.Find(&expenses).Error; err != nil {
			return nil, nil, fmt.Errorf("error trying to fetch expenses data with patterns")
		}
	}

	if parameters.HistoricalSync {
		if err := db.Find(&expenses).Error; err != nil {
			return nil, nil, fmt.Errorf("error trying to fetch all expenses")
		}
	}

	uuidsFromDataBase, err := expenseDatabaseDataToMap(expenses)
	if err != nil {
		return nil, nil, fmt.Errorf("error trying to parse database records to map")
	}

	expensesToInsert := getExpensesToInsert(uuidsFromSheet, uuidsFromDataBase)
	expensesToDelete := getExpensesToDelete(uuidsFromSheet, uuidsFromDataBase)

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

func expenseSheetDataToMap(data [][]interface{}) (map[string]models.Expenses, error) {
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

		// Parsear la fecha del formato "17/6/2025 18:11:40" a time.Time
		parsedDate, err := time.Parse("2/1/2006 15:04:05", row[DateTime].(string))
		if err != nil {
			return nil, fmt.Errorf("error parsing date at row %d: %v", i, err)
		}

		uuidStr := toString(row[UUID]) // Clave del mapa
		expensesMap[uuidStr] = models.Expenses{
			UUID:        uuidStr,
			DateTime:    toString(row[DateTime]),
			Description: toString(row[Description]),
			Amount:      parseAmount(row[Amount]),
			Type:        toString(row[Type]),
			Date:        parsedDate,
		}
	}

	return expensesMap, nil
}

func getExpensesToInsert(sheetData map[string]models.Expenses, databaseData map[string]models.Expenses) (expensesToInsert []models.Expenses) {
	for _, row := range sheetData {
		if _, exists := databaseData[row.UUID]; !exists {
			expensesToInsert = append(expensesToInsert, row)
		}
	}
	return expensesToInsert
}

func getExpensesToDelete(sheetData map[string]models.Expenses, databaseData map[string]models.Expenses) (expensesToDelete []models.Expenses) {
	for _, row := range databaseData {
		if _, exists := sheetData[row.UUID]; !exists {
			expensesToDelete = append(expensesToDelete, row)
		}
	}
	return expensesToDelete
}

func expenseDatabaseDataToMap(data []models.Expenses) (map[string]models.Expenses, error) {

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
			Date:        row.Date,
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
