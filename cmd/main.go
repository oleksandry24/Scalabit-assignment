package main

import (
	"log"
	"net/http"
	"os"

	"github.com/oleksandry24/github-api-manager/internal/github"
	"github.com/oleksandry24/github-api-manager/internal/handlers"
)

func main() {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("fATAL: GITHUB_TOKEN environment variable is required")
	}

	ghService := github.NewClient(token)
	if err := ghService.ValidateToken(); err != nil {
		log.Fatalf("Failed to validate GitHub token: %v", err)
	}

	hands := &handlers.Handlers{
		GithubService: ghService,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("POST /repos", hands.CreateRepo)
	mux.HandleFunc("DELETE /repos/{owner}/{repo}", hands.DeleteRepo)

	mux.HandleFunc("GET /repos", hands.ListRepos)

	mux.HandleFunc("POST /repos/{owner}/{repo}/prs", hands.ListOpenPrs)

	port := "8080"
	log.Printf("Server is running on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
