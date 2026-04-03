package download

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/emre-tiryaki/repograb/internal/models"
	"github.com/emre-tiryaki/repograb/internal/provider"
)

type DownloadEngine struct {
	Provider    provider.GitProvider
	BaseDir     string
	MaxParallel int
}

func (e *DownloadEngine) DownloadItems(items []models.FileNode) error {
	var wg sync.WaitGroup

	sem := make(chan struct{}, e.MaxParallel)
	errChan := make(chan error, len(items))

	for _, item := range items {
		if item.Type == "dir" {
			continue
		}

		wg.Add(1)
		go func(node models.FileNode) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() {
				<-sem
			}()

			err := e.DownloadSingleFile(node)
			if err != nil {
				errChan <- fmt.Errorf("Error when downloading %s: %w", node.Path, err)
			}
		}(item)
	}

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		return <-errChan
	}

	return nil
}

func (e *DownloadEngine) DownloadSingleFile(node models.FileNode) error {
	content, err := e.Provider.DownloadFile(node.DownloadUrl)
	if err != nil {
		return err
	}

	destPath := filepath.Join(e.BaseDir, node.Path)

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(destPath, content, 0644)
}
