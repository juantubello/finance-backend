package transactions

import (
	"finance-backend/config"
	"fmt"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gorm.io/gorm"
)

/*
BaseController provides shared transaction functionality:
- DB connection management
- Consistent amount/date formatting
Embed this in specific controllers (expenses/incomes) to avoid duplication.
*/
type BaseController struct{}

/*
GetDatabaseInstance returns an instance of a database
- database string, parameter checks name on .env file
*/
func (b *BaseController) GetDatabaseInstance(database string) (*gorm.DB, error) {
	databaseName := config.GetEnv(database)
	db, ok := config.DBs[databaseName]
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
