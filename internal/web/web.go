// web.go defines the HTTP server structure and core routing logic.
package web

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	totConfig "tot-tally/internal/config"
	totCore "tot-tally/internal/core"
	totShards "tot-tally/internal/shards"
	totStats "tot-tally/internal/stats"
	totStorage "tot-tally/internal/storage"
)

// Start initializes all application layers and runs the web server.
func Start() {
	// 1. Setup Configuration.
	cfg := totConfig.NewDefaultConfig()

	// 2. Initialize Infrastructure.
	if err := os.MkdirAll(cfg.TotDirectory, 0755); err != nil {
		slog.Error("failed to create tot directory", "err", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(cfg.LimitDirectory, 0755); err != nil {
		slog.Error("failed to create limit directory", "err", err)
		os.Exit(1)
	}

	// 3. Instantiate Dependency Graph.
	pool := totShards.NewPool(cfg.NumShards)
	repo := totStorage.NewRepository(cfg, pool)
	engine := totStats.NewEngine(cfg)
	service := totCore.NewService(cfg, repo, engine)
	cleaner := NewCleaner(cfg, repo)
	router := NewServer(cfg, service, repo, engine, pool)

	// 4. Define Lifecycle Context.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 5. Start Background Workers.
	cleaner.StartBackgroundCleaner(ctx)

	// 6. Setup Routing.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", handlerWrapper(router.homeHandler))
	mux.HandleFunc("GET /manifest.json", handlerWrapper(router.manifestHandler))
	mux.HandleFunc("GET /{id}", handlerWrapper(router.getTotHandler))
	mux.HandleFunc("POST /", handlerWrapper(router.createTotHandler))
	mux.HandleFunc("POST /{id}", handlerWrapper(router.updateTotHandler))
	mux.HandleFunc("GET /export/{id}", handlerWrapper(router.exportTotHandler))

	fileServer := http.FileServer(http.Dir("assets/static/"))
	mux.Handle("GET /favicon.ico", http.StripPrefix("", fileServer))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fileServer))

	// 7. Start HTTP Server.
	server := &http.Server{
		Addr:         cfg.Port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		slog.Info("server starting", "addr", "http://localhost"+cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen error: %v", err)
		}
	}()

	// 8. Graceful Shutdown.
	<-ctx.Done()
	slog.Info("shutting down server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "err", err)
		os.Exit(1)
	}
	slog.Info("server exited cleanly")
}
