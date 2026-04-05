package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/emre-tiryaki/repograb/internal/models"
)

// GithubProvider downloads repositories from GitHub.
type GithubProvider struct {
	Token string
}

func (g *GithubProvider) Name() string {
	return ProviderGitHub
}

func (g *GithubProvider) ParseRepositoryURL(rawURL string) (*RepositorySpec, error) {
	parsedURL, err := url.Parse(normalizeRepositoryURL(rawURL))
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, ErrInvalidRepositoryURL
	}

	if !strings.EqualFold(parsedURL.Host, "github.com") && !strings.HasSuffix(strings.ToLower(parsedURL.Host), ".github.com") {
		return nil, ErrInvalidRepositoryURL
	}

	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(pathParts) < 2 {
		return nil, ErrInvalidRepositoryURL
	}

	repoName := strings.TrimSuffix(pathParts[1], ".git")
	spec := &RepositorySpec{
		Provider:    ProviderGitHub,
		Host:        parsedURL.Host,
		ProjectPath: pathParts[0] + "/" + repoName,
	}

	if len(pathParts) >= 4 && pathParts[2] == "tree" {
		spec.DefaultBranch = pathParts[3]
		if len(pathParts) > 4 {
			spec.Path = strings.Join(pathParts[4:], "/")
		}
	}

	return spec, nil
}

func (g *GithubProvider) ResolveRepository(spec *RepositorySpec) error {
	parts := strings.SplitN(spec.ProjectPath, "/", 2)
	if len(parts) != 2 {
		return ErrInvalidRepositoryURL
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s", parts[0], parts[1])
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	if g.Token != "" {
		req.Header.Set("Authorization", "token "+g.Token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return ErrAuthRequired
	}
	if resp.StatusCode == http.StatusNotFound {
		if g.Token == "" {
			return ErrAuthRequired
		}
		return ErrRepositoryNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github repository lookup failed: %s", resp.Status)
	}

	var payload struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return err
	}

	spec.DefaultBranch = payload.DefaultBranch
	return nil
}

func (g *GithubProvider) FetchTree(spec *RepositorySpec) ([]models.FileNode, error) {
	parts := strings.SplitN(spec.ProjectPath, "/", 2)
	if len(parts) != 2 {
		return nil, ErrInvalidRepositoryURL
	}

	branch := spec.DefaultBranch
	if branch == "" {
		branch = "main"
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s", parts[0], parts[1], spec.Path, branch)
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	if g.Token != "" {
		req.Header.Set("Authorization", "token "+g.Token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, ErrAuthRequired
	}
	if resp.StatusCode == http.StatusNotFound {
		if g.Token == "" {
			return nil, ErrAuthRequired
		}
		return nil, ErrRepositoryNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github tree lookup failed: %s", resp.Status)
	}

	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	if len(raw) == 0 {
		return nil, nil
	}

	if raw[0] == '[' {
		var items []models.FileNode
		if err := json.Unmarshal(raw, &items); err != nil {
			return nil, err
		}
		return items, nil
	}

	var item models.FileNode
	if err := json.Unmarshal(raw, &item); err != nil {
		return nil, err
	}

	return []models.FileNode{item}, nil
}

func (g *GithubProvider) DownloadFile(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("request couldnt be created: %w", err)
	}

	if g.Token != "" {
		req.Header.Set("Authorization", "token "+g.Token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("file couldnt be downloaded: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, ErrAuthRequired
	}
	if resp.StatusCode == http.StatusNotFound {
		if g.Token == "" {
			return nil, ErrAuthRequired
		}
		return nil, ErrRepositoryNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server did not return 200 OK: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}
