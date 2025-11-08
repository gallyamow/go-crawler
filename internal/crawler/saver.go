package crawler

import (
	"fmt"
	"os"
	"path/filepath"
)

type SaveResult struct {
	Path string
	Size int64
}

func (s *Saver) saveUrlContent(fileURL string, fileContent []byte) (string, os.FileInfo, error) {
	filename, err := generateFilename(fileURL)
	if err != nil {
		return "", nil, fmt.Errorf("generate file name: %w", err)
	}

	savePath := filepath.Join(s.baseDir, filename)
	if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
		return "", nil, fmt.Errorf("create directory: %w", err)
	}

	if err := os.WriteFile(savePath, fileContent, 0644); err != nil {
		return "", nil, fmt.Errorf("write file: %w", err)
	}

	fileInfo, err := os.Stat(savePath)
	if err != nil {
		return "", nil, fmt.Errorf("read file stat: %w", err)
	}

	return savePath, fileInfo, nil
}
