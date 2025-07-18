package cards

import (
	"finance-backend/config"
	cards "finance-backend/controllers/base"
	"finance-backend/models"
	"finance-backend/services"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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

func (ec *CardsController) SyncResumes(c *gin.Context) {

	var resumes []models.Resume
	var holders []models.Holder
	var holdersExpenses []models.HolderExpense

	resumesPath, err := getResumesFilePath()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resumesParsedData := getResumeData(resumesPath)

	for _, resume := range resumesParsedData {

		for _, holder := range resume.ResumeData {

			for _, expense := range holder.Expenses {

				holdersExpenses = append(holdersExpenses, models.HolderExpense{
					DocumentNumber: resume.Hash,
					Holder:         holder.Holder,
					Position:       len(holdersExpenses) + 1,
					Date:           expense.Date,
					Description:    expense.Description,
					Amount:         expense.Amount,
				})
			}

			holders = append(holders, models.Holder{
				DocumentNumber: resume.Hash,
				Holder:         holder.Holder,
				TotalARS:       holder.Totals.ARS,
				TotalUSD:       holder.Totals.USD,
				Expenses:       holdersExpenses,
			})

			holdersExpenses = nil // Reset for next holder

		}

		resumeDate, err := parseMonthYear(resume.FileName)
		if err != nil {
			continue // Skip this resume if date parsing fails
		}

		resumes = append(resumes, models.Resume{
			DocumentNumber: resume.Hash,
			Holders:        holders,
			CardType:       resume.CardLogo,
			ResumeDate:     resumeDate,
			TotalARS:       resume.Totals.ARS,
			TotalUSD:       resume.Totals.USD,
		})

		holders = nil // Reset for next resume

	}

	// db, err := ec.GetDatabaseInstance("TRANSACTION_DB")
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	// }

	// for _, resume := range resumes {

	// }

	c.JSON(http.StatusOK, gin.H{"Resumes": resumes})
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
			return nil, fmt.Errorf("error reading directory %s: %w", dir, err)
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

func getResumeData(paths []resumePaths) []ResumesData {

	var ResumeData []ResumesData
	var ResumeDetail []ResumeDetails

	bbvaReader, err := services.NewPdfReaderBBVA()
	if err != nil {
		//return fmt.Errorf("error trying to create a new google reader instance at SyncExpensesByMonth(): %w", err)
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

	return ResumeData

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
