package skill

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxExtractedFiles = 1000
	maxExtractedSize  = 100 * 1024 * 1024 // 100 MB
)

// URLAdapter installs skills from remote archive URLs.
type URLAdapter struct {
	client *http.Client
}

// NewURLAdapter creates a URLAdapter with configured timeouts.
func NewURLAdapter() *URLAdapter {
	return &URLAdapter{
		client: &http.Client{
			Transport: &http.Transport{
				ResponseHeaderTimeout: HTTPHeaderTimeout,
			},
		},
	}
}

// Prefix returns "https://".
func (a *URLAdapter) Prefix() string { return "https://" }

// Install downloads an archive from the URL, extracts it, and installs discovered skills.
func (a *URLAdapter) Install(ctx context.Context, ref string, store Store) (*InstallResult, error) {
	if !strings.HasPrefix(ref, "https://") && !strings.HasPrefix(ref, "http://") {
		return nil, fmt.Errorf("invalid URL: must start with https:// or http://")
	}

	ctx, cancel := context.WithTimeout(ctx, HTTPTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ref, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download %s: %w", ref, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d downloading %s", resp.StatusCode, ref)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "wave-skill-url-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Detect archive format by URL extension
	lower := strings.ToLower(ref)
	switch {
	case strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz"):
		if err := extractTarGz(bytes.NewReader(body), tmpDir); err != nil {
			return nil, fmt.Errorf("failed to extract tar.gz: %w", err)
		}
	case strings.HasSuffix(lower, ".zip"):
		if err := extractZip(body, tmpDir); err != nil {
			return nil, fmt.Errorf("failed to extract zip: %w", err)
		}
	default:
		return nil, fmt.Errorf("unrecognized archive format for %s: supported formats are .tar.gz, .tgz, .zip", ref)
	}

	paths, err := discoverSkillFiles(tmpDir)
	if err != nil {
		return nil, err
	}

	return parseAndWriteSkills(ctx, paths, store)
}

// extractTarGz extracts a tar.gz archive to destDir.
func extractTarGz(r io.Reader, destDir string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	fileCount := 0
	var totalSize int64

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		// Reject path traversal
		if strings.Contains(hdr.Name, "..") {
			return fmt.Errorf("path traversal detected in archive: %s", hdr.Name)
		}

		target := filepath.Join(destDir, hdr.Name)

		// Ensure target stays within destDir
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)+string(filepath.Separator)) {
			return fmt.Errorf("path traversal detected: %s escapes extraction directory", hdr.Name)
		}

		fileCount++
		if fileCount > maxExtractedFiles {
			return fmt.Errorf("archive contains too many files (limit: %d)", maxExtractedFiles)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			totalSize += hdr.Size
			if totalSize > maxExtractedSize {
				return fmt.Errorf("archive exceeds size limit (%d bytes)", maxExtractedSize)
			}

			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)&0755)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("failed to write file: %w", err)
			}
			f.Close()
		}
	}

	return nil
}

// extractZip extracts a zip archive from bytes to destDir.
func extractZip(data []byte, destDir string) error {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}

	fileCount := 0
	var totalSize int64

	for _, f := range r.File {
		// Reject path traversal
		if strings.Contains(f.Name, "..") {
			return fmt.Errorf("path traversal detected in archive: %s", f.Name)
		}

		target := filepath.Join(destDir, f.Name)

		// Ensure target stays within destDir
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)+string(filepath.Separator)) {
			return fmt.Errorf("path traversal detected: %s escapes extraction directory", f.Name)
		}

		fileCount++
		if fileCount > maxExtractedFiles {
			return fmt.Errorf("archive contains too many files (limit: %d)", maxExtractedFiles)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			continue
		}

		totalSize += int64(f.UncompressedSize64)
		if totalSize > maxExtractedSize {
			return fmt.Errorf("archive exceeds size limit (%d bytes)", maxExtractedSize)
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open file in archive: %w", err)
		}

		outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode()&0755)
		if err != nil {
			rc.Close()
			return fmt.Errorf("failed to create file: %w", err)
		}

		if _, err := io.Copy(outFile, rc); err != nil {
			outFile.Close()
			rc.Close()
			return fmt.Errorf("failed to write file: %w", err)
		}

		outFile.Close()
		rc.Close()
	}

	return nil
}
