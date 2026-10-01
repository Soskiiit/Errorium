package app

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/soskiiit/errorium/services/gateway/internal/handler"
	"github.com/soskiiit/errorium/services/gateway/pkg/config"
)

type App struct {
	cfg    *config.Config
	server *http.Server
}

func New(cfg *config.Config) *App {
	mux := http.NewServeMux()

	healthHandler := handler.NewHealthHandler()
	healthHandler.RegisterRoutes(mux)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &App{
		cfg:    cfg,
		server: server,
	}
}

func (a *App) Run() error {
	log.Printf("Gateway starting on :%s", a.cfg.Port)
	if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	log.Println("Shutting down gateway...")
	return a.server.Shutdown(ctx)
}
