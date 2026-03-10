package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/pipeline"
	"github.com/recinq/wave/internal/state"
	"gopkg.in/yaml.v3"
)

// HealthCheckStatus represents the result of a health check.
type HealthCheckStatus int

const (
	HealthCheckOK       HealthCheckStatus = iota
	HealthCheckWarn
	HealthCheckErr
	HealthCheckChecking
)

// HealthCheckResultMsg carries the result of an async health check.
type HealthCheckResultMsg struct {
	Name    string
	Status  HealthCheckStatus
	Message string
	Details map[string]string
}

// HealthDataProvider executes health checks for the Health view.
type HealthDataProvider interface {
	RunCheck(name string) HealthCheckResultMsg
	CheckNames() []string
}

// DefaultHealthDataProvider implements HealthDataProvider.
type DefaultHealthDataProvider struct {
	manifest     *manifest.Manifest
	store        state.StateStore
	pipelinesDir string
}

// NewDefaultHealthDataProvider creates a new health data provider.
func NewDefaultHealthDataProvider(m *manifest.Manifest, store state.StateStore, pipelinesDir string) *DefaultHealthDataProvider {
	return &DefaultHealthDataProvider{
		manifest:     m,
		store:        store,
		pipelinesDir: pipelinesDir,
	}
}

// CheckNames returns the ordered list of check names.
func (p *DefaultHealthDataProvider) CheckNames() []string {
	return []string{
		"Git Repository",
		"Adapter Binary",
		"SQLite Database",
		"Wave Configuration",
		"Required Tools",
		"Required Skills",
	}
}

// RunCheck runs a single health check by name.
func (p *DefaultHealthDataProvider) RunCheck(name string) HealthCheckResultMsg {
	switch name {
	case "Git Repository":
		return p.checkGitRepository()
	case "Adapter Binary":
		return p.checkAdapterBinary()
	case "SQLite Database":
		return p.checkSQLiteDatabase()
	case "Wave Configuration":
		return p.checkWaveConfiguration()
	case "Required Tools":
		return p.checkRequiredTools()
	case "Required Skills":
		return p.checkRequiredSkills()
	default:
		return HealthCheckResultMsg{
			Name:    name,
			Status:  HealthCheckErr,
			Message: "Unknown check",
		}
	}
}

func (p *DefaultHealthDataProvider) checkGitRepository() HealthCheckResultMsg {
	details := make(map[string]string)

	out, err := exec.Command("git", "rev-parse", "--is-inside-work-tree").CombinedOutput()
	if err != nil {
		return HealthCheckResultMsg{
			Name:    "Git Repository",
			Status:  HealthCheckErr,
			Message: "Not a git repository",
			Details: details,
		}
	}
	details["Work tree"] = strings.TrimSpace(string(out))

	if branchOut, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").CombinedOutput(); err == nil {
		details["Branch"] = strings.TrimSpace(string(branchOut))
	}

	if remoteOut, err := exec.Command("git", "remote", "get-url", "origin").CombinedOutput(); err == nil {
		details["Remote"] = strings.TrimSpace(string(remoteOut))
	}

	status := HealthCheckOK
	message := "Valid git repository"

	if dirtyOut, err := exec.Command("git", "status", "--porcelain").CombinedOutput(); err == nil {
		dirty := strings.TrimSpace(string(dirtyOut))
		if dirty != "" {
			status = HealthCheckWarn
			message = "Working tree has uncommitted changes"
			details["Dirty files"] = fmt.Sprintf("%d", strings.Count(dirty, "\n")+1)
		}
	}

	return HealthCheckResultMsg{
		Name:    "Git Repository",
		Status:  status,
		Message: message,
		Details: details,
	}
}

func (p *DefaultHealthDataProvider) checkAdapterBinary() HealthCheckResultMsg {
	details := make(map[string]string)

	if p.manifest == nil || len(p.manifest.Adapters) == 0 {
		return HealthCheckResultMsg{
			Name:    "Adapter Binary",
			Status:  HealthCheckOK,
			Message: "No adapters configured",
			Details: details,
		}
	}

	allFound := true
	for name, adapter := range p.manifest.Adapters {
		path, err := exec.LookPath(adapter.Binary)
		if err != nil {
			details[name] = fmt.Sprintf("%s — not found", adapter.Binary)
			allFound = false
		} else {
			details[name] = fmt.Sprintf("%s — %s", adapter.Binary, path)
		}
	}

	if allFound {
		return HealthCheckResultMsg{
			Name:    "Adapter Binary",
			Status:  HealthCheckOK,
			Message: fmt.Sprintf("All %d adapter binaries found", len(p.manifest.Adapters)),
			Details: details,
		}
	}

	return HealthCheckResultMsg{
		Name:    "Adapter Binary",
		Status:  HealthCheckErr,
		Message: "Some adapter binaries not found",
		Details: details,
	}
}

func (p *DefaultHealthDataProvider) checkSQLiteDatabase() HealthCheckResultMsg {
	details := make(map[string]string)

	if p.store == nil {
		return HealthCheckResultMsg{
			Name:    "SQLite Database",
			Status:  HealthCheckErr,
			Message: "No state store configured",
			Details: details,
		}
	}

	_, err := p.store.ListRuns(state.ListRunsOptions{Limit: 1})
	if err != nil {
		return HealthCheckResultMsg{
			Name:    "SQLite Database",
			Status:  HealthCheckErr,
			Message: fmt.Sprintf("Database query failed: %s", err),
			Details: details,
		}
	}

	details["Status"] = "Connected"

	return HealthCheckResultMsg{
		Name:    "SQLite Database",
		Status:  HealthCheckOK,
		Message: "Database accessible",
		Details: details,
	}
}

func (p *DefaultHealthDataProvider) checkWaveConfiguration() HealthCheckResultMsg {
	details := make(map[string]string)

	if p.manifest == nil {
		return HealthCheckResultMsg{
			Name:    "Wave Configuration",
			Status:  HealthCheckErr,
			Message: "No manifest loaded",
			Details: details,
		}
	}

	details["Personas"] = fmt.Sprintf("%d", len(p.manifest.Personas))
	details["Adapters"] = fmt.Sprintf("%d", len(p.manifest.Adapters))

	// Count pipelines from directory
	pipelineCount := 0
	if p.pipelinesDir != "" {
		entries, err := os.ReadDir(p.pipelinesDir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					ext := filepath.Ext(entry.Name())
					if ext == ".yaml" || ext == ".yml" {
						pipelineCount++
					}
				}
			}
		}
	}
	details["Pipelines"] = fmt.Sprintf("%d", pipelineCount)

	return HealthCheckResultMsg{
		Name:    "Wave Configuration",
		Status:  HealthCheckOK,
		Message: "Configuration valid",
		Details: details,
	}
}

func (p *DefaultHealthDataProvider) checkRequiredTools() HealthCheckResultMsg {
	details := make(map[string]string)

	if p.pipelinesDir == "" {
		return HealthCheckResultMsg{
			Name:    "Required Tools",
			Status:  HealthCheckOK,
			Message: "No pipelines directory configured",
			Details: details,
		}
	}

	// Collect all required tools across pipelines
	toolSet := make(map[string]bool)
	entries, err := os.ReadDir(p.pipelinesDir)
	if err != nil {
		return HealthCheckResultMsg{
			Name:    "Required Tools",
			Status:  HealthCheckOK,
			Message: "No pipelines found",
			Details: details,
		}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(p.pipelinesDir, entry.Name()))
		if err != nil {
			continue
		}
		var pl pipeline.Pipeline
		if err := yaml.Unmarshal(data, &pl); err != nil {
			continue
		}
		if pl.Requires != nil {
			for _, tool := range pl.Requires.Tools {
				toolSet[tool] = true
			}
		}
	}

	if len(toolSet) == 0 {
		return HealthCheckResultMsg{
			Name:    "Required Tools",
			Status:  HealthCheckOK,
			Message: "No tools required",
			Details: details,
		}
	}

	allFound := true
	for tool := range toolSet {
		if _, err := exec.LookPath(tool); err != nil {
			details[tool] = "not found"
			allFound = false
		} else {
			details[tool] = "available"
		}
	}

	if allFound {
		return HealthCheckResultMsg{
			Name:    "Required Tools",
			Status:  HealthCheckOK,
			Message: fmt.Sprintf("All %d tools available", len(toolSet)),
			Details: details,
		}
	}

	return HealthCheckResultMsg{
		Name:    "Required Tools",
		Status:  HealthCheckErr,
		Message: "Some required tools missing",
		Details: details,
	}
}

func (p *DefaultHealthDataProvider) checkRequiredSkills() HealthCheckResultMsg {
	details := make(map[string]string)

	if p.pipelinesDir == "" {
		return HealthCheckResultMsg{
			Name:    "Required Skills",
			Status:  HealthCheckOK,
			Message: "No pipelines directory configured",
			Details: details,
		}
	}

	entries, err := os.ReadDir(p.pipelinesDir)
	if err != nil {
		return HealthCheckResultMsg{
			Name:    "Required Skills",
			Status:  HealthCheckOK,
			Message: "No pipelines found",
			Details: details,
		}
	}

	// Collect all required skills
	type skillEntry struct {
		check string
	}
	skillMap := make(map[string]skillEntry)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(p.pipelinesDir, entry.Name()))
		if err != nil {
			continue
		}
		var pl pipeline.Pipeline
		if err := yaml.Unmarshal(data, &pl); err != nil {
			continue
		}
		if pl.Requires != nil {
			for name, cfg := range pl.Requires.Skills {
				skillMap[name] = skillEntry{check: cfg.Check}
			}
		}
	}

	if len(skillMap) == 0 {
		return HealthCheckResultMsg{
			Name:    "Required Skills",
			Status:  HealthCheckOK,
			Message: "No skills required",
			Details: details,
		}
	}

	allOK := true
	for name, entry := range skillMap {
		if entry.check == "" {
			details[name] = "no check command"
			continue
		}
		parts := strings.Fields(entry.check)
		cmd := exec.Command(parts[0], parts[1:]...)
		if err := cmd.Run(); err != nil {
			details[name] = "not installed"
			allOK = false
		} else {
			details[name] = "installed"
		}
	}

	if allOK {
		return HealthCheckResultMsg{
			Name:    "Required Skills",
			Status:  HealthCheckOK,
			Message: fmt.Sprintf("All %d skills available", len(skillMap)),
			Details: details,
		}
	}

	return HealthCheckResultMsg{
		Name:    "Required Skills",
		Status:  HealthCheckErr,
		Message: "Some required skills missing",
		Details: details,
	}
}
