package fileutil

import (
	"io"
	"os"
	"path/filepath"
)

// CopyFile copies a file or directory from src to dest.
// Parent directories of dest are created automatically.
// If src is a directory, it is copied recursively.
func CopyFile(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}

	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(src, dest)
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}

	if _, err = io.Copy(destFile, srcFile); err != nil {
		destFile.Close()
		return err
	}
	return destFile.Close()
}

func copyDir(src, dest string) error {
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, destPath); err != nil {
				return err
			}
		} else {
			if err := CopyFile(srcPath, destPath); err != nil {
				return err
			}
		}
	}

	return nil
}
