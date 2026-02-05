package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/challenge-github-events/internal/config"
	"github.com/challenge-github-events/internal/github"
	"github.com/challenge-github-events/internal/pubsub"
	"github.com/challenge-github-events/internal/server"
	"github.com/challenge-github-events/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	slog.Info("starting", "poll_interval_sec", cfg.PollIntervalSec, "consumer_workers", cfg.ConsumerWorkers, "channel_size", cfg.ChannelSize, "http_addr", cfg.HTTPAddr)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("connect to database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("database connected")

	st := store.NewPostgres(pool)
	gh := github.NewClient(cfg.GHToken)

	// Bounded channel for backpressure
	jobs := make(chan pubsub.CommitJob, cfg.ChannelSize)
	defer close(jobs)

	// Consumer workers
	cons := pubsub.NewConsumer(st, gh, jobs)
	var wg sync.WaitGroup
	for i := 0; i < cfg.ConsumerWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cons.Run(ctx)
		}()
	}
	slog.Info("consumer workers started", "workers", cfg.ConsumerWorkers)

	// Producer
	pollInterval := time.Duration(cfg.PollIntervalSec) * time.Second
	prod := pubsub.NewProducer(st, gh, jobs, pollInterval)
	runCtx, cancel := context.WithCancel(ctx)
	go prod.Run(runCtx)
	slog.Info("producer started", "poll_interval", pollInterval)

	// HTTP server
	srv := server.NewServer(cfg.HTTPAddr, st)
	go func() {
		slog.Info("http server listening", "addr", cfg.HTTPAddr)
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server", "err", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	slog.Info("shutting down", "signal", "received")

	cancel()
	close(jobs)
	wg.Wait()
	slog.Info("consumer workers stopped")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Warn("http server shutdown", "err", err)
	} else {
		slog.Info("http server stopped")
	}
}
