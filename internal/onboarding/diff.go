package onboarding

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// FileStatus represents the status of a file in the merge change summary.
type FileStatus string

const (
	FileStatusNew       FileStatus = "new"        // File does not exist, will be created
	FileStatusPreserved FileStatus = "preserved"  // File exists, differs from default
	FileStatusUpToDate  FileStatus = "up_to_date" // File exists, matches default byte-for-byte
)

// FileChangeEntry represents a single file's status in the change summary.
type FileChangeEntry struct {
	RelPath  string
	Category string
	Status   FileStatus
}

// ManifestAction represents the type of change to a manifest key.
type ManifestAction string

const (
	ManifestActionAdded     ManifestAction = "added"
	ManifestActionPreserved ManifestAction = "preserved"
)

// ManifestChangeEntry represents a change to a manifest key.
type ManifestChangeEntry struct {
	KeyPath string
	Action  ManifestAction
}

// ChangeSummary holds the complete pre-mutation change report.
type ChangeSummary struct {
	Files           []FileChangeEntry
	ManifestChanges []ManifestChangeEntry
	MergedManifest  map[string]interface{}
	Assets          *AssetSet
	AlreadyUpToDate bool
}

// ComputeChangeSummary builds a pre-mutation change report by comparing on-disk
// files with embedded defaults and computing the manifest diff.
func ComputeChangeSummary(assets *AssetSet, existingManifest, defaultManifest map[string]interface{}) *ChangeSummary {
	var files []FileChangeEntry

	classifyFile := func(path, category, defaultContent string) FileChangeEntry {
		entry := FileChangeEntry{
			RelPath:  path,
			Category: category,
		}
		existing, err := os.ReadFile(path)
		switch {
		case err != nil:
			entry.Status = FileStatusNew
		case bytes.Equal(existing, []byte(defaultContent)):
			entry.Status = FileStatusUpToDate
		default:
			entry.Status = FileStatusPreserved
		}
		return entry
	}

	for filename, content := range assets.Personas {
		path := filepath.Join(".agents", "personas", filename)
		files = append(files, classifyFile(path, "persona", content))
	}
	for filename, content := range assets.Pipelines {
		path := filepath.Join(".agents", "pipelines", filename)
		files = append(files, classifyFile(path, "pipeline", content))
	}
	for filename, content := range assets.Contracts {
		path := filepath.Join(".agents", "contracts", filename)
		files = append(files, classifyFile(path, "contract", content))
	}
	for relPath, content := range assets.Prompts {
		path := filepath.Join(".agents", "prompts", relPath)
		files = append(files, classifyFile(path, "prompt", content))
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].RelPath < files[j].RelPath
	})

	merged, manifestChanges := ComputeManifestDiff(defaultManifest, existingManifest)

	alreadyUpToDate := true
	for _, f := range files {
		if f.Status == FileStatusNew {
			alreadyUpToDate = false
			break
		}
	}
	if alreadyUpToDate {
		for _, mc := range manifestChanges {
			if mc.Action == ManifestActionAdded {
				alreadyUpToDate = false
				break
			}
		}
	}

	return &ChangeSummary{
		Files:           files,
		ManifestChanges: manifestChanges,
		MergedManifest:  merged,
		Assets:          assets,
		AlreadyUpToDate: alreadyUpToDate,
	}
}

// ComputeManifestDiff performs the manifest merge and tracks what changed.
func ComputeManifestDiff(defaults, existing map[string]interface{}) (map[string]interface{}, []ManifestChangeEntry) {
	merged := MergeManifests(defaults, existing)
	var changes []ManifestChangeEntry
	collectManifestDiff("", defaults, existing, &changes)

	sort.Slice(changes, func(i, j int) bool {
		return changes[i].KeyPath < changes[j].KeyPath
	})
	return merged, changes
}

func collectManifestDiff(prefix string, defaults, existing map[string]interface{}, entries *[]ManifestChangeEntry) {
	for key, defaultVal := range defaults {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}

		existingVal, exists := existing[key]
		if !exists {
			*entries = append(*entries, ManifestChangeEntry{
				KeyPath: path,
				Action:  ManifestActionAdded,
			})
			continue
		}

		defaultMap, defaultIsMap := defaultVal.(map[string]interface{})
		existingMap, existingIsMap := existingVal.(map[string]interface{})

		if defaultIsMap && existingIsMap {
			collectManifestDiff(path, defaultMap, existingMap, entries)
		} else if fmt.Sprintf("%v", defaultVal) != fmt.Sprintf("%v", existingVal) {
			*entries = append(*entries, ManifestChangeEntry{
				KeyPath: path,
				Action:  ManifestActionPreserved,
			})
		}
	}

	for key := range existing {
		if _, inDefaults := defaults[key]; !inDefaults {
			path := key
			if prefix != "" {
				path = prefix + "." + key
			}
			*entries = append(*entries, ManifestChangeEntry{
				KeyPath: path,
				Action:  ManifestActionPreserved,
			})
		}
	}
}

// DisplayChangeSummary renders the ChangeSummary as a categorized table to the
// given writer (typically stderr).
func DisplayChangeSummary(w io.Writer, summary *ChangeSummary) {
	fmt.Fprintf(w, "\n  Change Summary:\n\n")

	categories := []struct {
		name  string
		label string
	}{
		{"persona", "Personas"},
		{"pipeline", "Pipelines"},
		{"contract", "Contracts"},
		{"prompt", "Prompts"},
	}

	for _, cat := range categories {
		var catFiles []FileChangeEntry
		for _, f := range summary.Files {
			if f.Category == cat.name {
				catFiles = append(catFiles, f)
			}
		}
		if len(catFiles) == 0 {
			continue
		}

		fmt.Fprintf(w, "  %s:\n", cat.label)
		for _, f := range catFiles {
			var status string
			switch f.Status {
			case FileStatusNew:
				status = "+ new"
			case FileStatusPreserved:
				status = "~ preserved"
			case FileStatusUpToDate:
				status = "= up to date"
			}
			fmt.Fprintf(w, "    %-14s %s\n", status, f.RelPath)
		}
		fmt.Fprintf(w, "\n")
	}

	if len(summary.ManifestChanges) > 0 {
		fmt.Fprintf(w, "  Manifest (wave.yaml):\n")
		for _, mc := range summary.ManifestChanges {
			var action string
			switch mc.Action {
			case ManifestActionAdded:
				action = "+ added"
			case ManifestActionPreserved:
				action = "~ preserved"
			}
			fmt.Fprintf(w, "    %-14s %s\n", action, mc.KeyPath)
		}
		fmt.Fprintf(w, "\n")
	}
}

// ApplyChanges writes only "new" files from the ChangeSummary and writes the
// merged manifest. Files with status "preserved" or "up_to_date" are not touched.
func ApplyChanges(summary *ChangeSummary, outputPath string) error {
	if err := EnsureWaveDirs(DefaultWaveDirs); err != nil {
		return err
	}

	for _, f := range summary.Files {
		if f.Status != FileStatusNew {
			continue
		}

		var content string
		switch f.Category {
		case "persona":
			content = summary.Assets.Personas[filepath.Base(f.RelPath)]
		case "pipeline":
			content = summary.Assets.Pipelines[filepath.Base(f.RelPath)]
		case "contract":
			content = summary.Assets.Contracts[filepath.Base(f.RelPath)]
		case "prompt":
			promptPrefix := filepath.Join(".agents", "prompts") + string(filepath.Separator)
			relPath := strings.TrimPrefix(f.RelPath, promptPrefix)
			content = summary.Assets.Prompts[relPath]
		}

		if err := os.MkdirAll(filepath.Dir(f.RelPath), 0755); err != nil {
			absPath, _ := filepath.Abs(f.RelPath)
			return fmt.Errorf("failed to create directory for %s: %w", absPath, err)
		}

		if err := os.WriteFile(f.RelPath, []byte(content), 0644); err != nil {
			absPath, _ := filepath.Abs(f.RelPath)
			return fmt.Errorf("failed to write %s: %w", absPath, err)
		}
	}

	mergedData, err := yaml.Marshal(summary.MergedManifest)
	if err != nil {
		return fmt.Errorf("failed to marshal merged manifest: %w", err)
	}

	if err := os.WriteFile(outputPath, mergedData, 0644); err != nil {
		absPath, _ := filepath.Abs(outputPath)
		return fmt.Errorf("failed to write manifest to %s: %w", absPath, err)
	}

	return nil
}
