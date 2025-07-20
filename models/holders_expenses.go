package models

type HolderExpense struct {
	DocumentNumber  string  `gorm:"primaryKey" json:"document_number"`
	Holder          string  `gorm:"primaryKey" json:"holder"`
	Position        int     `gorm:"primaryKey" json:"position"`
	Date            string  `json:"date" gorm:"type:datetime"`
	Description     string  `json:"description"`
	Amount          float64 `json:"amount"`
	FormattedAmount string  `json:"formatted_amount"`
}
