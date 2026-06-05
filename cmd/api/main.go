package main

import (
	"log"
	"net/http"
	"os"

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

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mux.HandleFunc("PUT /repos/{owner}/{repo}/change-visibility", hands.ChangeRepoVisibility)

	port := "8080"
	log.Printf("Server is running on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
