package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/emre-tiryaki/repograb/internal/download"
	"github.com/emre-tiryaki/repograb/internal/provider"
)

func main() {
	token := os.Getenv("GITHUB_TOKEN")
	ghProvider := &provider.GithubProvider{Token: token}

	owner := "abhixdd"
	repo := "ghgrab"
	branch := "main"

	fmt.Println("Fetch Repo Test Started:")

	fmt.Println("File system fetching")
	items, err := ghProvider.FetchTree(owner, repo, branch, "")
	if err != nil {
		log.Fatalf("fetching file system error: %v", err)
	}

	baseDir, _ := os.Getwd()
	outputDir := filepath.Join(baseDir, "outtput_test")

	engine := &download.DownloadEngine{
		Provider:    ghProvider,
		BaseDir:     baseDir,
		MaxParallel: 5,
	}

	fmt.Println("File Download Started")
	err = engine.DownloadItems(items)
	if err != nil {
		log.Fatalf("Error when downloading: %v", err)
	}

	fmt.Println("TEst complete")
	fmt.Printf("Files are in: %s\n", outputDir)
}
