package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/StefanPerez-Dev/BugdetTool/apps/api/internal/api"
	"github.com/StefanPerez-Dev/BugdetTool/apps/api/internal/config"
	"github.com/StefanPerez-Dev/BugdetTool/apps/api/internal/ollama"
	"github.com/StefanPerez-Dev/BugdetTool/apps/api/internal/store"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	st, err := store.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer st.Close()

	if err := st.SeedIfEmpty(ctx); err != nil {
		log.Fatalf("seed: %v", err)
	}

	ol := ollama.New(cfg.OllamaBaseURL, cfg.OllamaModel, cfg.OllamaVisionModel)
	srv := api.New(cfg, st, ol)

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("kasboek api listening on %s", cfg.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
}
