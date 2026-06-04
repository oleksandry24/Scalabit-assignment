package handlers

import (
	"net/http"

	"github.com/oleksandry24/github-api-manager/internal/github"

	"encoding/json"
)

type Handler struct {
	GithubService github.RepositoryService
}

// Create Repo handler (POST /repos)
func (h *Handler) CreateRepo(w http.ResponseWriter, r *http.Request) {

	type CreateRepoRequest struct {
		Name string `json:"name"`
	}

	// Parse request body
	var req CreateRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Name == "" {
		http.Error(w, "Repository name is required", http.StatusBadRequest)
		return
	}

	// Here we call the github to create the repositroy
	repo, err := h.GithubService.CreateRepo(r.Context(), req.Name)
	if err != nil {
		http.Error(w, "Failed to create repository", http.StatusInternalServerError)
		return
	}

	// output the details of created repo
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(repo)
}

// Delete Repo handler (DELETE /repos/{owner}/{repo})
func (h *Handler) DeleteRepo(w http.ResponseWriter, r *http.Request) {

	owner := r.PathValue("owner")
	repo := r.PathValue("repo")

	// Validate input
	if owner == "" || repo == "" {
		http.Error(w, "Owner and repository name are required", http.StatusBadRequest)
		return
	}

	// request the delete to the github client
	err := h.GithubService.DeleteRepo(r.Context(), owner, repo)
	if err != nil {
		http.Error(w, "Failed to delete repository", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// List Repos handler (GET /repos)
func (h *Handler) ListRepos(w http.ResponseWriter, r *http.Request) {

	repos, err := h.GithubService.ListRepos(r.Context())
	if err != nil {
		http.Error(w, "Failed to list repositories", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(repos)
}

// List Open PRs handler (GET /repos/{owner}/{repo}/prs)
func (h *Handler) ListOpenPrs(w http.ResponseWriter, r *http.Request) {

	owner := r.PathValue("owner")
	repo := r.PathValue("repo")

	// Validate input
	if owner == "" || repo == "" {
		http.Error(w, "Owner and repository name are required", http.StatusBadRequest)
		return
	}

	prs, err := h.GithubService.ListOpenPRs(r.Context(), owner, repo)
	if err != nil {
		http.Error(w, "Failed to list open pull requests", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(prs)
}
