// Command checker starts the website availability service.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/host1812/golang-tuts/internal/checker"
)

func main() {
	// Reuse a client: its transport manages a pool of connections safely across
	// concurrent requests. Timeout includes redirects and reading the response.
	client := &http.Client{Timeout: 5 * time.Second}
	mux := http.NewServeMux()
	// Method-aware routes automatically reject other methods with HTTP 405.
	mux.HandleFunc("GET /check", checker.Handler(client, checker.Sites))
	server := &http.Server{
		Addr: ":8080", Handler: mux,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// SIGINT (Ctrl+C) and SIGTERM start a graceful shutdown, allowing active
	// handlers a short time to finish before the process exits.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}()

	log.Printf("listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
