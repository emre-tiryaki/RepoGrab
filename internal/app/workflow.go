package app

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/emre-tiryaki/repograb/internal/download"
	"github.com/emre-tiryaki/repograb/internal/models"
	"github.com/emre-tiryaki/repograb/internal/provider"
)

type Service struct{}

type ProgressUpdate = download.ProgressUpdate

type LoadResult struct {
	Spec  provider.RepositorySpec
	Items []models.FileNode
}

func (s Service) DetectProvider(rawURL string, tokens map[string]string) (provider.GitProvider, *provider.RepositorySpec, error) {
	providers := []provider.GitProvider{
		&provider.GithubProvider{Token: tokens[provider.ProviderGitHub]},
		&provider.GitLabProvider{Token: tokens[provider.ProviderGitLab]},
	}

	for _, selectedProvider := range providers {
		spec, err := selectedProvider.ParseRepositoryURL(rawURL)
		if err == nil {
			if spec.DefaultBranch == "" {
				spec.DefaultBranch = "main"
			}
			return selectedProvider, spec, nil
		}
	}

	return nil, nil, provider.ErrInvalidRepositoryURL
}

func (s Service) ProviderForSpec(spec provider.RepositorySpec, tokens map[string]string) provider.GitProvider {
	token := tokens[spec.Provider]

	switch spec.Provider {
	case provider.ProviderGitLab:
		return &provider.GitLabProvider{Token: token}
	default:
		return &provider.GithubProvider{Token: token}
	}
}

func (s Service) ResolveAndFetch(gitProvider provider.GitProvider, spec provider.RepositorySpec) (LoadResult, error) {
	resolvedSpec := spec
	if err := gitProvider.ResolveRepository(&resolvedSpec); err != nil {
		return LoadResult{}, err
	}

	items, err := gitProvider.FetchTree(&resolvedSpec)
	if err != nil {
		return LoadResult{}, err
	}

	return LoadResult{Spec: resolvedSpec, Items: items}, nil
}

func (s Service) DownloadSelected(
	gitProvider provider.GitProvider,
	spec provider.RepositorySpec,
	baseDir string,
	items []models.FileNode,
	onProgress func(download.ProgressUpdate),
) (string, int, error) {
	if len(items) == 0 {
		return "", 0, errors.New("no files selected")
	}

	cleanBaseDir := strings.TrimSpace(baseDir)
	if cleanBaseDir == "" {
		return "", 0, errors.New("download directory is required")
	}

	repoDirName := strings.ReplaceAll(spec.ProjectPath, "/", "_")
	targetDir := filepath.Join(cleanBaseDir, repoDirName)

	filesToDownload, err := s.expandDownloadItems(gitProvider, spec, items)
	if err != nil {
		return targetDir, 0, err
	}
	if len(filesToDownload) == 0 {
		return targetDir, 0, errors.New("no downloadable files found in selected items")
	}

	engine := &download.DownloadEngine{
		Provider:    gitProvider,
		BaseDir:     targetDir,
		MaxParallel: 5,
		OnProgress:  onProgress,
	}

	if err := engine.DownloadItems(filesToDownload); err != nil {
		return targetDir, 0, err
	}

	return targetDir, len(filesToDownload), nil
}

func (s Service) expandDownloadItems(
	gitProvider provider.GitProvider,
	spec provider.RepositorySpec,
	items []models.FileNode,
) ([]models.FileNode, error) {
	visitedDirs := make(map[string]struct{})
	seenFiles := make(map[string]struct{})
	result := make([]models.FileNode, 0)

	addFile := func(node models.FileNode) {
		if strings.TrimSpace(node.DownloadUrl) == "" {
			return
		}
		if _, ok := seenFiles[node.Path]; ok {
			return
		}

		seenFiles[node.Path] = struct{}{}
		result = append(result, node)
	}

	var walkDir func(string) error
	walkDir = func(path string) error {
		trimmedPath := strings.Trim(strings.TrimSpace(path), "/")
		if _, ok := visitedDirs[trimmedPath]; ok {
			return nil
		}
		visitedDirs[trimmedPath] = struct{}{}

		nextSpec := spec
		nextSpec.Path = trimmedPath

		nodes, err := gitProvider.FetchTree(&nextSpec)
		if err != nil {
			return err
		}

		for _, node := range nodes {
			if node.Type == "dir" {
				if err := walkDir(node.Path); err != nil {
					return err
				}
				continue
			}

			addFile(node)
		}

		return nil
	}

	for _, item := range items {
		if item.Type == "dir" {
			if err := walkDir(item.Path); err != nil {
				return nil, err
			}
			continue
		}

		addFile(item)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})

	return result, nil
}
