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

// GitLabProvider downloads repositories from GitLab.
type GitLabProvider struct {
	Token string
}

func (g *GitLabProvider) Name() string {
	return ProviderGitLab
}

func (g *GitLabProvider) ParseRepositoryURL(rawURL string) (*RepositorySpec, error) {
	parsedURL, err := url.Parse(normalizeRepositoryURL(rawURL))
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, ErrInvalidRepositoryURL
	}

	trimmedPath := strings.Trim(parsedURL.Path, "/")
	if trimmedPath == "" {
		return nil, ErrInvalidRepositoryURL
	}

	hostLower := strings.ToLower(parsedURL.Host)
	if !strings.Contains(hostLower, "gitlab") && !strings.Contains(trimmedPath, "/-/tree/") && !strings.Contains(trimmedPath, "/-/blob/") {
		return nil, ErrInvalidRepositoryURL
	}

	spec := &RepositorySpec{
		Provider:    ProviderGitLab,
		Host:        parsedURL.Host,
		ProjectPath: trimmedPath,
	}

	if idx := strings.Index(trimmedPath, "/-/tree/"); idx >= 0 {
		spec.ProjectPath = strings.TrimSuffix(trimmedPath[:idx], "/")
		branchAndPath := trimmedPath[idx+len("/-/tree/"):]
		spec.DefaultBranch, spec.Path = splitFirstPath(branchAndPath)
		spec.ProjectPath = strings.TrimSuffix(spec.ProjectPath, ".git")
		return spec, nil
	}

	if idx := strings.Index(trimmedPath, "/-/blob/"); idx >= 0 {
		spec.ProjectPath = strings.TrimSuffix(trimmedPath[:idx], "/")
		branchAndPath := trimmedPath[idx+len("/-/blob/"):]
		spec.DefaultBranch, spec.Path = splitFirstPath(branchAndPath)
		spec.ProjectPath = strings.TrimSuffix(spec.ProjectPath, ".git")
		return spec, nil
	}

	spec.ProjectPath = strings.TrimSuffix(spec.ProjectPath, ".git")
	return spec, nil
}

func (g *GitLabProvider) ResolveRepository(spec *RepositorySpec) error {
	apiURL := fmt.Sprintf("https://%s/api/v4/projects/%s", spec.Host, url.PathEscape(spec.ProjectPath))
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return err
	}

	if g.Token != "" {
		req.Header.Set("PRIVATE-TOKEN", g.Token)
		req.Header.Set("Authorization", "Bearer "+g.Token)
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
		return fmt.Errorf("gitlab repository lookup failed: %s", resp.Status)
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

func (g *GitLabProvider) FetchTree(spec *RepositorySpec) ([]models.FileNode, error) {
	branch := spec.DefaultBranch
	if branch == "" {
		branch = "main"
	}

	baseURL := fmt.Sprintf("https://%s/api/v4/projects/%s/repository/tree", spec.Host, url.PathEscape(spec.ProjectPath))
	query := url.Values{}
	query.Set("ref", branch)
	if spec.Path != "" {
		query.Set("path", spec.Path)
	}

	req, err := http.NewRequest(http.MethodGet, baseURL+"?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}

	if g.Token != "" {
		req.Header.Set("PRIVATE-TOKEN", g.Token)
		req.Header.Set("Authorization", "Bearer "+g.Token)
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
		return nil, fmt.Errorf("gitlab tree lookup failed: %s", resp.Status)
	}

	var payload []struct {
		Name string `json:"name"`
		Path string `json:"path"`
		Type string `json:"type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	items := make([]models.FileNode, 0, len(payload))
	for _, item := range payload {
		nodeType := item.Type
		if nodeType == "tree" {
			nodeType = "dir"
		}

		node := models.FileNode{
			Name: item.Name,
			Path: item.Path,
			Type: nodeType,
		}

		if nodeType != "dir" {
			node.DownloadUrl = fmt.Sprintf(
				"https://%s/api/v4/projects/%s/repository/files/%s/raw?ref=%s",
				spec.Host,
				url.PathEscape(spec.ProjectPath),
				url.PathEscape(item.Path),
				url.QueryEscape(branch),
			)
		}

		items = append(items, node)
	}

	return items, nil
}

func (g *GitLabProvider) DownloadFile(rawURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request couldnt be created: %w", err)
	}

	if g.Token != "" {
		req.Header.Set("PRIVATE-TOKEN", g.Token)
		req.Header.Set("Authorization", "Bearer "+g.Token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("file couldnt be downloaded: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, ErrAuthRequired
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server did not return 200 OK: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func splitFirstPath(raw string) (string, string) {
	parts := strings.SplitN(strings.Trim(raw, "/"), "/", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}

	return parts[0], parts[1]
}
