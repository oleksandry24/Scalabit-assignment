package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v59/github"
)

// hols the official client isntance
type Client struct {
	*github.Client
}

// RepositoryService defines the interface for repository operations that our handlers will use.
// This allows us to mock this interface in tests.
type RepositoryService interface {
	CreateRepo(ctx context.Context, name string) (*github.Repository, error)
	DeleteRepo(ctx context.Context, owner, repo string) error
	ListRepos(ctx context.Context) ([]*github.Repository, error)
	ListOpenPRs(ctx context.Context, owner, repo string) ([]*github.PullRequest, error)

	CheckRepo(ctx context.Context, owner, repo string) (*github.Repository, error) // repo stats

	ChangeRepoVisibility(ctx context.Context, owner, repo string, private bool) (*github.Repository, error)
}

// the real github client using the real token
func NewClient(token string) RepositoryService {
	if token == "" || !strings.HasPrefix(token, "ghp_") && !strings.HasPrefix(token, "github_pat_") {
		return &Client{github.NewClient(nil)}
	}

	// authenticate the client with the token and check if the token is valid by making a simple API call
	ghClient := github.NewClient(nil).WithAuthToken(token)

	_, _, err := ghClient.Users.Get(context.Background(), "")
	if err != nil {
		panic(fmt.Sprintf("Invalid GitHub token: %v", err))
	}
	return &Client{ghClient}
}

// Methods

func (c *Client) CreateRepo(ctx context.Context, name string) (*github.Repository, error) {

	// create a new repository with the given name and confirm if the token is valid
	repo := &github.Repository{
		Name:    github.String(name),
		Private: github.Bool(true),
	}

	createdRepo, _, err := c.Repositories.Create(ctx, "", repo) // "" means user checked by the token
	return createdRepo, err
}

func (c *Client) DeleteRepo(ctx context.Context, owner, repo string) error {
	_, err := c.Repositories.Delete(ctx, owner, repo)
	return err
}

func (c *Client) ListRepos(ctx context.Context) ([]*github.Repository, error) {
	// Order by lat updated repo
	sortedRepos := &github.RepositoryListByAuthenticatedUserOptions{
		Sort: "updated",
	}

	// List repositories for the authenticated client
	repos, _, err := c.Repositories.ListByAuthenticatedUser(ctx, sortedRepos)
	return repos, err
}

func (c *Client) ListOpenPRs(ctx context.Context, owner, repo string) ([]*github.PullRequest, error) {
	openPrs := &github.PullRequestListOptions{
		State: "open",
	}

	prs, _, err := c.PullRequests.List(ctx, owner, repo, openPrs)
	return prs, err
}

func (c *Client) CheckRepo(ctx context.Context, owner, repo string) (*github.Repository, error) {
	repoInfo, _, err := c.Repositories.Get(ctx, owner, repo)
	return repoInfo, err
}

func (c *Client) ChangeRepoVisibility(ctx context.Context, owner, repo string, private bool) (*github.Repository, error) {

	repoUpdate := &github.Repository{
		Private: github.Bool(private),
	}

	updatedRepo, _, err := c.Repositories.Edit(ctx, owner, repo, repoUpdate)
	return updatedRepo, err
}
