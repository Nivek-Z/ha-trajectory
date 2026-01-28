package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"ha-trajectory/internal/config"
	"ha-trajectory/internal/handlers"
	"ha-trajectory/internal/repository"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	db, err := repository.ConnectWithRetry(cfg.DBDSN, 10, 2*time.Second)
	if err != nil {
		log.Fatalf("db connect error: %v", err)
	}

	repo := repository.NewTrackRepository(db)
	trackHandler := handlers.NewTrackHandler(repo)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/track", trackHandler.HandleTrack)
	mux.HandleFunc("/api/path", trackHandler.HandlePath)

	srv := &http.Server{
	Addr:              ":" + getPort(),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}

	_ = srv.Shutdown(context.Background())
	_ = os.Stdout.Sync()
}

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		return "8080"
	}
	return port
}
