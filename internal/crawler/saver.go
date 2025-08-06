package crawler

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type SaveResult struct {
	Path string
	Size int64
}

func (sr *SaveResult) String() string {
	return fmt.Sprintf("File saved to: %s, size: %d bytes", sr.Path, sr.Size)
}

type Saver struct {
	baseDir string
}

func NewSaver(baseDir string) *Saver {
	return &Saver{
		baseDir: baseDir,
	}
}

func (s *Saver) SavePage(page *Page) (*SaveResult, error) {
	path, fileInfo, err := s.saveUrlContent(page.URL, page.Content)
	if err != nil {
		return nil, fmt.Errorf("save page: %w", err)
	}

	return &SaveResult{Path: path, Size: fileInfo.Size()}, nil
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

func generateFilename(URL string) (string, error) {
	u, err := url.Parse(URL)
	if err != nil {
		return "", fmt.Errorf("parse url: %w", err)
	}

	segments := strings.Split(strings.Trim(u.Path, "/"), "/")

	// find last segment in url
	lastSegment := "index"
	if len(segments) > 0 {
		lastSegment = segments[len(segments)-1]
	}

	// cleaning
	safeSegment := strings.Map(func(r rune) rune {
		if r == '.' || r == '-' || r == '_' || r == '~' {
			return r
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, lastSegment)

	hash := md5.Sum([]byte(URL))
	hashStr := hex.EncodeToString(hash[:])

	return fmt.Sprintf("%s_%s.html", safeSegment, hashStr[:8]), nil
}
