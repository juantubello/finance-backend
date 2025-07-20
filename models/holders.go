package models

type Holder struct {
	DocumentNumber    string          `gorm:"primaryKey" json:"document_number"`
	Holder            string          `gorm:"primaryKey" json:"holder"`
	TotalARS          float64         `json:"total_ars"`
	TotalUSD          float64         `json:"total_usd"`
	FormattedTotalARS string          `json:"formatted_total_ars"`
	FormattedTotalUSD string          `json:"formatted_total_usd"`
	Expenses          []HolderExpense `gorm:"foreignKey:DocumentNumber,Holder;references:DocumentNumber,Holder"`
}
