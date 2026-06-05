package handlers

import (
	"context"

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
	return []*github.Repository{}, nil
}

func (m *MockGithubService) ListOpenPRs(ctx context.Context, owner, repo string) ([]*github.PullRequest, error) {
	return []*github.PullRequest{}, nil
}

// Tests
