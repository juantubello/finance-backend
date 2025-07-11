package controllers

import (
	"finance-backend/config"
	"finance-backend/models"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type FormattedExpenses struct {
	models.Expenses
	FormattedAmount string `json:"formatted_amount"`
}

func GetExpenses(c *gin.Context) {
	year := c.Query("year")
	month := c.Query("month")

	var transactionsTable string = config.GetEnv("TRANSACTION_TABLE")
	var expenses []models.Expenses

	// Filter by month: "2025-07%"
	datePattern := fmt.Sprintf("%s-%s%%", year, month)

	// Retrieve the instance from the instanciated map, on map [key][value] GOlang returns ok if key was found
	db, ok := config.DBs[transactionsTable]

	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not available"})
		return
	}

	// Run the query
	result := db.Where("date_time LIKE ?", datePattern).Find(&expenses)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	// Create printer for Argentine peso amount
	argentinePrinter := message.NewPrinter(language.Spanish)

	// Create slice to hold formatted transactions
	formattedTransactions := make([]FormattedExpenses, len(expenses))

	// Format each transaction
	for i, transaction := range expenses {
		formattedTransactions[i] = FormattedExpenses{
			Expenses:        transaction,                                           // Copy all original fields
			FormattedAmount: argentinePrinter.Sprintf("$%.2f", transaction.Amount), // Formats as $1.234,56
		}
	}

	c.JSON(http.StatusOK, formattedTransactions)

}
