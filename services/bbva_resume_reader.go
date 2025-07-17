package services

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"finance-backend/config"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ---------- JSON structs in Spanish for unmarshaling ----------

type Gasto struct {
	Fecha          string `json:"fecha"`
	FechaTimestamp string `json:"fechaTimestamp"`
	Descripcion    string `json:"descripcion"`
	Importe        string `json:"importe"`
}

type Totales struct {
	Pesos   string `json:"pesos"`
	Dolares string `json:"dolares"`
}

type PersonaData struct {
	Detail []Gasto `json:"Detail"`
	Total  Totales `json:"Total"`
}

// ---------- Internal English structs ----------

type Expense struct {
	Date        string
	Description string
	Amount      float64
}

type Totals struct {
	ARS float64
	USD float64
}

type Holders struct {
	Holder   string
	Expenses []Expense
	Totals   Totals
}

// ---------- Main Reader ----------

type PdfReaderBBVA struct {
	service string
}

type ResumePath struct {
	CardLogo string
	FilePath string
	FileName string
}

type ResumeData struct {
	CardLogo string
	FilePath string
	FileName string
}

func NewPdfReaderBBVA() (*PdfReaderBBVA, error) {
	return &PdfReaderBBVA{
		service: config.GetEnv("BBVA_PDF_SERVICE"),
	}, nil
}

func (reader *PdfReaderBBVA) ReadResumes(path ResumePath) ([]Holders, Totals, string, error) {

	if path.FilePath == "" {
		return nil, Totals{}, "", fmt.Errorf("file path is empty")
	}

	file, err := os.Open(path.FilePath)
	if err != nil {
		return nil, Totals{}, "", fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filepath.Base(path.FilePath)))
	header.Set("Content-Type", "application/pdf")

	part, err := writer.CreatePart(header)
	if err != nil {
		return nil, Totals{}, "", fmt.Errorf("error creating form part: %w", err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return nil, Totals{}, "", fmt.Errorf("error copying file: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, Totals{}, "", fmt.Errorf("error closing writer: %w", err)
	}

	req, err := http.NewRequest("POST", reader.service, body)
	if err != nil {
		return nil, Totals{}, "", fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, Totals{}, "", fmt.Errorf("error making request: %w", err)
	}
	defer res.Body.Close()

	responseJSON, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, Totals{}, "", fmt.Errorf("error reading response: %w", err)
	}

	hash, err := hashString(responseJSON)
	if err != nil {
		return nil, Totals{}, "", fmt.Errorf("error hashing response: %w", err)
	}

	holders, globalTotals, err := ParseCompleteResponse(responseJSON)
	if err != nil {
		return nil, Totals{}, "", err
	}

	for _, holder := range holders {
		fmt.Printf("[%s]\n", holder.Holder)
		for _, e := range holder.Expenses {
			fmt.Printf("  %s - %s - %.2f\n", e.Date, e.Description, e.Amount)
		}
		fmt.Printf("Total: %.2f pesos / %.2f dollars\n\n", holder.Totals.ARS, holder.Totals.USD)
	}

	fmt.Println("Global Totals:")
	fmt.Printf("  %.2f pesos / %.2f dollars\n", globalTotals.ARS, globalTotals.USD)

	return holders, globalTotals, hash, nil
}

// ---------- Helpers ----------

func ParseCompleteResponse(jsonData []byte) ([]Holders, Totals, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(jsonData, &raw); err != nil {
		return nil, Totals{}, fmt.Errorf("error parsing JSON: %w", err)
	}

	var (
		holders      []Holders
		globalTotals Totals
	)

	for name, rawData := range raw {
		var maybeTotal Totales
		if err := json.Unmarshal(rawData, &maybeTotal); err == nil && maybeTotal.Pesos != "" && maybeTotal.Dolares != "" {
			if name == "Total" {
				pesosF, _ := parseAmount(maybeTotal.Pesos)
				dollarsF, _ := parseAmount(maybeTotal.Dolares)
				globalTotals = Totals{ARS: pesosF, USD: dollarsF}
				continue
			}
		}
		var persona PersonaData
		if err := json.Unmarshal(rawData, &persona); err != nil {
			log.Printf("Error parsing person %s: %v", name, err)
			continue
		}

		pesosF, _ := parseAmount(persona.Total.Pesos)
		dollarsF, _ := parseAmount(persona.Total.Dolares)

		var expenses []Expense
		for _, g := range persona.Detail {
			amountF, err := parseAmount(g.Importe)
			if err != nil {
				log.Printf("error parsing amount %q for %s: %v", g.Importe, name, err)
				continue
			}
			expenses = append(expenses, Expense{
				Date:        g.Fecha,
				Description: g.Descripcion,
				Amount:      amountF,
			})
		}

		holders = append(holders, Holders{
			Holder:   name,
			Expenses: expenses,
			Totals: Totals{
				ARS: pesosF,
				USD: dollarsF,
			},
		})
	}

	return holders, globalTotals, nil
}

func parseAmount(input string) (float64, error) {
	if strings.TrimSpace(input) == "" {
		return 0.0, nil
	}
	cleaned := strings.ReplaceAll(input, ".", "")
	cleaned = strings.ReplaceAll(cleaned, ",", ".")
	return strconv.ParseFloat(cleaned, 64)
}

func hashString(data []byte) (string, error) {
	hasher := sha256.New()
	_, err := hasher.Write(data)
	if err != nil {
		return "", fmt.Errorf("error writing to hasher: %v", err)
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
