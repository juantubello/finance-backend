package services

import (
	"bytes"
	"finance-backend/config"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
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

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Error reading the response: ", err)
		return nil, err
	}

	fmt.Println("Status:", res.Status)
	fmt.Println("Response:", string(resBody))

	return resumeData, nil
}
