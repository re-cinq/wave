package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ArtifactRef struct {
	Step     string `yaml:"step"`
	Artifact string `yaml:"artifact"`
	As       string `yaml:"as"`
}

type WorkspaceConfig struct {
	Root  string  `yaml:"root"`
	Mount []Mount `yaml:"mount"`
}

type Mount struct {
	Source string `yaml:"source"`
	Target string `yaml:"target"`
	Mode   string `yaml:"mode"`
}

type WorkspaceManager interface {
	Create(cfg WorkspaceConfig, templateVars map[string]string) (string, error)
	InjectArtifacts(workspacePath string, refs []ArtifactRef, resolvedPaths map[string]string) error
	CleanAll(root string) error
}

type workspaceManager struct {
	baseDir string
}

func NewWorkspaceManager(baseDir string) (WorkspaceManager, error) {
	if baseDir == "" {
		baseDir = "/tmp/wave"
	}
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base workspace directory: %w", err)
	}
	return &workspaceManager{baseDir: baseDir}, nil
}

func substituteVars(path string, vars map[string]string) string {
	if vars == nil {
		return path
	}
	result := path
	for key, value := range vars {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

func (wm *workspaceManager) Create(cfg WorkspaceConfig, templateVars map[string]string) (string, error) {
	if len(cfg.Mount) == 0 {
		return "", errors.New("at least one mount configuration is required")
	}

	pipelineID, ok := templateVars["pipeline_id"]
	if !ok {
		pipelineID = "unknown"
	}

	stepID, ok := templateVars["step_id"]
	if !ok {
		stepID = "unknown"
	}

	workspacePath := filepath.Join(wm.baseDir, pipelineID, stepID)
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		return "", fmt.Errorf("failed to create workspace directory: %w", err)
	}

	for _, mount := range cfg.Mount {
		if mount.Source == "" || mount.Target == "" {
			return "", fmt.Errorf("mount source and target cannot be empty")
		}

		source := substituteVars(mount.Source, templateVars)
		target := filepath.Join(workspacePath, substituteVars(mount.Target, templateVars))

		if _, err := os.Stat(source); os.IsNotExist(err) {
			return "", fmt.Errorf("mount source does not exist: %s", source)
		}

		if err := os.MkdirAll(target, 0755); err != nil {
			return "", fmt.Errorf("failed to create mount target: %w", err)
		}

		perm := os.FileMode(0755)
		if mount.Mode == "readonly" {
			perm = os.FileMode(0555)
		}

		if err := os.Chmod(target, perm); err != nil {
			return "", fmt.Errorf("failed to set mount permissions: %w", err)
		}

		if err := copyRecursive(source, target); err != nil {
			return "", fmt.Errorf("failed to copy mount source: %w", err)
		}
	}

	return workspacePath, nil
}

func (wm *workspaceManager) InjectArtifacts(workspacePath string, refs []ArtifactRef, resolvedPaths map[string]string) error {
	if workspacePath == "" {
		return errors.New("workspacePath cannot be empty")
	}

	artifactsDir := filepath.Join(workspacePath, "artifacts")
	if err := os.MkdirAll(artifactsDir, 0755); err != nil {
		return fmt.Errorf("failed to create artifacts directory: %w", err)
	}

	for _, ref := range refs {
		if ref.Step == "" || ref.Artifact == "" {
			continue
		}

		resolvedKey := fmt.Sprintf("%s:%s", ref.Step, ref.Artifact)
		resolvedPath, ok := resolvedPaths[resolvedKey]
		if !ok {
			resolvedPath, ok = resolvedPaths[ref.Step]
		}
		if !ok {
			continue
		}

		if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
			continue
		}

		artName := ref.As
		if artName == "" {
			artName = strings.ReplaceAll(ref.Artifact, "/", "_")
		}
		artName = fmt.Sprintf("%s_%s", ref.Step, artName)

		artPath := filepath.Join(artifactsDir, artName)

		if err := copyRecursive(resolvedPath, artPath); err != nil {
			return fmt.Errorf("failed to inject artifact %s: %w", ref.Artifact, err)
		}
	}

	return nil
}

func copyRecursive(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if srcInfo.IsDir() {
		return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			relPath, _ := filepath.Rel(src, path)
			targetPath := filepath.Join(dst, relPath)
			if info.IsDir() {
				return os.MkdirAll(targetPath, 0755)
			}
			return copyFile(path, targetPath)
		})
	}

	return copyFile(src, dst)
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := dstFile.ReadFrom(srcFile); err != nil {
		return err
	}

	return dstFile.Sync()
}

func (wm *workspaceManager) CleanAll(root string) error {
	if root == "" {
		return errors.New("root cannot be empty")
	}

	if !filepath.IsAbs(root) {
		root = filepath.Join(wm.baseDir, root)
	}

	return os.RemoveAll(root)
}
