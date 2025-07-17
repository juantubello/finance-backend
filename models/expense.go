package models

import "time"

type Expenses struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UUID        string    `gorm:"unique" json:"uuid"`
	DateTime    string    `json:"date_time"`
	Date        time.Time `json:"date" gorm:"type:datetime"`
	Description string    `json:"description"`
	Amount      float64   `json:"amount"`
	Type        string    `json:"type"`
}
