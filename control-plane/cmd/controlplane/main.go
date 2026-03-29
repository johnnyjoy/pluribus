// Package main is the control-plane API server (memory, recall, ingest, enforcement, evidence).
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"control-plane/internal/app"
	"control-plane/internal/apiserver"
)

// Set at link time: go build -ldflags="-X main.version=1.2.3"; Docker passes --build-arg VERSION.
var version = "dev"

func main() {
	configPath := os.Getenv("CONFIG")
	if configPath == "" {
		configPath = "configs/config.example.yaml"
	}
	cfg, err := app.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	container, err := app.Boot(cfg)
	if err != nil {
		log.Fatalf("boot: %v", err)
	}
	if len(container.APIKey) > 0 {
		log.Printf("AUTH: enabled (API key required)")
	} else {
		log.Printf("AUTH: disabled (no PLURIBUS_API_KEY configured)")
		log.Printf("WARNING: all endpoints are publicly accessible")
	}
	defer container.DB.Close()

	router, err := apiserver.NewRouter(cfg, container)
	if err != nil {
		log.Fatalf("router: %v", err)
	}
	server := &http.Server{Addr: cfg.Server.Bind, Handler: router}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("server: %v", err)
		}
	}()
	log.Printf("controlplane %s listening on %s", version, cfg.Server.Bind)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	<-ctx.Done()
	stop()
	if err := server.Shutdown(context.Background()); err != nil {
		log.Printf("shutdown: %v", err)
	}
	fmt.Println("controlplane stopped")
}
