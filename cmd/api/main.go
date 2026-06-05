package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/oleksandry24/github-api-manager/internal/github"
	"github.com/oleksandry24/github-api-manager/internal/handlers"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, relying on OS environment variables")
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("fATAL: GITHUB_TOKEN environment variable is required")
	}

	ghService := github.NewClient(token)

	hands := &handlers.Handler{
		GithubService: ghService,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("POST /repos", hands.CreateRepo)
	mux.HandleFunc("DELETE /repos/{owner}/{repo}", hands.DeleteRepo)

	mux.HandleFunc("GET /repos", hands.ListRepos)

	mux.HandleFunc("GET /repos/{owner}/{repo}/prs", hands.ListOpenPrs)

	mux.HandleFunc("GET /health", handlers.HealthCheck)
	// mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
	// 	w.WriteHeader(http.StatusOK)
	// 	w.Header().Set("Content-Type", "application/json")
	// 	_, _ = w.Write([]byte(`{"status": "ok"}`))
	// })

	mux.HandleFunc("PUT /repos/{owner}/{repo}/change-visibility", hands.ChangeRepoVisibility)

	port := "8080"

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,  // Max time to read the request
		WriteTimeout: 10 * time.Second, // Max time to write the response
		IdleTimeout:  15 * time.Second, // Max time to keep connection open
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("Starting server on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}

	}()

	<-ctx.Done()
	log.Print("Shutting down server...")

	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdown); err != nil {
		log.Fatalf("Failed to shutdown server: %v", err)
	}

	log.Print("Server gracefully stopped")
}
