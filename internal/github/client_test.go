package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	gh "github.com/google/go-github/v59/github"
)

// =====================================================================
// 1. THE TEST HELPER (The Fake Internet Factory)
// =====================================================================

// setupMockServer creates a fake GitHub API and wires a client to talk to it.
func setupMockServer(t *testing.T, expectedMethod, expectedPath string, statusCode int, responseBody interface{}) (*httptest.Server, *Client) {
	// 1. Create the Fake Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A. Verify the Outbound Translation (Did we send the right request?)
		if r.Method != expectedMethod {
			t.Errorf("Expected method %s, got %s", expectedMethod, r.Method)
		}
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		// B. Fulfill the Inbound Translation (Send data back to the client)
		w.WriteHeader(statusCode)
		if responseBody != nil {
			json.NewEncoder(w).Encode(responseBody)
		}
	}))

	// 2. Configure the underlying go-github client to talk to the fake server
	ghClient := gh.NewClient(nil)
	baseURL, _ := url.Parse(server.URL + "/") // go-github requires a trailing slash!
	ghClient.BaseURL = baseURL

	// Initialize YOUR wrapper struct with the faked underlying client
	myClient := &Client{
		Client: ghClient,
	}

	return server, myClient
}

// =====================================================================
// 2. THE TESTS
// =====================================================================

func TestCreateRepo_Success(t *testing.T) {
	// 1. Define the fake GitHub response
	expectedRepo := &gh.Repository{
		Name:    gh.String("new-test-repo"),
		HTMLURL: gh.String("https://github.com/testuser/new-test-repo"),
	}

	// 2. Spin up the fake server expecting a POST to /user/repos
	server, client := setupMockServer(t, http.MethodPost, "/user/repos", http.StatusCreated, expectedRepo)
	defer server.Close() // ALWAYS shut down the fake server when the test ends

	// 3. Call your actual function
	repo, err := client.CreateRepo(context.Background(), "new-test-repo")

	// 4. Assert your code handled the response correctly
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if repo.GetName() != "new-test-repo" {
		t.Errorf("Expected repo name 'new-test-repo', got '%s'", repo.GetName())
	}
}

func TestDeleteRepo_Success(t *testing.T) {
	// Delete returns 204 No Content and an empty body, so we pass 'nil' for the body
	server, client := setupMockServer(t, http.MethodDelete, "/repos/testowner/testrepo", http.StatusNoContent, nil)
	defer server.Close()

	err := client.DeleteRepo(context.Background(), "testowner", "testrepo")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestCheckRepo_NotFound(t *testing.T) {
	// 1. Define a standard GitHub Error response for a 404
	githubError := map[string]string{
		"message": "Not Found",
	}

	// 2. Spin up the server expecting a GET, but returning a 404 Status Code
	server, client := setupMockServer(t, http.MethodGet, "/repos/invalid-owner/missing-repo", http.StatusNotFound, githubError)
	defer server.Close()

	// 3. Call your code
	repo, err := client.CheckRepo(context.Background(), "invalid-owner", "missing-repo")

	// 4. Assert your code correctly trapped the error and didn't crash
	if err == nil {
		t.Fatal("Expected an error for a 404, got nil")
	}
	if repo != nil {
		t.Errorf("Expected nil repository on error, got %v", repo)
	}
}

func TestListRepos_Success(t *testing.T) {
	// 1. Define an array of fake repositories
	expectedRepos := []*gh.Repository{
		{Name: gh.String("repo-1"), HTMLURL: gh.String("https://github.com/user/repo-1")},
		{Name: gh.String("repo-2"), HTMLURL: gh.String("https://github.com/user/repo-2")},
	}

	// 2. Setup server expecting a GET request
	// Note: Adjust the expectedPath to match exactly what your client.go actually calls (e.g., "/user/repos" or "/users/username/repos")
	server, client := setupMockServer(t, http.MethodGet, "/user/repos", http.StatusOK, expectedRepos)
	defer server.Close()

	// 3. Call the client
	repos, err := client.ListRepos(context.Background())

	// 4. Verify it decoded the array correctly
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(repos) != 2 {
		t.Fatalf("Expected 2 repositories, got %d", len(repos))
	}
	if repos[0].GetName() != "repo-1" {
		t.Errorf("Expected first repo to be 'repo-1', got '%s'", repos[0].GetName())
	}
}

func TestListOpenPRs_Success(t *testing.T) {
	// 1. Define fake PRs
	expectedPRs := []*gh.PullRequest{
		{Title: gh.String("Fix a massive bug"), State: gh.String("open")},
	}

	// 2. Setup server expecting a GET to the specific pull requests endpoint
	expectedPath := "/repos/torvalds/linux/pulls"
	server, client := setupMockServer(t, http.MethodGet, expectedPath, http.StatusOK, expectedPRs)
	defer server.Close()

	// 3. Call the client
	prs, err := client.ListOpenPRs(context.Background(), "torvalds", "linux")

	// 4. Verify
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(prs) != 1 {
		t.Fatalf("Expected 1 PR, got %d", len(prs))
	}
	if prs[0].GetTitle() != "Fix a massive bug" {
		t.Errorf("Expected PR title 'Fix a massive bug', got '%s'", prs[0].GetTitle())
	}
}

func TestChangeRepoVisibility_Success(t *testing.T) {
	// 1. Define the expected updated repo response
	expectedRepo := &gh.Repository{
		Name:    gh.String("secret-repo"),
		Private: gh.Bool(true),
	}

	// 2. Setup server expecting a PATCH request
	expectedPath := "/repos/testuser/secret-repo"
	server, client := setupMockServer(t, http.MethodPatch, expectedPath, http.StatusOK, expectedRepo)
	defer server.Close()

	// 3. Call the client (setting private = true)
	repo, err := client.ChangeRepoVisibility(context.Background(), "testuser", "secret-repo", true)

	// 4. Verify
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if repo.GetPrivate() != true {
		t.Errorf("Expected repo to be private, but it was public")
	}
}
