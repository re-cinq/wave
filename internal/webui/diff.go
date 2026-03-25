package webui

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const maxDiffSize = 100 * 1024 // 100KB per file diff

// gitCommandTimeout is the maximum time allowed for a single git subprocess.
const gitCommandTimeout = 30 * time.Second

// gitCommand creates a git command with a context timeout and explicit working directory.
func gitCommand(ctx context.Context, repoDir string, args ...string) *exec.Cmd {
	ctx, cancel := context.WithTimeout(ctx, gitCommandTimeout)
	_ = cancel // cancel will fire when context expires or parent cancels
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoDir
	return cmd
}

// resolveBaseBranch determines the base branch for diff comparison.
// Resolution order: (1) git symbolic-ref refs/remotes/origin/HEAD,
// (2) check if main exists, (3) check if master exists.
func resolveBaseBranch(ctx context.Context, repoDir string) (string, error) {
	// Try origin/HEAD first
	out, err := gitCommand(ctx, repoDir, "symbolic-ref", "refs/remotes/origin/HEAD").Output()
	if err == nil {
		ref := strings.TrimSpace(string(out))
		branch := strings.TrimPrefix(ref, "refs/remotes/origin/")
		if branch != ref && branch != "" {
			return branch, nil
		}
	}

	// Fallback: check if main exists
	if err := gitCommand(ctx, repoDir, "rev-parse", "--verify", "main").Run(); err == nil {
		return "main", nil
	}

	// Fallback: check if master exists
	if err := gitCommand(ctx, repoDir, "rev-parse", "--verify", "master").Run(); err == nil {
		return "master", nil
	}

	return "", fmt.Errorf("no base branch could be determined")
}

// computeDiffSummary computes the diff summary between base and head branches.
// Returns a DiffSummary with Available=false if the branch is deleted or unreachable.
func computeDiffSummary(ctx context.Context, repoDir, baseBranch, headBranch string) *DiffSummary {
	diffRange := baseBranch + "..." + headBranch

	// Get per-file additions/deletions
	numstatOut, err := gitCommand(ctx, repoDir, "diff", "--numstat", diffRange).Output()
	if err != nil {
		return &DiffSummary{
			Available: false,
			Message:   "Branch deleted — diff unavailable",
		}
	}

	// Get per-file change status
	nameStatusOut, err := gitCommand(ctx, repoDir, "diff", "--name-status", diffRange).Output()
	if err != nil {
		return &DiffSummary{
			Available: false,
			Message:   "Branch deleted — diff unavailable",
		}
	}

	// Parse name-status into a map
	statusMap := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(string(nameStatusOut)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 2 {
			continue
		}
		status := parts[0]
		path := parts[1]
		switch {
		case strings.HasPrefix(status, "R"):
			statusMap[path] = "renamed"
			if len(parts) >= 3 {
				statusMap[parts[2]] = "renamed"
			}
		case status == "A":
			statusMap[path] = "added"
		case status == "D":
			statusMap[path] = "deleted"
		case status == "M":
			statusMap[path] = "modified"
		default:
			statusMap[path] = "modified"
		}
	}

	// Parse numstat output
	var files []FileSummary
	totalAdditions := 0
	totalDeletions := 0

	for _, line := range strings.Split(strings.TrimSpace(string(numstatOut)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}

		path := parts[2]
		// Handle renames: numstat shows "oldpath => newpath" or "{old => new}/file"
		if strings.Contains(path, " => ") {
			// Use the new path for the summary
			idx := strings.Index(path, " => ")
			if strings.Contains(path, "{") {
				// Format: path/{old => new}/rest
				braceStart := strings.Index(path, "{")
				braceEnd := strings.Index(path, "}")
				if braceStart >= 0 && braceEnd > braceStart {
					prefix := path[:braceStart]
					suffix := path[braceEnd+1:]
					inner := path[braceStart+1 : braceEnd]
					innerParts := strings.SplitN(inner, " => ", 2)
					if len(innerParts) == 2 {
						path = prefix + innerParts[1] + suffix
					}
				}
			} else {
				path = path[idx+4:]
			}
		}

		binary := parts[0] == "-" && parts[1] == "-"
		var additions, deletions int
		if !binary {
			additions, _ = strconv.Atoi(parts[0])
			deletions, _ = strconv.Atoi(parts[1])
			totalAdditions += additions
			totalDeletions += deletions
		}

		status := statusMap[path]
		if status == "" {
			status = "modified"
		}

		files = append(files, FileSummary{
			Path:      path,
			Status:    status,
			Additions: additions,
			Deletions: deletions,
			Binary:    binary,
		})
	}

	// Sort alphabetically by path (FR-007)
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	if files == nil {
		files = []FileSummary{}
	}

	return &DiffSummary{
		Files:          files,
		TotalFiles:     len(files),
		TotalAdditions: totalAdditions,
		TotalDeletions: totalDeletions,
		BaseBranch:     baseBranch,
		HeadBranch:     headBranch,
		Available:      true,
	}
}

// computeFileDiff computes the diff for a single file between base and head branches.
func computeFileDiff(ctx context.Context, repoDir, baseBranch, headBranch, filePath string) (*FileDiff, error) {
	// Validate path: reject traversal and absolute paths (FR-013)
	cleanPath := filepath.Clean(filePath)
	if strings.Contains(cleanPath, "..") || strings.HasPrefix(cleanPath, "/") {
		return nil, fmt.Errorf("invalid file path")
	}

	diffRange := baseBranch + "..." + headBranch

	// Check if file is binary
	numstatOut, err := gitCommand(ctx, repoDir, "diff", "--numstat", diffRange, "--", cleanPath).Output()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}

	numstatLine := strings.TrimSpace(string(numstatOut))
	if numstatLine == "" {
		return &FileDiff{
			Path:   cleanPath,
			Status: "modified",
		}, nil
	}

	parts := strings.SplitN(numstatLine, "\t", 3)
	isBinary := len(parts) >= 2 && parts[0] == "-" && parts[1] == "-"

	if isBinary {
		return &FileDiff{
			Path:   cleanPath,
			Status: detectFileStatus(ctx, repoDir, baseBranch, headBranch, cleanPath),
			Binary: true,
		}, nil
	}

	// Get the actual diff content
	diffOut, err := gitCommand(ctx, repoDir, "diff", diffRange, "--", cleanPath).Output()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}

	content := string(diffOut)
	originalSize := len(content)
	truncated := false

	if originalSize > maxDiffSize {
		content = content[:maxDiffSize]
		truncated = true
	}

	// Count additions and deletions from diff lines
	additions := 0
	deletions := 0
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
			continue
		}
		if strings.HasPrefix(line, "+") {
			additions++
		} else if strings.HasPrefix(line, "-") {
			deletions++
		}
	}

	return &FileDiff{
		Path:      cleanPath,
		Status:    detectFileStatus(ctx, repoDir, baseBranch, headBranch, cleanPath),
		Additions: additions,
		Deletions: deletions,
		Content:   content,
		Truncated: truncated,
		Size:      originalSize,
	}, nil
}

// detectFileStatus determines the change status of a file.
func detectFileStatus(ctx context.Context, repoDir, baseBranch, headBranch, filePath string) string {
	diffRange := baseBranch + "..." + headBranch
	out, err := gitCommand(ctx, repoDir, "diff", "--name-status", diffRange, "--", filePath).Output()
	if err != nil {
		return "modified"
	}
	line := strings.TrimSpace(string(out))
	if line == "" {
		return "modified"
	}
	parts := strings.SplitN(line, "\t", 2)
	switch {
	case strings.HasPrefix(parts[0], "R"):
		return "renamed"
	case parts[0] == "A":
		return "added"
	case parts[0] == "D":
		return "deleted"
	default:
		return "modified"
	}
}
