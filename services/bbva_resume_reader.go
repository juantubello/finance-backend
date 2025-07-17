package services

import (
	"bytes"
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

type ReaderBBVA struct {
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
type Gasto struct {
	Fecha          string `json:"fecha"`
	FechaTimestamp string `json:"fechaTimestamp"`
	Descripcion    string `json:"descripcion"`
	Importe        string `json:"importe"`
	ImporteFloat   float64
}

type Totales struct {
	Pesos        string `json:"pesos"`
	Dolares      string `json:"dolares"`
	PesosFloat   float64
	DolaresFloat float64
}

type PersonaData struct {
	Detail []Gasto `json:"Detail"`
	Total  Totales `json:"Total"`
}

type GastoConTitular struct {
	Titular        string
	Fecha          string
	FechaTimestamp string
	Descripcion    string
	Importe        string
	ImporteFloat   float64
}

// NewGoogleSheetsReader crea una nueva instancia del lector de Sheets
func NewPdfReaderBBVA() (*ReaderBBVA, error) {
	return &ReaderBBVA{
		service: config.GetEnv("BBVA_PDF_SERVICE"),
	}, nil
}

func (bbvaReader *ReaderBBVA) ReadResumes(path ResumePath) ([]ResumeData, error) {

	var resumeData []ResumeData

	url := bbvaReader.service
	method := "POST"

	if path.FilePath == "" {
		return nil, fmt.Errorf("file path is empty")
	}

	filePath := path.FilePath

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error trying to open file:", err)
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Header con tipo MIME explícito
	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filepath.Base(filePath)))
	partHeader.Set("Content-Type", "application/pdf") // <--- in Go lang header content type must be explicit

	part, err := writer.CreatePart(partHeader)
	if err != nil {
		fmt.Println("Error trying to create form part: ", err)
		return nil, err
	}

	_, err = io.Copy(part, file)
	if err != nil {
		fmt.Println("Error copying file: ", err)
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		fmt.Println("Error while closing writer: ", err)
		return nil, err
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		fmt.Println("Error creating the request: ", err)
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		fmt.Println("Error while performing request:", err)
		return nil, err
	}
	defer res.Body.Close()

	responseJSON, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Error reading the response: ", err)
		return nil, err
	}

	fmt.Println("Status:", res.Status)
	//fmt.Println("Response:", string(resBody))

	gastos, totalesPorPersona, totalGlobal, err := ParsearRespuestaCompleta(responseJSON)
	if err != nil {
		log.Fatal(err)
	}

	for _, g := range gastos {
		fmt.Printf("[%s] %s - %s - %.2f\n", g.Titular, g.Fecha, g.Descripcion, g.ImporteFloat)
	}

	fmt.Println("\nTotales por persona:")
	for nombre, total := range totalesPorPersona {
		fmt.Printf("  %s: %.2f pesos / %.2f dólares\n", nombre, total.PesosFloat, total.DolaresFloat)
	}

	fmt.Println("\nTotal global:")
	fmt.Printf("  %.2f pesos / %.2f dólares\n", totalGlobal.PesosFloat, totalGlobal.DolaresFloat)

	return resumeData, nil
}

func ParsearRespuestaCompleta(jsonData []byte) ([]GastoConTitular, map[string]Totales, Totales, error) {
	var raw map[string]PersonaData
	err := json.Unmarshal(jsonData, &raw)
	if err != nil {
		return nil, nil, Totales{}, fmt.Errorf("error parseando JSON: %w", err)
	}

	var (
		gastos            []GastoConTitular
		totalesPorPersona = make(map[string]Totales)
		totalGlobal       Totales
	)

	for nombre, data := range raw {
		// Parsear totales
		pesosF, err1 := ParsearImporte(data.Total.Pesos)
		dolaresF, err2 := ParsearImporte(data.Total.Dolares)
		if err1 != nil || err2 != nil {
			log.Printf("Error parseando total de %s: %v %v", nombre, err1, err2)
		}
		data.Total.PesosFloat = pesosF
		data.Total.DolaresFloat = dolaresF

		if nombre == "Total" {
			totalGlobal = data.Total
			continue
		}

		totalesPorPersona[nombre] = data.Total

		// Parsear gastos
		for _, gasto := range data.Detail {
			valorFloat, err := ParsearImporte(gasto.Importe)
			if err != nil {
				log.Printf("Error parseando importe %q de %s: %v", gasto.Importe, nombre, err)
				continue
			}

			gastos = append(gastos, GastoConTitular{
				Titular:        nombre,
				Fecha:          gasto.Fecha,
				FechaTimestamp: gasto.FechaTimestamp,
				Descripcion:    gasto.Descripcion,
				Importe:        gasto.Importe,
				ImporteFloat:   valorFloat,
			})
		}
	}

	return gastos, totalesPorPersona, totalGlobal, nil
}

func ParsearImporte(importeStr string) (float64, error) {
	if strings.TrimSpace(importeStr) == "" {
		return 0.0, nil
	}
	sanitized := strings.ReplaceAll(importeStr, ".", "")
	sanitized = strings.ReplaceAll(sanitized, ",", ".")
	return strconv.ParseFloat(sanitized, 64)
}
