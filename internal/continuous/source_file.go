package continuous

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
)

// FileSource reads work items from a file, one line per item.
type FileSource struct {
	Path  string
	items []*WorkItem
	index int
}

// NewFileSource creates a FileSource by loading all lines from the given path.
func NewFileSource(path string) (*FileSource, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open source file %q: %w", path, err)
	}
	defer f.Close()

	var items []*WorkItem
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		items = append(items, &WorkItem{
			ID:    line,
			Input: line,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read source file %q: %w", path, err)
	}

	return &FileSource{Path: path, items: items}, nil
}

func (s *FileSource) Next(_ context.Context) (*WorkItem, error) {
	if s.index >= len(s.items) {
		return nil, nil
	}
	item := s.items[s.index]
	s.index++
	return item, nil
}

func (s *FileSource) Name() string {
	return fmt.Sprintf("file(%s)", s.Path)
}
