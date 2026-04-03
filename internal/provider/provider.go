package provider

import "github.com/emre-tiryaki/repograb/internal/models"

type GitProvider interface {
	FetchTree(owner, repo, branch, path string) ([]models.FileNode, error)	//Fetching the file tree for the repo
	DownloadFile(url string) ([]byte, error)								//Downloading single file
}