package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/soskiiit/errorium/services/gateway/internal/app"
	"github.com/soskiiit/errorium/services/gateway/internal/migrator"
	"github.com/soskiiit/errorium/services/gateway/pkg/config"
)

func main() {
	cfg := config.Load()

	if cfg.AutoMigrate && cfg.DB.Host != "" {
		if err := migrator.Up(cfg.DB.DSN()); err != nil {
			log.Fatalf("Database migration failed: %v", err)
		}
	}

	application := app.New(cfg)

	go func() {
		if err := application.Run(); err != nil {
			log.Fatalf("Gateway server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := application.Shutdown(ctx); err != nil {
		log.Printf("Gateway shutdown error: %v", err)
	}

	log.Println("Gateway stopped gracefully")
}
