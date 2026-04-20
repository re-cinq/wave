package ontology

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/state"
)

// sentinelPath is the post-merge staleness marker. Only this package may
// read or write it — consumers must go through CheckStaleness.
const sentinelPath = ".agents/.ontology-stale"

// realService is the production Service implementation. It is constructed
// indirectly through New so callers cannot accidentally bypass the feature
// gate.
type realService struct {
	manifest  *manifest.Manifest
	store     state.StateStore
	emitter   event.EventEmitter
	auditSink AuditSink
}

func newRealService(deps Deps) *realService {
	return &realService{
		manifest:  deps.Manifest,
		store:     deps.Store,
		emitter:   deps.Emitter,
		auditSink: deps.AuditSink,
	}
}

func (s *realService) Enabled() bool { return true }

// CheckStaleness returns a warning if the ontology may be out of date. Two
// signals: (1) the post-merge sentinel exists, (2) wave.yaml is older than
// the most recent git commit that touched ontology-relevant files. Reading
// the sentinel clears it.
func (s *realService) CheckStaleness() string {
	if s.manifest == nil || s.manifest.Ontology == nil || len(s.manifest.Ontology.Contexts) == 0 {
		return ""
	}
	if _, err := os.Stat(sentinelPath); err == nil {
		_ = os.Remove(sentinelPath)
		return "ontology may be stale (post-merge changes detected) — run 'wave analyze' to refresh"
	}

	manifestStat, err := os.Stat("wave.yaml")
	if err != nil {
		return ""
	}

	out, err := exec.Command("git", "log", "-1", "--format=%cI", "--",
		"wave.yaml", "internal/defaults/pipelines/", "internal/defaults/personas/").Output()
	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		return ""
	}
	lastCommit, err := time.Parse(time.RFC3339, strings.TrimSpace(string(out)))
	if err != nil {
		return ""
	}
	if lastCommit.After(manifestStat.ModTime()) {
		return "ontology may be stale (wave.yaml older than latest commit) — run 'wave analyze' to refresh"
	}
	return ""
}

// BuildStepSection renders the ontology markdown for the step and emits
// injection/warn events as a side effect.
func (s *realService) BuildStepSection(pipelineID, stepID string, stepContexts []string) string {
	if s.manifest == nil || s.manifest.Ontology == nil {
		return ""
	}

	defined := make(map[string]bool, len(s.manifest.Ontology.Contexts))
	for _, ctx := range s.manifest.Ontology.Contexts {
		defined[ctx.Name] = true
	}

	// Warn on step.Contexts entries that don't exist in the manifest
	if len(stepContexts) > 0 {
		var undefined []string
		for _, name := range stepContexts {
			if !defined[name] {
				undefined = append(undefined, name)
			}
		}
		if len(undefined) > 0 {
			s.logEvent("ONTOLOGY_WARN",
				fmt.Sprintf("pipeline=%s step=%s undefined_contexts=[%s]",
					pipelineID, stepID, strings.Join(undefined, ",")))
			s.emit(event.Event{
				PipelineID: pipelineID,
				StepID:     stepID,
				State:      event.StateOntologyWarn,
				Message:    fmt.Sprintf("undefined_contexts=[%s]", strings.Join(undefined, ",")),
				Timestamp:  time.Now(),
			})
		}
	}

	section := s.manifest.Ontology.RenderMarkdown(stepContexts)
	if section == "" {
		return ""
	}

	injected := stepContexts
	if len(injected) == 0 {
		for _, ctx := range s.manifest.Ontology.Contexts {
			injected = append(injected, ctx.Name)
		}
	}
	totalInvariants := 0
	for _, ctx := range s.manifest.Ontology.Contexts {
		for _, name := range injected {
			if ctx.Name == name {
				totalInvariants += len(ctx.Invariants)
				break
			}
		}
	}
	s.logEvent("ONTOLOGY_INJECT",
		fmt.Sprintf("pipeline=%s step=%s contexts=[%s] invariants=%d",
			pipelineID, stepID, strings.Join(injected, ","), totalInvariants))
	s.emit(event.Event{
		PipelineID: pipelineID,
		StepID:     stepID,
		State:      event.StateOntologyInject,
		Message: fmt.Sprintf("contexts=[%s] invariants=%d",
			strings.Join(injected, ","), totalInvariants),
		Timestamp: time.Now(),
	})
	return section
}

// RecordUsage writes a per-context lineage row for the step. Only explicitly
// declared contexts are recorded — bulk injection inflates stats.
func (s *realService) RecordUsage(runID, stepID string, stepContexts []string, hasContract bool, stepStatus string) {
	if s.store == nil || s.manifest == nil || s.manifest.Ontology == nil || len(s.manifest.Ontology.Contexts) == 0 {
		return
	}
	if len(stepContexts) == 0 {
		return
	}

	var contractPassed *bool
	if hasContract {
		passed := stepStatus == "success"
		contractPassed = &passed
	}

	defined := make(map[string]bool, len(s.manifest.Ontology.Contexts))
	for _, ctx := range s.manifest.Ontology.Contexts {
		defined[ctx.Name] = true
	}

	for _, ctxName := range stepContexts {
		invariantCount := 0
		for _, ctx := range s.manifest.Ontology.Contexts {
			if ctx.Name == ctxName {
				invariantCount = len(ctx.Invariants)
				break
			}
		}
		lineageStatus := stepStatus
		if !defined[ctxName] {
			lineageStatus = "undefined"
		}
		if err := s.store.RecordOntologyUsage(runID, stepID, ctxName,
			invariantCount, lineageStatus, contractPassed); err != nil {
			s.logEvent("ONTOLOGY_TOOL_CALL_ERR",
				fmt.Sprintf("pipeline=%s step=%s context=%s err=%v",
					runID, stepID, ctxName, err))
			continue
		}
		s.logEvent("ONTOLOGY_LINEAGE",
			fmt.Sprintf("pipeline=%s step=%s context=%s status=%s invariants=%d",
				runID, stepID, ctxName, lineageStatus, invariantCount))
		s.emit(event.Event{
			PipelineID: runID,
			StepID:     stepID,
			State:      event.StateOntologyLineage,
			Message: fmt.Sprintf("context=%s status=%s invariants=%d",
				ctxName, lineageStatus, invariantCount),
			Timestamp: time.Now(),
		})
	}
}

func (s *realService) emit(evt event.Event) {
	if s.emitter != nil {
		s.emitter.Emit(evt)
	}
}

func (s *realService) logEvent(kind, body string) {
	if s.auditSink != nil {
		_ = s.auditSink.LogEvent(kind, body)
	}
}
