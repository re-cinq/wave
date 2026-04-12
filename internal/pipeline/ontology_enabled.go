//go:build ontology

package pipeline

import (
	"fmt"
	"strings"
	"time"

	"github.com/recinq/wave/internal/event"
)

// buildOntologySection renders ontology contexts into a markdown section for
// AGENTS.md injection. Only compiled when built with the "ontology" tag.
func (e *DefaultPipelineExecutor) buildOntologySection(execution *PipelineExecution, step *Step, pipelineID string) string {
	if execution.Manifest.Ontology == nil {
		return ""
	}

	// Build set of defined context names for undefined-reference detection
	definedContexts := make(map[string]bool, len(execution.Manifest.Ontology.Contexts))
	for _, ctx := range execution.Manifest.Ontology.Contexts {
		definedContexts[ctx.Name] = true
	}

	// Warn on any step.Contexts entries that don't exist in the manifest
	if len(step.Contexts) > 0 {
		var undefinedContexts []string
		for _, name := range step.Contexts {
			if !definedContexts[name] {
				undefinedContexts = append(undefinedContexts, name)
			}
		}
		if len(undefinedContexts) > 0 {
			if e.logger != nil {
				_ = e.logger.LogOntologyWarn(pipelineID, step.ID, undefinedContexts)
			}
			e.emit(event.Event{
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      event.StateOntologyWarn,
				Message:    fmt.Sprintf("undefined_contexts=[%s]", strings.Join(undefinedContexts, ",")),
				Timestamp:  time.Now(),
			})
		}
	}

	ontologySection := execution.Manifest.Ontology.RenderMarkdown(step.Contexts)
	if ontologySection != "" {
		injected := step.Contexts
		if len(injected) == 0 {
			for _, ctx := range execution.Manifest.Ontology.Contexts {
				injected = append(injected, ctx.Name)
			}
		}
		totalInvariants := 0
		for _, ctx := range execution.Manifest.Ontology.Contexts {
			for _, name := range injected {
				if ctx.Name == name {
					totalInvariants += len(ctx.Invariants)
					break
				}
			}
		}
		if e.logger != nil {
			_ = e.logger.LogOntologyInject(pipelineID, step.ID, injected, totalInvariants)
		}
		e.emit(event.Event{
			PipelineID: pipelineID,
			StepID:     step.ID,
			State:      event.StateOntologyInject,
			Message:    fmt.Sprintf("contexts=[%s] invariants=%d", strings.Join(injected, ","), totalInvariants),
			Timestamp:  time.Now(),
		})
	}
	return ontologySection
}

// recordOntologyUsage records ontology context usage for a step after its final status is determined.
// This enables decision lineage tracking across pipeline runs.
func (e *DefaultPipelineExecutor) recordOntologyUsage(execution *PipelineExecution, step *Step, stepStatus string) {
	if e.store == nil || execution.Manifest.Ontology == nil || len(execution.Manifest.Ontology.Contexts) == 0 {
		return
	}

	// Determine which contexts were injected: if step declares contexts, only those;
	// otherwise all contexts are injected (RenderMarkdown with empty filter passes all).
	// Only record targeted (explicitly declared) contexts — bulk injection inflates stats.
	injectedContexts := step.Contexts
	if len(injectedContexts) == 0 {
		return
	}

	// Determine contract pass/fail: nil if no contract, true/false based on step outcome
	var contractPassed *bool
	if step.Handover.Contract.Type != "" {
		passed := stepStatus == "success"
		contractPassed = &passed
	}

	// Build set of defined context names for lineage status
	definedCtx := make(map[string]bool, len(execution.Manifest.Ontology.Contexts))
	for _, ctx := range execution.Manifest.Ontology.Contexts {
		definedCtx[ctx.Name] = true
	}

	for _, ctxName := range injectedContexts {
		invariantCount := 0
		for _, ctx := range execution.Manifest.Ontology.Contexts {
			if ctx.Name == ctxName {
				invariantCount = len(ctx.Invariants)
				break
			}
		}
		lineageStatus := stepStatus
		if !definedCtx[ctxName] {
			lineageStatus = "undefined"
		}
		if err := e.store.RecordOntologyUsage(
			execution.Status.ID, step.ID, ctxName,
			invariantCount, lineageStatus, contractPassed,
		); err != nil {
			if e.logger != nil {
				_ = e.logger.LogToolCall(execution.Status.ID, step.ID, "recordOntologyUsage",
					fmt.Sprintf("context=%s err=%v", ctxName, err))
			}
		} else {
			if e.logger != nil {
				_ = e.logger.LogOntologyLineage(execution.Status.ID, step.ID, ctxName, lineageStatus, invariantCount)
			}
			e.emit(event.Event{
				PipelineID: execution.Status.ID,
				StepID:     step.ID,
				State:      event.StateOntologyLineage,
				Message:    fmt.Sprintf("context=%s status=%s invariants=%d", ctxName, lineageStatus, invariantCount),
				Timestamp:  time.Now(),
			})
		}
	}
}
