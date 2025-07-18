package models

import "time"

type Resume struct {
	DocumentNumber string    `gorm:"primaryKey" json:"document_number"`
	CardType       string    `json:"card_type"`
	ResumeDate     time.Time `json:"date" gorm:"type:resume_date"`
	TotalARS       float64   `json:"total_ars"`
	TotalUSD       float64   `json:"total_usd"`

	Holders []Holder `gorm:"foreignKey:DocumentNumber;references:DocumentNumber"`
}
