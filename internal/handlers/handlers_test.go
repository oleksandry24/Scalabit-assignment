package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-github/v59/github"
)

// MOCK CLIENT
type MockGithubService struct{}

func (m *MockGithubService) CreateRepo(ctx context.Context, name string) (*github.Repository, error) {
	return &github.Repository{
		Name:       github.String(name),
		Visibility: github.String("private"),
		Owner: &github.User{
			Login: github.String("testuser"),
		},
	}, nil
}

func (m *MockGithubService) DeleteRepo(ctx context.Context, owner, repo string) error {
	return nil
}

func (m *MockGithubService) ListRepos(ctx context.Context) ([]*github.Repository, error) {
	return []*github.Repository{
		{
			Name:       github.String("frontend-app"),
			Visibility: github.String("public"),
			Owner:      &github.User{Login: github.String("testuser")},
		},
		{
			Name:       github.String("backend-api"),
			Visibility: github.String("private"),
			Owner:      &github.User{Login: github.String("testuser")},
		},
	}, nil
}

func (m *MockGithubService) ListOpenPRs(ctx context.Context, owner, repo string) ([]*github.PullRequest, error) {
	return []*github.PullRequest{}, nil
}

func (m *MockGithubService) CheckRepo(ctx context.Context, owner, repo string) (*github.Repository, error) {
	if repo == "linux" {
		repoInfo := &github.Repository{
			Name:       github.String("linux"),
			ForksCount: github.Int(500), // Trigger the guardrail!
			OpenIssues: github.Int(100),
			Size:       github.Int(1500),
		}
		return repoInfo, nil
	}
	return &github.Repository{
		Name:       github.String(repo),
		ForksCount: github.Int(0),
		OpenIssues: github.Int(0),
		Size:       github.Int(100),
	}, nil
}

func (m *MockGithubService) ChangeRepoVisibility(ctx context.Context, owner, repo string, private bool) (*github.Repository, error) {
	vis := "public"
	if private == true {
		vis = "private"
	}

	return &github.Repository{
		Name:       github.String(repo),
		Visibility: github.String(vis),
	}, nil
}

// Tests

func TestCreateRepo(t *testing.T) {
	hands := &Handler{GithubService: &MockGithubService{}}

	body := strings.NewReader(`{"name": "test-repo"}`)
	req, _ := http.NewRequest("POST", "/repos", body)
	rr := httptest.NewRecorder()

	hands.CreateRepo(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, rr.Code)
	}

	var response RepoResponse
	json.NewDecoder(rr.Body).Decode(&response)

	if response.Name != "test-repo" {
		t.Errorf("Expected repository name 'test-repo', got '%s'", response.Name)
	}

	if response.OwnerLogin != "testuser" {
		t.Errorf("Expected owner login 'testuser', got '%s'", response.OwnerLogin)
	}
}

func TestDeleteRepoWithProtection(t *testing.T) {

	hands := &Handler{GithubService: &MockGithubService{}}

	req, _ := http.NewRequest("DELETE", "/repos/testuser/linux", nil)
	rr := httptest.NewRecorder()

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /repos/{owner}/{repo}", hands.DeleteRepo)
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status code %d, got %d", http.StatusForbidden, rr.Code)
	}
}

func TestDeleteRepoWithoutProtection(t *testing.T) {
	hands := &Handler{GithubService: &MockGithubService{}}

	req, _ := http.NewRequest("DELETE", "/repos/testuser/linux?force=true", nil)
	rr := httptest.NewRecorder()

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /repos/{owner}/{repo}", hands.DeleteRepo)
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	var response map[string]string
	json.NewDecoder(rr.Body).Decode(&response)

	if response["message"] != "Repository testuser/linux deleted successfully" {
		t.Errorf("Expected message 'Repository testuser/linux deleted successfully', got '%s'", response["message"])
	}
}

func TestListRepos(t *testing.T) {
	hands := &Handler{GithubService: &MockGithubService{}}

	req := httptest.NewRequest("GET", "/repos", nil)
	rr := httptest.NewRecorder()

	hands.ListRepos(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	var response []RepoResponse
	err := json.NewDecoder(rr.Body).Decode(&response)

	if err != nil {
		t.Errorf("Failed to decode response body: %v", err)
	}

	// mock data mapped correctly?
	if len(response) != 2 {
		t.Errorf("Expected 2 repositories, got %d", len(response))
	}

	if response[0].Name != "frontend-app" {
		t.Errorf("Expected status code 'test-repo', got %d", response[0].Name)
	}

	if response[1].Name != "backend-api" || response[1].Visibility != "private" {
		t.Errorf("Expected second repo to be 'backend-api', got %s", response[1].Name)
	}

}

func TestChangeRepoVisibility(t *testing.T) {
	hands := &Handler{GithubService: &MockGithubService{}}

	body := strings.NewReader(`{"private": true}`)
	req, err := http.NewRequest("POST", "/repos/testuser/test-repo/change-visibility", body)
	rr := httptest.NewRecorder()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /repos/{owner}/{repo}/change-visibility", hands.ChangeRepoVisibility)
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	var response RepoResponse
	json.NewDecoder(rr.Body).Decode(&response)

	if response.Visibility != "private" {
		t.Errorf("Expected visibility 'private', got '%s'", response.Visibility)
	}
}
