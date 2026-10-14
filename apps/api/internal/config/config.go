package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL      string
	Addr             string
	OllamaBaseURL    string
	OllamaModel      string
	OllamaVisionModel string
	CORSOrigin       string
}

func Load() Config {
	_ = godotenv.Load()
	_ = godotenv.Load("../../.env")

	return Config{
		DatabaseURL:       getenv("DATABASE_URL", "postgres://kasboek:kasboek@localhost:5433/kasboek?sslmode=disable"),
		Addr:              getenv("API_ADDR", ":8080"),
		OllamaBaseURL:     strings.TrimRight(getenv("OLLAMA_BASE_URL", "http://127.0.0.1:11434"), "/"),
		OllamaModel:       getenv("OLLAMA_MODEL", "llama3.2"),
		OllamaVisionModel: getenv("OLLAMA_VISION_MODEL", "llava"),
		CORSOrigin:        getenv("CORS_ORIGIN", "http://localhost:5173"),
	}
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
