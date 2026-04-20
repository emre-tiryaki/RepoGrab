package download

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/emre-tiryaki/repograb/internal/models"
	"github.com/emre-tiryaki/repograb/internal/provider"
)

type DownloadEngine struct {
	Provider    provider.GitProvider
	BaseDir     string
	MaxParallel int
	OnProgress  func(ProgressUpdate)
}

type ProgressUpdate struct {
	Completed int
	Total     int
	Path      string
	Err       error
}

func (e *DownloadEngine) DownloadItems(items []models.FileNode) error {
	var wg sync.WaitGroup
	var completedCount int32

	sem := make(chan struct{}, e.MaxParallel)
	errChan := make(chan error, len(items))

	totalFiles := 0
	for _, item := range items {
		if item.Type != "dir" {
			totalFiles++
		}
	}

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
			completed := int(atomic.AddInt32(&completedCount, 1))
			if e.OnProgress != nil {
				e.OnProgress(ProgressUpdate{
					Completed: completed,
					Total:     totalFiles,
					Path:      node.Path,
					Err:       err,
				})
			}
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
