package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/discovery/backend/internal/config"
	"github.com/discovery/backend/internal/httpapi"
	"github.com/discovery/backend/internal/ingest"
	"github.com/discovery/backend/internal/llm"
	"github.com/discovery/backend/internal/storage"
	"github.com/discovery/backend/internal/store"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.Migrate(cfg.DatabaseURL); err != nil {
		return err
	}
	slog.Info("migrations applied")

	objectStore, err := storage.New(ctx, storage.Config{
		Endpoint:     cfg.S3Endpoint,
		Region:       cfg.S3Region,
		Bucket:       cfg.S3Bucket,
		AccessKey:    cfg.S3AccessKey,
		SecretKey:    cfg.S3SecretKey,
		UsePathStyle: cfg.S3UsePathStyle,
	})
	if err != nil {
		return err
	}
	if err := objectStore.EnsureBucket(ctx); err != nil {
		return err
	}

	provider := llm.NewOpenAIProvider(llm.Config{
		BaseURL:       cfg.LLMBaseURL,
		APIKey:        cfg.LLMAPIKey,
		MaxConcurrent: cfg.LLMMaxConcurrent,
		Timeout:       time.Duration(cfg.LLMTimeoutSeconds) * time.Second,
	})

	ingestor := &ingest.Ingestor{
		Documents:         db,
		Objects:           objectStore,
		Summariser:        &llm.Summariser{Provider: provider, Model: cfg.LLMSummaryModel},
		MaxUploadBytes:    cfg.MaxUploadBytes,
		DeleteAfterIngest: cfg.StorageDeleteAfterIngest,
	}

	worker := &ingest.Worker{
		Ingestor:     ingestor,
		PollInterval: time.Second,
		// No point claiming more Documents than can have an LLM call running
		// at once — LLM_MAX_CONCURRENT is already the real bottleneck.
		BatchSize: cfg.LLMMaxConcurrent,
	}
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		worker.Run(ctx)
	}()

	router := httpapi.NewRouter(db, objectStore, ingestor, cfg.CORSAllowedOrigins)

	srv := &http.Server{
		Addr:              ":" + cfg.APIPort,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("listening", "port", cfg.APIPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		slog.Info("shutting down, draining in-flight requests")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}

	// worker.Run already stopped claiming new batches when ctx was cancelled
	// above; this just waits for whatever it had already claimed to finish
	// processing, same "drain, don't kill" contract as srv.Shutdown.
	select {
	case <-workerDone:
	case <-time.After(15 * time.Second):
		slog.Warn("ingest worker did not drain in time")
	}
	return nil
}
