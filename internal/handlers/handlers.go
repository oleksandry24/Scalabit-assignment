package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/oleksandry24/github-api-manager/internal/github"

	gh "github.com/google/go-github/v59/github"
)

type Handler struct {
	GithubService github.RepositoryService
}

type RepoResponse struct {
	OwnerLogin   string `json:"owner_login"`
	OwnerHTMLURL string `json:"owner_html_url"`
	Name         string `json:"name"`
	FullName     string `json:"full_name"`
	HTMLURL      string `json:"html_url"`
	CloneURL     string `json:"clone_url"`
	GitURL       string `json:"git_url"`
	SSHURL       string `json:"ssh_url"`
	Visibility   string `json:"visibility"`
}

func mapToRepoResponse(repo *gh.Repository) RepoResponse {
	return RepoResponse{
		OwnerLogin:   repo.GetOwner().GetLogin(),
		OwnerHTMLURL: repo.GetOwner().GetHTMLURL(),
		Name:         repo.GetName(),
		FullName:     repo.GetFullName(),
		HTMLURL:      repo.GetHTMLURL(),
		CloneURL:     repo.GetCloneURL(),
		GitURL:       repo.GetGitURL(),
		SSHURL:       repo.GetSSHURL(),
		Visibility:   repo.GetVisibility(),
	}
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
	_ = json.NewEncoder(w).Encode(mapToRepoResponse(repo))
}

// Delete Repo handler (DELETE /repos/{owner}/{repo})
func (h *Handler) DeleteRepo(w http.ResponseWriter, r *http.Request) {

	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	force := r.URL.Query().Get("force") == "true" // Force is used to delete a repo even if it has forks, open issues, or is larger than 1000kb, we block it unless "force=true" is in the URL.

	// Validate input
	if owner == "" || repo == "" {
		http.Error(w, "Owner and repository name are required", http.StatusBadRequest)
		return
	}

	repoInfo, err := h.GithubService.CheckRepo(r.Context(), owner, repo)
	if err != nil {
		http.Error(w, "Failed to check repository, Not Found!", http.StatusInternalServerError)
		return
	}

	if !force {
		if repoInfo.GetForksCount() > 0 || repoInfo.GetOpenIssuesCount() > 0 || repoInfo.GetSize() > 1000 {
			http.Error(w, "Repository Protection! This repository looks important. Pass ?force=true in the URL to delete it anyway.", http.StatusForbidden)
			return
		}
	}

	// request the delete to the github client
	err = h.GithubService.DeleteRepo(r.Context(), owner, repo)
	if err != nil {
		http.Error(w, "Failed to delete repository", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprintf("Repository %s/%s deleted successfully", owner, repo)})
}

// List Repos handler (GET /repos)
func (h *Handler) ListRepos(w http.ResponseWriter, r *http.Request) {

	repos, err := h.GithubService.ListRepos(r.Context())
	if err != nil {
		http.Error(w, "Failed to list repositories", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	var repoResponses []RepoResponse
	for _, repo := range repos {
		repoResponses = append(repoResponses, mapToRepoResponse(repo))
	}
	_ = json.NewEncoder(w).Encode(repoResponses)
	w.WriteHeader(http.StatusOK)
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
	_ = json.NewEncoder(w).Encode(prs)
}

// Change Repo Visibility handler (POST /repos/{owner}/{repo}/change-visibility)
func (h *Handler) ChangeRepoVisibility(w http.ResponseWriter, r *http.Request) {

	owner := r.PathValue("owner")
	repo := r.PathValue("repo")

	// Validate input
	if owner == "" || repo == "" {
		http.Error(w, "Owner and repository name are required", http.StatusBadRequest)
		return
	}

	type ChangeVisibilityRequest struct {
		Private bool `json:"private"`
	}

	var req ChangeVisibilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updatedRepo, err := h.GithubService.ChangeRepoVisibility(r.Context(), owner, repo, req.Private)
	if err != nil {
		http.Error(w, "Failed to change repository visibility", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(mapToRepoResponse(updatedRepo))
}
