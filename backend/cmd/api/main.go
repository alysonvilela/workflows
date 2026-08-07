package main

import (
	"context"
	"fmt"
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
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
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
	if err := db.MigrateRiver(ctx); err != nil {
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

	workers := river.NewWorkers()
	river.AddWorker(workers, &ingest.ProcessDocumentWorker{Ingestor: ingestor})

	// WORKER_ENABLED=false makes this an insert-only client: Stage still
	// enqueues a job per Document via db.River, but this process never
	// dequeues and works one — for splitting HTTP-only replicas from
	// dedicated worker replicas on the same queue.
	riverConfig := &river.Config{Workers: workers}
	if cfg.WorkerEnabled {
		riverConfig.Queues = map[string]river.QueueConfig{
			// RIVER_MAX_WORKERS only bounds how many jobs this client may
			// dequeue at once; LLM_MAX_CONCURRENT's semaphore in the
			// Provider is what actually caps outbound LLM calls, and can
			// sit well below it.
			river.QueueDefault: {MaxWorkers: cfg.RiverMaxWorkers},
		}
	}
	riverClient, err := river.NewClient(riverpgxv5.New(db.Pool), riverConfig)
	if err != nil {
		return fmt.Errorf("river client: %w", err)
	}
	db.River = riverClient

	if cfg.WorkerEnabled {
		if err := riverClient.Start(ctx); err != nil {
			return fmt.Errorf("river client start: %w", err)
		}
	}

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

	// riverClient.Stop drains any job it has already fetched — e.g. a
	// summarise call mid-flight — before returning, same "drain, don't
	// kill" contract as srv.Shutdown above. air's kill_delay must not
	// sever it mid-call.
	if cfg.WorkerEnabled {
		stopCtx, cancelStop := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancelStop()
		if err := riverClient.Stop(stopCtx); err != nil {
			slog.Warn("river client did not stop cleanly", "error", err)
		}
	}
	return nil
}
