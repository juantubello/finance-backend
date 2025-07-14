package transactions

import (
	"finance-backend/config"
	"fmt"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gorm.io/gorm"
)

type BaseController struct{}

func (b *BaseController) GetTransactionsDB() (*gorm.DB, error) {
	transactionsTable := config.GetEnv("TRANSACTION_TABLE")
	db, ok := config.DBs[transactionsTable]
	if !ok {
		return nil, fmt.Errorf("database not available")
	}
	return db, nil
}

func (b *BaseController) FormatAmount(amount float64) string {
	p := message.NewPrinter(language.Spanish)
	return p.Sprintf("$%.2f", amount)
}

func (b *BaseController) FormatDate(dateStr string) string {
	t, err := time.Parse("2006-01-02T15:04:05", dateStr)
	if err != nil {
		return dateStr
	}
	return t.Format("2/1/2006 15:04:05")
}
