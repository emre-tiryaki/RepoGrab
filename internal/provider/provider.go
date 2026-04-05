package provider

import (
	"errors"
	"strings"

	"github.com/emre-tiryaki/repograb/internal/models"
)

const (
	ProviderGitHub = "github"
	ProviderGitLab = "gitlab"
)

var (
	ErrInvalidRepositoryURL = errors.New("invalid repository url")
	ErrAuthRequired         = errors.New("authentication required")
	ErrRepositoryNotFound   = errors.New("repository not found")
)

type RepositorySpec struct {
	Provider      string
	Host          string
	ProjectPath   string
	DefaultBranch string
	Path          string
}

type GitProvider interface {
	Name() string
	ParseRepositoryURL(rawURL string) (*RepositorySpec, error)
	ResolveRepository(spec *RepositorySpec) error
	FetchTree(spec *RepositorySpec) ([]models.FileNode, error)
	DownloadFile(url string) ([]byte, error)
}

func normalizeRepositoryURL(rawURL string) string {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return trimmed
	}

	if strings.Contains(trimmed, "://") {
		return trimmed
	}

	return "https://" + trimmed
}
