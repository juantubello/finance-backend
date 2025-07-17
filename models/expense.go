package models

import "time"

type Expenses struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UUID        string    `gorm:"unique" json:"uuid"`
	DateTime    string    `json:"date_time"`                 // formato: "2025-07-08 12:00:00"
	Date        time.Time `json:"date" gorm:"type:datetime"` // Nuevo campo de tipo fecha
	Description string    `json:"description"`
	Amount      float64   `json:"amount"`
	Type        string    `json:"type"`
}
