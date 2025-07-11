package models

type Expenses struct {
	ID          uint    `gorm:"primaryKey" json:"id"`
	UUID        string  `gorm:"unique" json:"uuid"`
	DateTime    string  `json:"date_time"` // formato: "2025-07-08 12:00:00"
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Type        string  `json:"type"`
}
