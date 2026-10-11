package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bonbonn1912/bonbonn-game-tracker/internal/config"
	"github.com/bonbonn1912/bonbonn-game-tracker/internal/faceit"
	"github.com/bonbonn1912/bonbonn-game-tracker/internal/handlers"
	"github.com/bonbonn1912/bonbonn-game-tracker/internal/store"
)

func checkHealth(port string) error {
	if port == "" {
		port = "12000"
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%s/healthz", port))
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}
	return nil
}

func main() {
	// Subcommand: health
	// Used by compose/k8s health checks: ./cmd/server health
	if len(os.Args) > 1 && os.Args[1] == "health" {
		port := os.Getenv("PORT")
		if err := checkHealth(port); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		fmt.Println("healthy")
		os.Exit(0)
	}

	// Initialize JSON structured logging to stdout
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load configuration and require ADMIN_PASSWORD
	cfg, err := config.Load(true)
	if err != nil {
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}

	slog.Info("starting bonbonn game tracker backend",
		"port", cfg.Port,
		"env", cfg.Env,
		"adminUsername", cfg.AdminUsername,
		"adminHost", cfg.AdminHost,
		"faceitBaseURL", cfg.FaceitBaseURL,
	)

	// Connect to PostgreSQL
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pgStore, err := store.NewPostgresStore(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer closeCancel()
		_ = pgStore.Close(closeCtx)
	}()

	// Initialize Faceit Client
	faceitClient := faceit.NewClient(cfg.FaceitBaseURL, cfg.FaceitAPIKey, nil)

	// Initialize HTTP Server
	server, err := handlers.NewServer(cfg, pgStore, faceitClient)
	if err != nil {
		slog.Error("failed to initialize http server", "error", err)
		os.Exit(1)
	}

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      server,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown handling
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("http server listening", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server failed", "error", err)
			os.Exit(1)
		}
	}()

	sig := <-stopChan
	slog.Info("shutting down server...", "signal", sig.String())

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("server exited cleanly")
}
