package services

import (
	"context"
	"fmt"

	"finance-backend/config"

	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type GoogleSheetsReader struct {
	service       *sheets.Service
	spreadsheetID string
}

// NewGoogleSheetsReader crea una nueva instancia del lector de Sheets
func NewGoogleSheetsReader(spreadsheetID string) (*GoogleSheetsReader, error) {
	ctx := context.Background()
	cfg := config.GetGoogleSheetsConfig()

	// Configurar JWT
	conf := &jwt.Config{
		Email:        cfg.ClientEmail,
		PrivateKey:   []byte(cfg.PrivateKey),
		PrivateKeyID: cfg.PrivateKeyID,
		TokenURL:     cfg.TokenURI,
		Scopes: []string{
			sheets.SpreadsheetsReadonlyScope,
		},
	}

	// Crear servicio
	srv, err := sheets.NewService(ctx,
		option.WithHTTPClient(conf.Client(ctx)),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create Sheets service: %v", err)
	}

	return &GoogleSheetsReader{
		service:       srv,
		spreadsheetID: spreadsheetID,
	}, nil
}

// ReadSheet lee los datos de una hoja específica
func (gsr *GoogleSheetsReader) ReadSheet(sheetName string, sheetRange string) ([][]interface{}, error) {

	resp, err := gsr.service.Spreadsheets.Values.Get(gsr.spreadsheetID, sheetRange).Do()

	if err != nil {
		return nil, fmt.Errorf("unable to retrieve data from sheet: %v", err)
	}

	return resp.Values, nil
}
