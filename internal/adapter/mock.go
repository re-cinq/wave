package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"time"
)

type MockAdapter struct {
	Config MockConfig
}

type MockConfig struct {
	StdoutJSON     string
	ExitCode       int
	TokensUsed     int
	SimulatedDelay time.Duration
	ShouldFail     bool
	FailError      error
}

type MockOption func(*MockConfig)

func WithStdoutJSON(stdout string) MockOption {
	return func(c *MockConfig) {
		c.StdoutJSON = stdout
	}
}

func WithExitCode(code int) MockOption {
	return func(c *MockConfig) {
		c.ExitCode = code
	}
}

func WithTokensUsed(tokens int) MockOption {
	return func(c *MockConfig) {
		c.TokensUsed = tokens
	}
}

func WithSimulatedDelay(delay time.Duration) MockOption {
	return func(c *MockConfig) {
		c.SimulatedDelay = delay
	}
}

func WithFailure(err error) MockOption {
	return func(c *MockConfig) {
		c.ShouldFail = true
		c.FailError = err
	}
}

func NewMockAdapter(opts ...MockOption) *MockAdapter {
	cfg := MockConfig{
		ExitCode:       0,
		TokensUsed:     0,
		SimulatedDelay: 0,
		ShouldFail:     false,
		StdoutJSON:     "",
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &MockAdapter{Config: cfg}
}

func (m *MockAdapter) Run(ctx context.Context, cfg AdapterRunConfig) (*AdapterResult, error) {
	if m.Config.SimulatedDelay > 0 {
		select {
		case <-time.After(m.Config.SimulatedDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if m.Config.ShouldFail {
		return nil, m.Config.FailError
	}

	stdout := m.Config.StdoutJSON
	if stdout == "" {
		stdout = generateRealisticOutput(cfg)
	}

	tokens := m.Config.TokensUsed
	if tokens == 0 {
		tokens = 2000 + rand.Intn(6000)
	}

	var artifacts []string
	for _, art := range extractArtifactNames(stdout) {
		artifacts = append(artifacts, art)
	}

	return &AdapterResult{
		ExitCode:   m.Config.ExitCode,
		Stdout:     bytes.NewReader([]byte(stdout)),
		TokensUsed: tokens,
		Artifacts:  artifacts,
	}, nil
}

// generateRealisticOutput produces persona-appropriate mock output
func generateRealisticOutput(cfg AdapterRunConfig) string {
	switch cfg.Persona {
	case "navigator":
		return generateNavigatorOutput(cfg)
	case "philosopher":
		return generatePhilosopherOutput(cfg)
	case "craftsman":
		return generateCraftsmanOutput(cfg)
	case "auditor":
		return generateAuditorOutput(cfg)
	case "summarizer":
		return generateSummarizerOutput(cfg)
	default:
		return generateGenericOutput(cfg)
	}
}

func generateNavigatorOutput(cfg AdapterRunConfig) string {
	data := map[string]interface{}{
		"files": []map[string]string{
			{"path": "internal/pipeline/executor.go", "purpose": "Core pipeline execution engine with DAG traversal and step orchestration"},
			{"path": "internal/adapter/adapter.go", "purpose": "Adapter interface and process group runner for LLM CLI wrappers"},
			{"path": "internal/manifest/types.go", "purpose": "Manifest type definitions for adapters, personas, and runtime config"},
			{"path": "internal/workspace/workspace.go", "purpose": "Ephemeral workspace creation with mount support and artifact injection"},
			{"path": "internal/contract/contract.go", "purpose": "Contract validation factory for handover gates between steps"},
			{"path": "cmd/wave/commands/run.go", "purpose": "CLI command handler for pipeline execution with event emission"},
		},
		"patterns": []map[string]string{
			{"name": "Interface-based dependency injection", "description": "All major components use interfaces (AdapterRunner, WorkspaceManager, StateStore) enabling testability and swappable implementations"},
			{"name": "Functional options pattern", "description": "Configuration via option functions (WithEmitter, WithStateStore) for clean builder-style initialization"},
			{"name": "DAG-based execution", "description": "Pipelines use topological sort for dependency resolution with cycle detection"},
			{"name": "Persona-scoped permissions", "description": "Each persona has explicit allowed/denied tool lists enforced at the adapter level"},
		},
		"dependencies": map[string]interface{}{
			"pipeline": []string{"adapter", "manifest", "workspace", "state", "event", "audit"},
			"adapter":  []string{"manifest"},
			"commands": []string{"pipeline", "adapter", "manifest", "event"},
		},
		"impact_areas": []string{
			"Pipeline executor step orchestration",
			"Workspace isolation and artifact flow",
			"Adapter process management and timeout handling",
			"Contract validation at handover boundaries",
			"State persistence for pipeline resumption",
		},
	}
	out, _ := json.MarshalIndent(data, "", "  ")
	return string(out)
}

func generatePhilosopherOutput(cfg AdapterRunConfig) string {
	return `# Feature Specification

## Overview
This specification covers the requested feature based on codebase analysis.

## User Stories

### US-1: Primary User Flow
**As a** developer
**I want to** define multi-step pipelines in YAML
**So that** I can orchestrate AI agents through structured workflows

**Acceptance Criteria:**
- Pipeline YAML is validated at load time
- Steps execute in dependency order (topological sort)
- Each step runs in an isolated workspace under .wave/workspaces/
- Artifacts flow between steps via injection

### US-2: Workspace Isolation
**As a** developer
**I want** each pipeline step to run in its own workspace
**So that** side effects are contained and reproducible

**Acceptance Criteria:**
- Workspaces created at .wave/workspaces/<pipeline>/<step>/
- Source mounts copy project files into workspace
- Artifacts from prior steps injected into artifacts/ subdirectory
- Workspace cleaned up via 'wave clean' command

## Data Model
- Pipeline: kind, metadata, input, steps[]
- Step: id, persona, dependencies, memory, workspace, exec, output_artifacts, handover
- Handover: contract (type, schema, on_failure, max_retries), compaction

## Edge Cases
- Circular dependencies detected at validation time
- Missing persona references caught before execution
- Adapter binary not found fails immediately with clear error
- Interrupted pipelines can resume from last checkpoint

## Testing Strategy
- Unit tests for DAG validation and topological sort
- Integration tests for full pipeline execution with mock adapter
- Contract validation tests with schema fixtures
`
}

func generateCraftsmanOutput(cfg AdapterRunConfig) string {
	return fmt.Sprintf(`## Implementation Report

### Changes Made

**internal/pipeline/executor.go**
- Integrated WorkspaceManager for step workspace creation under .wave/workspaces/
- Added artifact injection between steps via ArtifactPaths tracking
- Wired StateStore for pipeline/step state persistence
- Connected EventEmitter for real-time execution events
- Added AuditLogger for tool call tracing

**internal/adapter/adapter.go**
- Extended AdapterRunConfig with SystemPrompt, Temperature, AllowedTools, DenyTools
- Added OutputFormat field for adapter-specific output modes

**cmd/wave/commands/run.go**
- Initialized all components (WorkspaceManager, StateStore, AuditLogger, EventEmitter)
- Passed components to executor via functional options
- Events emitted during execution, not pre-computed

### Tests Run
` + "```" + `
=== RUN   TestDAGValidation
--- PASS: TestDAGValidation (0.00s)
=== RUN   TestTopologicalSort
--- PASS: TestTopologicalSort (0.00s)
=== RUN   TestPipelineExecution
--- PASS: TestPipelineExecution (0.12s)
=== RUN   TestArtifactInjection
--- PASS: TestArtifactInjection (0.03s)
=== RUN   TestWorkspaceCreation
--- PASS: TestWorkspaceCreation (0.01s)
PASS
ok      github.com/recinq/wave/internal/pipeline   0.16s
` + "```" + `

### Workspace: %s
All changes committed to workspace. Tests passing.
`, cfg.WorkspacePath)
}

func generateAuditorOutput(cfg AdapterRunConfig) string {
	return `## Security & Quality Review

### Summary
Implementation reviewed for OWASP Top 10, code quality, and test coverage.

### Findings

#### MEDIUM: Workspace path traversal
- **File**: internal/workspace/workspace.go:86
- **Issue**: Mount source paths not validated for path traversal (e.g., ../../etc/passwd)
- **Recommendation**: Validate that resolved source paths stay within project root

#### LOW: Error values not checked
- **File**: internal/pipeline/executor.go:463-464
- **Issue**: os.MkdirAll and os.WriteFile return values ignored in writeOutputArtifacts
- **Recommendation**: Log or return errors from artifact write operations

#### LOW: Token estimation is approximate
- **File**: internal/adapter/adapter.go:122
- **Issue**: estimateTokens uses len/4 which is rough for non-English text
- **Recommendation**: Use a proper tokenizer or accept the approximation with a comment

### Positive Observations
- Process group cleanup via SIGKILL prevents orphaned processes
- Credential scrubbing in audit logger catches common secret patterns
- Fresh memory strategy prevents cross-step context leakage
- Contract validation gates prevent bad artifacts from flowing downstream

### Verdict: GO
Implementation is production-ready with the noted improvements as follow-up items.
No critical or high-severity issues found.
`
}

func generateSummarizerOutput(cfg AdapterRunConfig) string {
	return `# Checkpoint Summary

## Objective
Executing pipeline step with workspace isolation and artifact flow.

## Progress
- Pipeline validated and topologically sorted
- Workspaces created under .wave/workspaces/
- Previous step artifacts injected successfully
- Adapter execution completed with expected output

## Key Decisions
- Workspaces use copy-on-mount (not symlinks) for true isolation
- Artifacts flow via filesystem paths tracked in ArtifactPaths map
- State persisted to SQLite for crash recovery

## Current State
Step execution completed. Output artifacts written to workspace.
Contract validation passed where configured.

## Next Steps
- Execute remaining pipeline steps in dependency order
- Run final review step for quality gate
`
}

func generateGenericOutput(cfg AdapterRunConfig) string {
	data := map[string]interface{}{
		"adapter":      cfg.Adapter,
		"persona":      cfg.Persona,
		"workspace":    cfg.WorkspacePath,
		"prompt_len":   len(cfg.Prompt),
		"status":       "completed",
		"tokens_used":  2000 + rand.Intn(4000),
	}
	out, _ := json.MarshalIndent(data, "", "  ")
	return string(out)
}

func extractArtifactNames(stdout string) []string {
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &parsed); err == nil {
		if artifactList, ok := parsed["artifacts"].([]interface{}); ok {
			var arts []string
			for _, a := range artifactList {
				if s, ok := a.(string); ok {
					arts = append(arts, s)
				}
			}
			return arts
		}
	}
	return nil
}

type MockAdapterRegistry struct {
	mu       sync.RWMutex
	adapters map[string]*MockAdapter
}

func NewMockAdapterRegistry() *MockAdapterRegistry {
	return &MockAdapterRegistry{
		adapters: make(map[string]*MockAdapter),
	}
}

func (r *MockAdapterRegistry) Register(name string, adapter *MockAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[name] = adapter
}

func (r *MockAdapterRegistry) Get(name string) *MockAdapter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.adapters[name]
}

func (r *MockAdapterRegistry) CreateRunner(name string) AdapterRunner {
	adapter := r.Get(name)
	if adapter == nil {
		adapter = NewMockAdapter()
	}
	return &registeredRunner{
		registry: r,
		name:     name,
	}
}

type registeredRunner struct {
	registry *MockAdapterRegistry
	name     string
}

func (r *registeredRunner) Run(ctx context.Context, cfg AdapterRunConfig) (*AdapterResult, error) {
	adapter := r.registry.Get(r.name)
	if adapter == nil {
		adapter = NewMockAdapter()
	}
	cfg.Adapter = r.name
	return adapter.Run(ctx, cfg)
}

type SlowReader struct {
	data      []byte
	readPos   int
	chunkSize int
	delay     time.Duration
	mu        sync.Mutex
}

func NewSlowReader(data string, chunkSize int, delay time.Duration) *SlowReader {
	return &SlowReader{
		data:      []byte(data),
		readPos:   0,
		chunkSize: chunkSize,
		delay:     delay,
	}
}

func (r *SlowReader) Read(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.readPos >= len(r.data) {
		return 0, io.EOF
	}

	remaining := len(r.data) - r.readPos
	toRead := r.chunkSize
	if toRead > remaining {
		toRead = remaining
	}
	if toRead > len(p) {
		toRead = len(p)
	}

	time.Sleep(r.delay)

	copy(p, r.data[r.readPos:r.readPos+toRead])
	r.readPos += toRead

	if r.readPos >= len(r.data) {
		return toRead, io.EOF
	}

	return toRead, nil
}
