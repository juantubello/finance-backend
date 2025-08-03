package cards

import (
	"finance-backend/config"
	cards "finance-backend/controllers/base"
	"finance-backend/models"
	"finance-backend/services"
	"finance-backend/utils"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type CardsController struct {
	*cards.BaseController // Embed base to share base methods
}

func NewCardsController() *CardsController {
	return &CardsController{
		BaseController: &cards.BaseController{},
	}
}

type resumePaths struct {
	CardLogo string `json:"cardLogo"`
	FilePath string `json:"filePath"`
	FileName string `json:"fileName"`
}

type ResumeDetails struct {
	Holder   string             `json:"holder"`
	Expenses []services.Expense `json:"expenses"`
	Totals   services.Totals    `json:"totals"`
}

type ResumesData struct {
	CardLogo   string          `json:"cardLogo"`
	FileName   string          `json:"fileName"`
	Hash       string          `json:"hash"`
	ResumeData []ResumeDetails `json:"resumeData"`
	Totals     services.Totals `json:"totals"`
}

type SubscriptionSummary struct {
	Servicio             string  `json:"service"`
	TotalAmount          float64 `json:"total_amount"`
	TotalAmountFormatted string  `json:"total_amount_formatted"`
	LogoName             string  `json:"logo_name"`
}

type subscriptionQueryResult struct {
	Servicio             string  `gorm:"column:servicio"`
	ReferenceDescription string  `gorm:"column:reference_description"`
	TotalAmount          float64 `gorm:"column:total_amount"`
}

type cuotasAboutToExpire struct {
	Description     string  `gorm:"column:description"`
	FormattedAmount string  `gorm:"column:formatted_amount"`
	Amount          float64 `gorm:"column:amount"`
}

type CuotasAboutToExpireSummary struct {
	Description     string  `gorm:"column:description"`
	FormattedAmount string  `gorm:"column:formatted_amount"`
	Amount          float64 `gorm:"column:amount"`
	LogoName        string  `json:"logo_name"`
}

func (ec *CardsController) GetCuotasAboutToExpire(c *gin.Context) {
	year := c.Query("year")
	month := c.Query("month")

	yearInt, err := strconv.Atoi(year)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid year"})
		return
	}
	monthInt, err := strconv.Atoi(month)
	if err != nil || monthInt < 1 || monthInt > 12 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid month"})
		return
	}
	targetMonth := fmt.Sprintf("%04d-%02d", yearInt, monthInt)

	subscriptionMap := utils.LoadMap("SPECIFIC_EXPENSES_MAP")
	//subscriptionLogoMap := utils.LoadLogosMap("SPECIFIC_LOGO_MAP")

	if len(subscriptionMap) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subscription map is empty"})
		return
	}

	db, err := ec.GetDatabaseInstance("CARDS_DB")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var rawResults []cuotasAboutToExpire
	var finalResults []CuotasAboutToExpireSummary

	// Ejecutar query
	tx := db.Table("holder_expenses AS e").
		Select("e.description, e.amount, e.formatted_amount").
		Joins("JOIN holders h ON e.document_number = h.document_number AND e.holder = h.holder").
		Joins("JOIN resumes r ON h.document_number = r.document_number").
		Where("strftime('%Y-%m', r.resume_date) = ?", targetMonth).
		Where(" LOWER(e.description) LIKE '%C.%'").
		Order("amount DESC").
		Scan(&rawResults)

	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": tx.Error.Error()})
		return
	}

	fmt.Println(rawResults)

	for _, row := range rawResults {
		description := row.Description

		// logo := subscriptionLogoMap[baseLogoKey]
		// if logo == "" {
		// 	logo = "default.png"
		// }

		finalResults = append(finalResults, CuotasAboutToExpireSummary{
			Description:     description,
			Amount:          row.Amount,
			FormattedAmount: row.FormattedAmount,
			LogoName:        "",
		})
	}

	c.JSON(http.StatusOK, finalResults)
}

func (ec *CardsController) GetSpecificCardExpenes(c *gin.Context) {
	year := c.Query("year")
	month := c.Query("month")

	yearInt, err := strconv.Atoi(year)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid year"})
		return
	}
	monthInt, err := strconv.Atoi(month)
	if err != nil || monthInt < 1 || monthInt > 12 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid month"})
		return
	}
	targetMonth := fmt.Sprintf("%04d-%02d", yearInt, monthInt)

	subscriptionMap := utils.LoadMap("SPECIFIC_EXPENSES_MAP")
	subscriptionLogoMap := utils.LoadLogosMap("SPECIFIC_LOGO_MAP")

	if len(subscriptionMap) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subscription map is empty"})
		return
	}

	// CASE WHEN dinámico
	caseExpr := "CASE\n"
	for keyword, label := range subscriptionMap {
		caseExpr += fmt.Sprintf("  WHEN LOWER(e.description) LIKE '%%%s%%' THEN '%s'\n", keyword, label)
	}
	caseExpr += "  ELSE 'otro'\nEND AS servicio"

	// WHERE dinámico
	var likeConditions []string
	var likeParams []interface{}
	for keyword := range subscriptionMap {
		likeConditions = append(likeConditions, "LOWER(e.description) LIKE ?")
		likeParams = append(likeParams, fmt.Sprintf("%%%s%%", keyword))
	}
	whereClause := "(" + strings.Join(likeConditions, " OR ") + ")"

	db, err := ec.GetDatabaseInstance("CARDS_DB")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var rawResults []subscriptionQueryResult
	var finalResults []SubscriptionSummary

	// Ejecutar query
	tx := db.Table("holder_expenses AS e").
		Select(caseExpr+", MAX(e.description) AS reference_description, SUM(e.amount) AS total_amount").
		Joins("JOIN holders h ON e.document_number = h.document_number AND e.holder = h.holder").
		Joins("JOIN resumes r ON h.document_number = r.document_number").
		Where("strftime('%Y-%m', r.resume_date) = ?", targetMonth).
		Where(whereClause, likeParams...).
		Group("servicio").
		Order("total_amount DESC").
		Scan(&rawResults)

	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": tx.Error.Error()})
		return
	}

	for _, row := range rawResults {
		service := row.Servicio
		if strings.Contains(strings.ToUpper(row.ReferenceDescription), "USD") && !strings.Contains(service, "USD") {
			service += " USD"
		}

		// Logo se basa en el nombre sin el " USD"
		baseLogoKey := strings.Replace(service, " USD", "", 1)
		logo := subscriptionLogoMap[baseLogoKey]
		if logo == "" {
			logo = "default.png"
		}

		finalResults = append(finalResults, SubscriptionSummary{
			Servicio:             service,
			TotalAmount:          row.TotalAmount,
			TotalAmountFormatted: ec.FormatAmount(row.TotalAmount),
			LogoName:             logo,
		})
	}

	c.JSON(http.StatusOK, finalResults)
}

func (ec *CardsController) GetSubscriptionSummary(c *gin.Context) {
	year := c.Query("year")
	month := c.Query("month")

	yearInt, err := strconv.Atoi(year)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid year"})
		return
	}
	monthInt, err := strconv.Atoi(month)
	if err != nil || monthInt < 1 || monthInt > 12 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid month"})
		return
	}
	targetMonth := fmt.Sprintf("%04d-%02d", yearInt, monthInt)

	subscriptionMap := utils.LoadMap("SUBSCRIPTION_MAP")
	subscriptionLogoMap := utils.LoadLogosMap("SUBSCRIPTION_LOGO_MAP")

	if len(subscriptionMap) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subscription map is empty"})
		return
	}

	// CASE WHEN dinámico
	caseExpr := "CASE\n"
	for keyword, label := range subscriptionMap {
		caseExpr += fmt.Sprintf("  WHEN LOWER(e.description) LIKE '%%%s%%' THEN '%s'\n", keyword, label)
	}
	caseExpr += "  ELSE 'otro'\nEND AS servicio"

	// WHERE dinámico
	var likeConditions []string
	var likeParams []interface{}
	for keyword := range subscriptionMap {
		likeConditions = append(likeConditions, "LOWER(e.description) LIKE ?")
		likeParams = append(likeParams, fmt.Sprintf("%%%s%%", keyword))
	}
	whereClause := "(" + strings.Join(likeConditions, " OR ") + ")"

	db, err := ec.GetDatabaseInstance("CARDS_DB")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var rawResults []subscriptionQueryResult
	var finalResults []SubscriptionSummary

	// Ejecutar query
	tx := db.Table("holder_expenses AS e").
		Select(caseExpr+", MAX(e.description) AS reference_description, SUM(e.amount) AS total_amount").
		Joins("JOIN holders h ON e.document_number = h.document_number AND e.holder = h.holder").
		Joins("JOIN resumes r ON h.document_number = r.document_number").
		Where("strftime('%Y-%m', r.resume_date) = ?", targetMonth).
		Where(whereClause, likeParams...).
		Group("servicio").
		Order("total_amount DESC").
		Scan(&rawResults)

	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": tx.Error.Error()})
		return
	}

	for _, row := range rawResults {
		service := row.Servicio
		if strings.Contains(strings.ToUpper(row.ReferenceDescription), "USD") && !strings.Contains(service, "USD") {
			service += " USD"
		}

		// Logo se basa en el nombre sin el " USD"
		baseLogoKey := strings.Replace(service, " USD", "", 1)
		logo := subscriptionLogoMap[baseLogoKey]
		if logo == "" {
			logo = "default.png"
		}

		finalResults = append(finalResults, SubscriptionSummary{
			Servicio:             service,
			TotalAmount:          row.TotalAmount,
			TotalAmountFormatted: ec.FormatAmount(row.TotalAmount),
			LogoName:             logo,
		})
	}

	c.JSON(http.StatusOK, finalResults)
}

func (ec *CardsController) GetCardsExpenses(c *gin.Context) {

	year := c.Query("year")
	month := c.Query("month")
	cardType := strings.ToLower(c.DefaultQuery("card_type", "all"))
	holderFilter := strings.ToLower(c.DefaultQuery("holder", "all"))
	monthsBackStr := c.DefaultQuery("months_back", "")

	// Validate and build the base string "YYYY-MM"
	yearInt, err := strconv.Atoi(year)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid year"})
		return
	}
	monthInt, err := strconv.Atoi(month)
	if err != nil || monthInt < 1 || monthInt > 12 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid month"})
		return
	}

	// Create a slice of valid months to compare with strftime('%Y-%m', resume_date)
	var monthFilters []string
	targetDate := time.Date(yearInt, time.Month(monthInt), 1, 0, 0, 0, 0, time.UTC)

	monthsBack := 0
	if monthsBackStr != "" {
		monthsBack, _ = strconv.Atoi(monthsBackStr)
	}
	for i := 0; i <= monthsBack; i++ {
		d := targetDate.AddDate(0, -i, 0)
		monthFilters = append(monthFilters, d.Format("2006-01"))
	}

	// Build query with strftime
	db, err := ec.GetDatabaseInstance("CARDS_DB")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	query := db.Preload("Holders.Expenses")
	query = query.Where("strftime('%Y-%m', resume_date) IN ?", monthFilters)

	if cardType != "all" {
		query = query.Where("LOWER(card_type) = ?", cardType)
	}

	var resumes []models.Resume
	err = query.Find(&resumes).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if monthsBackStr != "" {

	}

	// Filter holders in memory if specified

	var noHolders = false

	if holderFilter != "all" {
		for i := range resumes {
			var filteredHolders []models.Holder
			for _, h := range resumes[i].Holders {
				if strings.ToLower(h.Holder) == holderFilter {
					filteredHolders = append(filteredHolders, h)
				}
			}
			resumes[i].Holders = filteredHolders

			if len(resumes[i].Holders) == 0 {
				noHolders = true
			}

		}
	}

	if noHolders {
		c.JSON(http.StatusOK, gin.H{"result": "No holders found for the specified filter"})
		return
	}

	c.JSON(http.StatusOK, resumes)
}

func (ec *CardsController) SyncResumes(c *gin.Context) {

	type JSONResponse struct {
		ResumeDate string `json:"resumeDate"`
		Hash       string `json:"hash"`
		Message    string `json:"message"`
		CardType   string `json:"cardType"`
	}

	var resumes []models.Resume
	var holders []models.Holder
	var holdersExpenses []models.HolderExpense
	var response []JSONResponse

	resumesPath, err := getResumesFilePath()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resumesParsedData, err := getResumeData(resumesPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// ---------- Database records population ----------

	//Todo - Maybe change this logic to avoid this triple nested loop, for our use case it is not a problem ATM
	for _, resume := range resumesParsedData {

		for _, holder := range resume.ResumeData {

			for _, expense := range holder.Expenses {

				holdersExpenses = append(holdersExpenses, models.HolderExpense{
					DocumentNumber:  resume.Hash,
					Holder:          holder.Holder,
					Position:        len(holdersExpenses) + 1,
					Date:            expense.Date.Format("2006-01-02"), // Convert to string in YYYY-MM-DD format
					Description:     expense.Description,
					Amount:          expense.Amount,
					FormattedAmount: ec.FormatAmount(expense.Amount),
				})
			}

			holders = append(holders, models.Holder{
				DocumentNumber:    resume.Hash,
				Holder:            holder.Holder,
				TotalARS:          holder.Totals.ARS,
				FormattedTotalARS: ec.FormatAmount(holder.Totals.ARS),
				TotalUSD:          holder.Totals.USD,
				FormattedTotalUSD: ec.FormatAmount(holder.Totals.USD),
				Expenses:          holdersExpenses,
			})

			holdersExpenses = nil // Reset for next holder

		}

		resumeDate, err := parseMonthYear(resume.FileName)
		if err != nil {
			continue // Skip this resume if date parsing fails
		}

		resumes = append(resumes, models.Resume{
			DocumentNumber:    resume.Hash,
			Holders:           holders,
			CardType:          resume.CardLogo,
			ResumeDate:        resumeDate.Format("2006-01-02"), // Convert to string in YYYY-MM-DD format
			TotalARS:          resume.Totals.ARS,
			FormattedTotalARS: ec.FormatAmount(resume.Totals.ARS),
			TotalUSD:          resume.Totals.USD,
			FormattedTotalUSD: ec.FormatAmount(resume.Totals.USD),
		})

		holders = nil // Reset for next resume

	}

	// ---------- Database insertions ----------

	db, err := ec.GetDatabaseInstance("CARDS_DB")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for _, resume := range resumes {
		//check if the resume already exists on database
		existingResume := models.Resume{}
		db.Where("document_number = ?", resume.DocumentNumber).First(&existingResume)

		if existingResume.DocumentNumber != "" {
			response = append(response, JSONResponse{
				CardType:   resume.CardType,
				ResumeDate: resume.ResumeDate,
				Hash:       resume.DocumentNumber,
				Message:    "Resume already exists",
			})
		} else {
			result := db.Model(&models.Resume{}).Create(&resume)
			if result.Error != nil {
				response = append(response, JSONResponse{
					CardType:   resume.CardType,
					ResumeDate: resume.ResumeDate,
					Hash:       resume.DocumentNumber,
					Message:    "Error creating resume",
				})
			} else {
				response = append(response, JSONResponse{
					CardType:   resume.CardType,
					ResumeDate: resume.ResumeDate,
					Hash:       resume.DocumentNumber,
					Message:    "Resume created successfully",
				})
			}

		}
	}

	c.JSON(http.StatusOK, gin.H{"Resumes sync status": response})
}

func getResumesFilePath() ([]resumePaths, error) {

	type directoriesPath struct {
		path     string
		cardLogo string
	}

	var resumesPath []resumePaths
	directories := make([]directoriesPath, 2)
	directoryInfo := directoriesPath{
		path:     config.GetEnv("CARD_VISA_PATH"),
		cardLogo: "visa",
	}
	directories[0] = directoryInfo

	directoryInfo = directoriesPath{
		path:     config.GetEnv("CARD_MASTERCARD_PATH"),
		cardLogo: "mastercard",
	}

	directories[1] = directoryInfo

	for _, dir := range directories {
		entries, err := os.ReadDir(dir.path)
		if err != nil {
			return nil, fmt.Errorf("error reading directory %s: %w", dir.path, err)
		}
		for _, v := range entries {
			if filepath.Ext(v.Name()) != ".pdf" {
				continue
			}
			completePath := dir.path + "/" + v.Name()
			fmt.Println(completePath)
			resumesPath = append(resumesPath, resumePaths{
				CardLogo: dir.cardLogo,
				FilePath: completePath,
				FileName: v.Name()[:len(v.Name())-len(filepath.Ext(v.Name()))],
			})

		}
	}

	return resumesPath, nil
}

func getResumeData(paths []resumePaths) ([]ResumesData, error) {

	var ResumeData []ResumesData
	var ResumeDetail []ResumeDetails

	bbvaReader, err := services.NewPdfReaderBBVA()
	if err != nil {
		return nil, fmt.Errorf("error trying to create a BBVA PDF reader instance at getResumeData(): %w", err)
	}

	for _, path := range paths {
		holders, totals, hash, err := bbvaReader.ReadResumes(services.ResumePath{
			CardLogo: path.CardLogo,
			FilePath: path.FilePath,
			FileName: path.FileName,
		})

		if err != nil {
			fmt.Println("Error abriendo el archivo:", err)
			continue
		}

		for _, holder := range holders {
			detail := ResumeDetails{
				Holder:   holder.Holder,
				Expenses: holder.Expenses,
				Totals:   holder.Totals,
			}
			ResumeDetail = append(ResumeDetail, detail)
		}

		header := ResumesData{
			CardLogo:   path.CardLogo,
			FileName:   path.FileName,
			Hash:       hash,
			ResumeData: ResumeDetail,
			Totals:     totals,
		}

		ResumeData = append(ResumeData, header)
		ResumeDetail = nil // Reset for next iteration
	}

	return ResumeData, nil

}

func parseMonthYear(input string) (time.Time, error) {
	// Formato: "MM-YYYY"
	layout := "01-2006"
	t, err := time.Parse(layout, input)
	if err != nil {
		return time.Time{}, err
	}

	// Ajustar al primer día del mes
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC), nil
}
