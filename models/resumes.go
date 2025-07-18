package models

type Resume struct {
	DocumentNumber string  `gorm:"primaryKey" json:"document_number"`
	CardType       string  `json:"card_type"`
	ResumeDate     string  `json:"resume_date"` // formato: "2025-07-08T00:00:00"
	TotalARS       float64 `json:"total_ars"`
	TotalUSD       float64 `json:"total_usd"`

	Holders []Holder `gorm:"foreignKey:DocumentNumber;references:DocumentNumber"`
}
