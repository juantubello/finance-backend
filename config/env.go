package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env")
	}
}

func GetEnv(key string) string {
	return os.Getenv(key)
}

// Get config from google sheet service
func GetGoogleSheetsConfig() *GoogleSheetsConfig {
	return &GoogleSheetsConfig{
		Type:                GetEnv("GS_TYPE"),
		ProjectID:           GetEnv("GS_PROJECT_ID"),
		PrivateKeyID:        GetEnv("GS_PRIVATE_KEY_ID"),
		PrivateKey:          strings.ReplaceAll(GetEnv("GS_PRIVATE_KEY"), `\n`, "\n"),
		ClientEmail:         GetEnv("GS_CLIENT_EMAIL"),
		ClientID:            GetEnv("GS_CLIENT_ID"),
		AuthURI:             GetEnv("GS_AUTH_URI"),
		TokenURI:            GetEnv("GS_TOKEN_URI"),
		AuthProviderCertURL: GetEnv("GS_AUTH_PROVIDER_CERT_URL"),
		ClientCertURL:       GetEnv("GS_CLIENT_CERT_URL"),
		UniverseDomain:      GetEnv("GS_UNIVERSE_DOMAIN"),
	}
}

type GoogleSheetsConfig struct {
	Type                string `json:"type"`
	ProjectID           string `json:"project_id"`
	PrivateKeyID        string `json:"private_key_id"`
	PrivateKey          string `json:"private_key"`
	ClientEmail         string `json:"client_email"`
	ClientID            string `json:"client_id"`
	AuthURI             string `json:"auth_uri"`
	TokenURI            string `json:"token_uri"`
	AuthProviderCertURL string `json:"auth_provider_x509_cert_url"`
	ClientCertURL       string `json:"client_x509_cert_url"`
	UniverseDomain      string `json:"universe_domain"`
}
