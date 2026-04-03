package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/emre-tiryaki/repograb/internal/models"
)

// Provider for downloading repositories from github
type GithubProvider struct {
	Token string //access token for github
}

func (g *GithubProvider) FetchTree(owner, repo, branch, path string) ([]models.FileNode, error) {
	apiUrl := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s", owner, repo, path, branch)

	req, _ := http.NewRequest("GET", apiUrl, nil)
	if g.Token != "" {
		req.Header.Set("Authorization", "token "+g.Token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var items []models.FileNode
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}

	return items, nil
}

func (g *GithubProvider) DownloadFile(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("Request couldnt created %w", err)
	}

	if g.Token != "" {
		req.Header.Set("Authorization", "token "+g.Token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("File couldnt downloaded %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return  nil, fmt.Errorf("Server didnt return 200(OK): %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}
