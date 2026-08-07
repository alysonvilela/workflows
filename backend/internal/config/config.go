// Package config loads the single root .env into typed configuration,
// failing fast when a required variable is missing.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	APIPort            string
	CORSAllowedOrigins string

	DatabaseURL string

	S3Endpoint     string
	S3Region       string
	S3Bucket       string
	S3AccessKey    string
	S3SecretKey    string
	S3UsePathStyle bool

	MaxUploadBytes           int64
	StorageDeleteAfterIngest bool

	LLMBaseURL        string
	LLMAPIKey         string
	LLMModel          string
	LLMSummaryModel   string
	LLMTimeoutSeconds int
	LLMMaxConcurrent  int
}

// Load reads the root .env (if present — a real deployment may set the
// environment directly instead) and builds a Config, returning an error
// naming the first required variable that is missing.
func Load() (Config, error) {
	loadRootEnv()

	var missing []string
	req := func(key string) string {
		v := os.Getenv(key)
		if v == "" {
			missing = append(missing, key)
		}
		return v
	}

	cfg := Config{
		APIPort:            req("API_PORT"),
		CORSAllowedOrigins: req("CORS_ALLOWED_ORIGINS"),

		DatabaseURL: req("DATABASE_URL"),

		S3Endpoint:  req("S3_ENDPOINT"),
		S3Region:    req("S3_REGION"),
		S3Bucket:    req("S3_BUCKET"),
		S3AccessKey: req("S3_ACCESS_KEY"),
		S3SecretKey: req("S3_SECRET_KEY"),

		LLMBaseURL:      req("LLM_BASE_URL"),
		LLMModel:        req("LLM_MODEL"),
		LLMSummaryModel: req("LLM_SUMMARY_MODEL"),
	}
	maxUploadBytesStr := req("MAX_UPLOAD_BYTES")
	llmTimeoutSecondsStr := req("LLM_TIMEOUT_SECONDS")
	llmMaxConcurrentStr := req("LLM_MAX_CONCURRENT")

	if len(missing) > 0 {
		return Config{}, fmt.Errorf("config: missing required environment variable(s): %v (copy .env.example to .env and fill them in)", missing)
	}

	// Not every OpenAI-compatible provider needs a key (local vLLM, Ollama,
	// LM Studio) — unlike the rest of this block, an empty value is valid.
	cfg.LLMAPIKey = os.Getenv("LLM_API_KEY")

	cfg.S3UsePathStyle = boolEnv("S3_USE_PATH_STYLE", true)
	cfg.StorageDeleteAfterIngest = boolEnv("STORAGE_DELETE_AFTER_INGEST", false)

	maxUploadBytes, err := strconv.ParseInt(maxUploadBytesStr, 10, 64)
	if err != nil {
		return Config{}, fmt.Errorf("config: MAX_UPLOAD_BYTES must be an integer: %w", err)
	}
	cfg.MaxUploadBytes = maxUploadBytes

	llmTimeoutSeconds, err := strconv.Atoi(llmTimeoutSecondsStr)
	if err != nil {
		return Config{}, fmt.Errorf("config: LLM_TIMEOUT_SECONDS must be an integer: %w", err)
	}
	cfg.LLMTimeoutSeconds = llmTimeoutSeconds

	llmMaxConcurrent, err := strconv.Atoi(llmMaxConcurrentStr)
	if err != nil {
		return Config{}, fmt.Errorf("config: LLM_MAX_CONCURRENT must be an integer: %w", err)
	}
	cfg.LLMMaxConcurrent = llmMaxConcurrent

	return cfg, nil
}

// loadRootEnv looks for .env starting at the working directory and walking
// up, so `go run ./cmd/api` from backend/ still finds the repo-root .env.
func loadRootEnv() {
	dir, err := os.Getwd()
	if err != nil {
		return
	}
	for {
		path := filepath.Join(dir, ".env")
		if _, err := os.Stat(path); err == nil {
			_ = godotenv.Load(path)
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}

func boolEnv(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
