package bench

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadDataset reads a JSONL file where each line is a JSON-encoded BenchTask.
func LoadDataset(path string) ([]BenchTask, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open dataset %s: %w", path, err)
	}
	defer f.Close()

	var tasks []BenchTask
	scanner := bufio.NewScanner(f)
	// Increase buffer for large problem statements.
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var task BenchTask
		if err := json.Unmarshal([]byte(line), &task); err != nil {
			return nil, fmt.Errorf("parse line %d in %s: %w", lineNum, path, err)
		}
		if task.ID == "" {
			return nil, fmt.Errorf("line %d in %s: missing instance_id", lineNum, path)
		}
		tasks = append(tasks, task)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read dataset %s: %w", path, err)
	}
	return tasks, nil
}

// ListDatasets scans a directory for .jsonl files and returns their paths.
func ListDatasets(dir string) ([]DatasetInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dataset directory %s: %w", dir, err)
	}
	var datasets []DatasetInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".jsonl" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		datasets = append(datasets, DatasetInfo{
			Name: strings.TrimSuffix(entry.Name(), ".jsonl"),
			Path: filepath.Join(dir, entry.Name()),
			Size: info.Size(),
		})
	}
	return datasets, nil
}

// DatasetInfo describes a discovered benchmark dataset file.
type DatasetInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size_bytes"`
}
