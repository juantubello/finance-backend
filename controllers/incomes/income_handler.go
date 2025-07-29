package incomes

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

type IncomeController struct {
	*transactions.BaseController // Embed base to share base methods
}

func NewIncomeController() *IncomeController {
	return &IncomeController{
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
func (ec *IncomeController) GetIncomes(c *gin.Context) {

	// Expenses response inherit Expenses model and add new json field for formatted amount
	type FormattedIncomesResponse struct {
		models.Incomes
		FormattedAmount string `json:"formatted_amount"`
	}

	type incomesResponse struct {
		IncomeTotal          float64                    `json:"income_total"`
		IncomeTotalFormatted string                     `json:"income_total_formatted"`
		IncomesDetail        []FormattedIncomesResponse `json:"incomes_details"`
	}

	var totalIncome []struct {
		Total float64
	}

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

	datePattern2 := fmt.Sprintf("%%/%s/%s%%", new_month_format, year)

	db, err := ec.GetDatabaseInstance("TRANSACTION_DB")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var incomes []models.Incomes

	query := db.Distinct().Where("date_time LIKE ?", datePattern)
	if datePattern2 != "" {
		query = query.Or("date_time LIKE ?", datePattern2)
	}

	query = query.Order("id DESC")

	if err := query.Find(&incomes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	formatted := make([]FormattedIncomesResponse, len(incomes))
	for i, exp := range incomes {
		formatted[i] = FormattedIncomesResponse{
			Incomes: models.Incomes{
				ID:          exp.ID,
				UUID:        exp.UUID,
				Description: exp.Description,
				Amount:      exp.Amount,
				Currency:    exp.Currency,
				DateTime:    ec.FormatDate(exp.DateTime), // acá el cambio
			},
			FormattedAmount: ec.FormatAmount(exp.Amount),
		}
	}

	//Calculate total
	//Utilizing query again, because it already has the previous where / distintc etc filters applied, just changing Select condition
	if err := query.Select("sum(amount) as total").Find(&totalIncome).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := incomesResponse{
		IncomeTotal:          totalIncome[0].Total,
		IncomeTotalFormatted: ec.FormatAmount(totalIncome[0].Total),
		IncomesDetail:        formatted,
	}

	c.JSON(http.StatusOK, response)
}

func (ec *IncomeController) SyncCurrentMonthIncomes(c *gin.Context) {

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
	sheetRange := "IncomeMesActual!A:Z" // Lee todas las columnas

	syncParameters := SyncIncomeData{
		HistoricalSync: false,
		DatePattern:    datePattern,
		DatePattern2:   datePattern2,
		SheetId:        spreadsheetID,
		SheetName:      sheetName,
		SheetRange:     sheetRange,
	}

	incomesInserted, incomesDeleted, err := SyncData(syncParameters)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create json response
	response := IncomeSyncResponse{
		DeletedRows:        len(incomesDeleted),
		DeletedRowsDetail:  incomesDeleted,
		InsertedRows:       len(incomesInserted),
		InsertedRowsDetail: incomesInserted,
	}

	c.JSON(http.StatusOK, response)
}

func (ec *IncomeController) SyncIncomesHistorical(c *gin.Context) {

	spreadsheetID := config.GetEnv("GS_SPREADSHEET_ID")
	sheetName := config.GetEnv("GS_SHEET_ID")
	sheetRange := "Income!A:Z" // Lee todas las columnas

	syncParameters := SyncIncomeData{
		HistoricalSync: true,
		SheetId:        spreadsheetID,
		SheetName:      sheetName,
		SheetRange:     sheetRange,
	}

	incomesInserted, incomesDeleted, err := SyncData(syncParameters)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create json response
	response := IncomeSyncResponse{
		DeletedRows:        len(incomesDeleted),
		DeletedRowsDetail:  incomesDeleted,
		InsertedRows:       len(incomesInserted),
		InsertedRowsDetail: incomesInserted,
	}

	c.JSON(http.StatusOK, response)

}

func SyncData(parameters SyncIncomeData) (incomesInserted []models.Incomes, incomesDeleted []models.Incomes, Error error) {

	ec := NewIncomeController()

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

	uuidsFromSheet, err := incomeSheetDataToMap(data)
	if err != nil {
		return nil, nil, fmt.Errorf("error trying to parse sheet data to map at ExpenseSheetDataToMap ): %w", err)
	}

	db, err := ec.GetDatabaseInstance("TRANSACTION_DB")
	if err != nil {
		return nil, nil, fmt.Errorf("error trying to connect to database at getDB()")
	}

	var incomes []models.Incomes

	if !parameters.HistoricalSync {

		query := db.Distinct().Where("date_time LIKE ?", parameters.DatePattern)

		if parameters.DatePattern2 != "" {
			query = query.Or("date_time LIKE ?", parameters.DatePattern2)
		}

		if err := query.Find(&incomes).Error; err != nil {
			return nil, nil, fmt.Errorf("error trying to fetch expenses data with patterns: %w", err)
		}
	}

	if parameters.HistoricalSync {
		if err := db.Find(&incomes).Error; err != nil {
			return nil, nil, fmt.Errorf("error trying to fetch all expenses")
		}
	}

	uuidsFromDataBase, err := incomesDatabaseDataToMap(incomes)
	if err != nil {
		return nil, nil, fmt.Errorf("error trying to parse database records to map")
	}

	incomesToInsert := getIncomesToInsert(uuidsFromSheet, uuidsFromDataBase)
	incomesToDelete := getIncomesToDelete(uuidsFromSheet, uuidsFromDataBase)

	//Handle records insertions
	if len(incomesToInsert) > 0 {
		db.Create(&incomesToInsert)
	}

	//Handle records deletions, it will delete by primary key ID
	if len(incomesToDelete) > 0 {
		db.Delete(&incomesToDelete)
	}

	return incomesToInsert, incomesToDelete, nil
}

func incomeSheetDataToMap(data [][]interface{}) (map[string]models.Incomes, error) {
	const (
		DateTime    int8 = 0
		Amount      int8 = 1
		Description int8 = 3
		Currency    int8 = 2
		UUID        int8 = 4
	)

	incomesMap := make(map[string]models.Incomes) // Mapa clave: UUID (string), valor: ExpenseSheet

	for i, row := range data {

		if i == 0 { // Saltar encabezados
			continue
		}
		uuidStr := toString(row[UUID]) // Clave del mapa
		incomesMap[uuidStr] = models.Incomes{
			UUID:        uuidStr,
			DateTime:    toString(row[DateTime]),
			Description: toString(row[Description]),
			Amount:      parseAmount(row[Amount]),
			Currency:    toString(row[Currency]),
		}
	}

	return incomesMap, nil
}

func getIncomesToInsert(sheetData map[string]models.Incomes, databaseData map[string]models.Incomes) (incomesToInsert []models.Incomes) {
	for _, row := range sheetData {
		if _, exists := databaseData[row.UUID]; !exists {
			incomesToInsert = append(incomesToInsert, row)
		}
	}
	return incomesToInsert
}

func getIncomesToDelete(sheetData map[string]models.Incomes, databaseData map[string]models.Incomes) (incomesToDelete []models.Incomes) {
	for _, row := range databaseData {
		if _, exists := sheetData[row.UUID]; !exists {
			incomesToDelete = append(incomesToDelete, row)
		}
	}
	return incomesToDelete
}

func incomesDatabaseDataToMap(data []models.Incomes) (map[string]models.Incomes, error) {

	incomesMap := make(map[string]models.Incomes, len(data)) // Mapa clave: UUID (string), valor: ExpenseSheet

	for _, row := range data {
		uuidStr := toString(row.UUID) // Clave del mapa
		incomesMap[uuidStr] = models.Incomes{
			ID:          row.ID,
			UUID:        uuidStr,
			DateTime:    toString(row.DateTime),
			Description: toString(row.Description),
			Amount:      parseAmount(row.Amount),
			Currency:    toString(row.Currency),
		}
	}

	return incomesMap, nil
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
