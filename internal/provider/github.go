package provider

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/emre-tiryaki/repograb/internal/models"
)

//Provider for downloading repositories from github
type GithubProvider struct {
	Token string	//access token for github
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
	return nil, nil
}