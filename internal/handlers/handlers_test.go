package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-github/v88/github"
)

// MOCK CLIENT
type MockGithubService struct {
	ShouldFail       bool
	ShouldProtect    bool
	ShouldFailDelete bool
}

func (m *MockGithubService) CreateRepo(ctx context.Context, name string) (*github.Repository, error) {
	if m.ShouldFail {
		return nil, errors.New("simulated error")
	}

	return &github.Repository{
		Name:       github.String(name),
		Visibility: github.String("private"),
		Owner: &github.User{
			Login: github.String("testuser"),
		},
	}, nil
}

func (m *MockGithubService) DeleteRepo(ctx context.Context, owner, repo string) error {
	if m.ShouldFail || m.ShouldFailDelete {
		return errors.New("fail")
	}
	return nil
}

func (m *MockGithubService) ListRepos(ctx context.Context) ([]*github.Repository, error) {
	if m.ShouldFail {
		return nil, errors.New("fail")
	}
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
	if m.ShouldFail {
		return nil, errors.New("fail")
	}
	prs := []*github.PullRequest{
		{
			Title: github.String("commit 1"),
			User:  &github.User{Login: github.String("testuser")},
			State: github.String("open"),
		},
		{
			Title: github.String("commit 2"),
			User:  &github.User{Login: github.String("testuser-2")},
			State: github.String("open"),
		},
	}
	return prs, nil
}

func (m *MockGithubService) CheckRepo(ctx context.Context, owner, repo string) (*github.Repository, error) {
	if m.ShouldFail {
		return nil, errors.New("fail")
	}
	if m.ShouldProtect {
		// Return a repo with forks to trigger the 403 Forbidden block
		return &github.Repository{ForksCount: github.Int(5)}, nil
	}

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
	if m.ShouldFail {
		return nil, errors.New("fail")
	}
	vis := "public"
	if private == true {
		vis = "private"
	}

	return &github.Repository{
		Name:       github.String(repo),
		Visibility: github.String(vis),
	}, nil
}

// get 100% coverage forcing the json

// Test 1: The Error Path
func TestWriteJSON_Error(t *testing.T) {
	w := httptest.NewRecorder()
	poisonedData := map[string]interface{}{
		"trap": make(chan int),
	}

	WriteJson(w, http.StatusOK, poisonedData)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 status on encoding error, got %d", w.Code)
	}
}

// Test 2: The Success Path
func TestWriteJSON_Success(t *testing.T) {
	w := httptest.NewRecorder()
	validData := map[string]string{
		"message": "everything worked",
	}

	WriteJson(w, http.StatusOK, validData)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", w.Code)
	}
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}

func TestDeleteRepo_FailsAtDeleteCall(t *testing.T) {
	// CheckRepo will pass, but DeleteRepo will fail!
	mock := &MockGithubService{ShouldFailDelete: true}
	hands := &Handler{GithubService: mock}

	req, _ := http.NewRequest("DELETE", "/repos/o/r?force=true", nil)
	req.SetPathValue("owner", "o")
	req.SetPathValue("repo", "r")

	rr := httptest.NewRecorder()
	hands.DeleteRepo(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500, got %d", rr.Code)
	}
}

// get 100% coverage forcing http error

func TestHandler_AllErrorPaths(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		url            string
		body           string
		setupMock      func() *MockGithubService
		handler        http.HandlerFunc
		setPathValues  map[string]string
		expectedStatus int
	}{
		// --- CreateRepo Errors ---
		{
			name:           "CreateRepo_BadJSON",
			method:         "POST",
			url:            "/repos",
			body:           `{"name": broken`,
			setupMock:      func() *MockGithubService { return &MockGithubService{} },
			handler:        (&Handler{}).CreateRepo,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "CreateRepo_EmptyName",
			method:         "POST",
			url:            "/repos",
			body:           `{"name": ""}`,
			setupMock:      func() *MockGithubService { return &MockGithubService{} },
			handler:        (&Handler{}).CreateRepo,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "CreateRepo_APIFail",
			method:         "POST",
			url:            "/repos",
			body:           `{"name": "test-repo"}`,
			setupMock:      func() *MockGithubService { return &MockGithubService{ShouldFail: true} },
			handler:        (&Handler{}).CreateRepo,
			expectedStatus: http.StatusInternalServerError,
		},

		// --- DeleteRepo Errors ---
		{
			name:           "DeleteRepo_MissingPath",
			method:         "DELETE",
			url:            "/repos",
			body:           "",
			setupMock:      func() *MockGithubService { return &MockGithubService{} },
			handler:        (&Handler{}).DeleteRepo,
			setPathValues:  map[string]string{"owner": "", "repo": ""}, // Explicitly empty
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "DeleteRepo_CheckFail",
			method:         "DELETE",
			url:            "/repos",
			body:           "",
			setupMock:      func() *MockGithubService { return &MockGithubService{ShouldFail: true} },
			handler:        (&Handler{}).DeleteRepo,
			setPathValues:  map[string]string{"owner": "o", "repo": "r"},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "DeleteRepo_Protected",
			method:         "DELETE",
			url:            "/repos", // No ?force=true
			body:           "",
			setupMock:      func() *MockGithubService { return &MockGithubService{ShouldProtect: true} },
			handler:        (&Handler{}).DeleteRepo,
			setPathValues:  map[string]string{"owner": "o", "repo": "r"},
			expectedStatus: http.StatusForbidden,
		},

		// --- ListRepos Errors ---
		{
			name:           "ListRepos_APIFail",
			method:         "GET",
			url:            "/repos",
			body:           "",
			setupMock:      func() *MockGithubService { return &MockGithubService{ShouldFail: true} },
			handler:        (&Handler{}).ListRepos,
			expectedStatus: http.StatusInternalServerError,
		},

		// --- ListOpenPRs Errors ---
		{
			name:           "ListPRs_MissingPath",
			method:         "GET",
			url:            "/repos",
			body:           "",
			setupMock:      func() *MockGithubService { return &MockGithubService{} },
			handler:        (&Handler{}).ListOpenPrs,
			setPathValues:  map[string]string{"owner": "", "repo": ""},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "ListPRs_APIFail",
			method:         "GET",
			url:            "/repos",
			body:           "",
			setupMock:      func() *MockGithubService { return &MockGithubService{ShouldFail: true} },
			handler:        (&Handler{}).ListOpenPrs,
			setPathValues:  map[string]string{"owner": "o", "repo": "r"},
			expectedStatus: http.StatusInternalServerError,
		},

		// --- ChangeRepoVisibility Errors ---
		{
			name:           "ChangeVis_MissingPath",
			method:         "POST",
			url:            "/repos",
			body:           "",
			setupMock:      func() *MockGithubService { return &MockGithubService{} },
			handler:        (&Handler{}).ChangeRepoVisibility,
			setPathValues:  map[string]string{"owner": "", "repo": ""},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "ChangeVis_BadJSON",
			method:         "POST",
			url:            "/repos",
			body:           `{"private": broken}`,
			setupMock:      func() *MockGithubService { return &MockGithubService{} },
			handler:        (&Handler{}).ChangeRepoVisibility,
			setPathValues:  map[string]string{"owner": "o", "repo": "r"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "ChangeVis_APIFail",
			method:         "POST",
			url:            "/repos",
			body:           `{"private": true}`,
			setupMock:      func() *MockGithubService { return &MockGithubService{ShouldFail: true} },
			handler:        (&Handler{}).ChangeRepoVisibility,
			setPathValues:  map[string]string{"owner": "o", "repo": "r"},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize a fresh handler with the specific mock state for this test
			hands := &Handler{GithubService: tt.setupMock()}

			req, _ := http.NewRequest(tt.method, tt.url, strings.NewReader(tt.body))
			rr := httptest.NewRecorder()

			// Apply Go 1.22 path values manually (bypassing the router entirely for direct testing)
			for key, val := range tt.setPathValues {
				req.SetPathValue(key, val)
			}

			// Call the handler directly!
			// Notice we mapped `(&Handler{}).FunctionName` in the table above,
			// but we execute it here using our freshly initialized `hands` variable.
			switch tt.name[:8] {
			case "CreateRe":
				hands.CreateRepo(rr, req)
			case "DeleteRe":
				hands.DeleteRepo(rr, req)
			case "ListRepo":
				hands.ListRepos(rr, req)
			case "ListPRs_":
				hands.ListOpenPrs(rr, req)
			case "ChangeVi":
				hands.ChangeRepoVisibility(rr, req)
			}

			if rr.Code != tt.expectedStatus {
				t.Errorf("%s: Expected status %d, got %d", tt.name, tt.expectedStatus, rr.Code)
			}
		})
	}
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
	_ = json.NewDecoder(rr.Body).Decode(&response)

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
	_ = json.NewDecoder(rr.Body).Decode(&response)

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
		t.Errorf("Expected status code 'test-repo', got %s", response[0].Name)
	}

	if response[1].Name != "backend-api" || response[1].Visibility != "private" {
		t.Errorf("Expected second repo to be 'backend-api', got %s", response[1].Name)
	}

}

func TestListOpenPRs(t *testing.T) {
	hands := &Handler{GithubService: &MockGithubService{}}

	req, _ := http.NewRequest("GET", "/repos/testuser/test-repo/prs", nil)
	rr := httptest.NewRecorder()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /repos/{owner}/{repo}/prs", hands.ListOpenPrs)
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	var response []map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response body: %v", err)
	}

	if len(response) != 2 {
		t.Errorf("Expected 2 pull requests, got %d", len(response))
	}

	if response[0]["title"] != "commit 1" {
		t.Errorf("Expected first PR title 'commit 1', got '%s'", response[0]["title"])
	}
}

func TestChangeRepoVisibility(t *testing.T) {
	hands := &Handler{GithubService: &MockGithubService{}}

	body := strings.NewReader(`{"private": true}`)
	req, _ := http.NewRequest("POST", "/repos/testuser/test-repo/change-visibility", body)
	rr := httptest.NewRecorder()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /repos/{owner}/{repo}/change-visibility", hands.ChangeRepoVisibility)
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	var response RepoResponse
	_ = json.NewDecoder(rr.Body).Decode(&response)

	if response.Visibility != "private" {
		t.Errorf("Expected visibility 'private', got '%s'", response.Visibility)
	}
}
