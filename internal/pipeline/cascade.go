package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// StaleStep represents a downstream step that may be consuming stale artifacts.
type StaleStep struct {
	StepID            string
	Reasons           []string // Human-readable reasons why this step is stale
	AffectedArtifacts []string // Artifact keys that are stale (e.g., "specify:spec_status")
}

// CascadeDetector detects stale downstream steps when a step is modified.
// Unlike StaleArtifactDetector in validation.go (prototype-only), this works
// with any pipeline by walking inject_artifacts references.
type CascadeDetector struct{}

// NewCascadeDetector creates a new CascadeDetector.
func NewCascadeDetector() *CascadeDetector {
	return &CascadeDetector{}
}

// artifactEdge represents a dependency from a consuming step to a source step's artifact.
type artifactEdge struct {
	SourceStep   string
	ArtifactName string
}

// buildArtifactGraph builds two maps from the pipeline definition:
//   - produces: stepID -> list of artifact names that the step outputs
//   - consumers: stepID -> list of artifactEdge describing what the step injects
func (d *CascadeDetector) buildArtifactGraph(p *Pipeline) (produces map[string][]string, consumers map[string][]artifactEdge) {
	produces = make(map[string][]string)
	consumers = make(map[string][]artifactEdge)

	for _, step := range p.Steps {
		// What this step produces
		for _, art := range step.OutputArtifacts {
			produces[step.ID] = append(produces[step.ID], art.Name)
		}

		// What this step consumes (from inject_artifacts)
		for _, ref := range step.Memory.InjectArtifacts {
			consumers[step.ID] = append(consumers[step.ID], artifactEdge{
				SourceStep:   ref.Step,
				ArtifactName: ref.Artifact,
			})
		}
	}
	return
}

// GetStaleDownstream finds all steps that directly or transitively consume
// artifacts from the modified step. It walks the pipeline DAG forward in
// topological (pipeline-definition) order.
//
// projectRoot is used by VerifyStaleByMtime for optional mtime filtering;
// this method performs a purely structural (graph-based) detection.
func (d *CascadeDetector) GetStaleDownstream(p *Pipeline, modifiedStepID string, projectRoot string) ([]StaleStep, error) {
	// Validate that the modified step exists in the pipeline.
	found := false
	for _, step := range p.Steps {
		if step.ID == modifiedStepID {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("step %q not found in pipeline %q", modifiedStepID, p.Metadata.Name)
	}

	stale := d.walkForward(p, modifiedStepID)

	if projectRoot != "" {
		stale = d.VerifyStaleByMtime(stale, p, projectRoot)
	}

	return stale, nil
}

// walkForward traverses the pipeline steps in definition order and marks any
// step as stale if it consumes an artifact from the modified step or from
// another step that has already been marked stale (transitive staleness).
func (d *CascadeDetector) walkForward(p *Pipeline, modifiedStepID string) []StaleStep {
	_, consumers := d.buildArtifactGraph(p)

	// Track which steps are stale. The modified step itself is the root.
	stale := make(map[string]*StaleStep)
	staleSet := map[string]bool{modifiedStepID: true}

	// Process steps in pipeline order (assumed topologically sorted).
	for _, step := range p.Steps {
		if step.ID == modifiedStepID {
			continue
		}

		for _, edge := range consumers[step.ID] {
			if staleSet[edge.SourceStep] {
				if _, ok := stale[step.ID]; !ok {
					stale[step.ID] = &StaleStep{StepID: step.ID}
				}

				var reason string
				if edge.SourceStep == modifiedStepID {
					reason = fmt.Sprintf("consumes artifact %q from modified step %q", edge.ArtifactName, edge.SourceStep)
				} else {
					reason = fmt.Sprintf("consumes artifact %q from transitively stale step %q", edge.ArtifactName, edge.SourceStep)
				}
				stale[step.ID].Reasons = append(stale[step.ID].Reasons, reason)
				stale[step.ID].AffectedArtifacts = append(stale[step.ID].AffectedArtifacts,
					fmt.Sprintf("%s:%s", edge.SourceStep, edge.ArtifactName))
				staleSet[step.ID] = true
			}
		}
	}

	// Return in pipeline-definition order.
	var result []StaleStep
	for _, step := range p.Steps {
		if s, ok := stale[step.ID]; ok {
			result = append(result, *s)
		}
	}
	return result
}

// VerifyStaleByMtime filters structurally-stale steps by checking whether the
// source step's workspace was actually modified after the consuming step's
// workspace. This reduces false positives: if a downstream step was re-run
// after the upstream modification, its artifacts are already up to date.
func (d *CascadeDetector) VerifyStaleByMtime(staleSteps []StaleStep, p *Pipeline, projectRoot string) []StaleStep {
	var confirmed []StaleStep
	for _, ss := range staleSteps {
		// Get the consuming step's latest workspace mtime.
		wsPath := filepath.Join(projectRoot, ".wave", "workspaces", p.Metadata.Name, ss.StepID)
		wsMtime, err := latestMtime(wsPath)
		if err != nil {
			// Workspace doesn't exist or can't be read -- assume stale.
			confirmed = append(confirmed, ss)
			continue
		}

		// Check each source step's workspace mtime.
		isStale := false
		for _, affected := range ss.AffectedArtifacts {
			parts := splitArtifactKey(affected)
			if len(parts) != 2 {
				continue
			}
			srcWsPath := filepath.Join(projectRoot, ".wave", "workspaces", p.Metadata.Name, parts[0])
			srcMtime, err := latestMtime(srcWsPath)
			if err != nil {
				// Source workspace gone -- assume stale.
				isStale = true
				break
			}
			if srcMtime.After(wsMtime) {
				isStale = true
				break
			}
		}

		if isStale {
			confirmed = append(confirmed, ss)
		}
	}
	return confirmed
}

// latestMtime returns the most recent modification time of any regular file
// under the given directory tree.
func latestMtime(dir string) (time.Time, error) {
	var latest time.Time
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.ModTime().After(latest) {
			latest = info.ModTime()
		}
		return nil
	})
	if err != nil {
		return time.Time{}, err
	}
	return latest, nil
}

// splitArtifactKey splits a "step:artifact" key into its two components.
func splitArtifactKey(key string) []string {
	return strings.SplitN(key, ":", 2)
}

// SelectCascadeTargets filters stale steps to only those whose IDs appear in
// the selectedIDs slice. This is used when the user chooses which stale steps
// to re-run rather than re-running all of them.
func SelectCascadeTargets(stale []StaleStep, selectedIDs []string) []StaleStep {
	selected := make(map[string]bool, len(selectedIDs))
	for _, id := range selectedIDs {
		selected[id] = true
	}
	var result []StaleStep
	for _, s := range stale {
		if selected[s.StepID] {
			result = append(result, s)
		}
	}
	return result
}

// FormatStaleReport generates a human-readable report of stale steps suitable
// for displaying in a terminal or injecting into a chat context.
func FormatStaleReport(stale []StaleStep) string {
	if len(stale) == 0 {
		return "No stale steps detected."
	}

	var b strings.Builder

	b.WriteString(fmt.Sprintf("Cascade analysis: %d downstream step(s) may be stale\n", len(stale)))
	b.WriteString(strings.Repeat("-", 60))
	b.WriteString("\n")

	for i, ss := range stale {
		b.WriteString(fmt.Sprintf("\n%d. Step: %s\n", i+1, ss.StepID))

		b.WriteString("   Reasons:\n")
		for _, reason := range ss.Reasons {
			b.WriteString(fmt.Sprintf("     - %s\n", reason))
		}

		if len(ss.AffectedArtifacts) > 0 {
			b.WriteString("   Affected artifacts:\n")
			for _, art := range ss.AffectedArtifacts {
				b.WriteString(fmt.Sprintf("     - %s\n", art))
			}
		}
	}

	b.WriteString("\n")
	b.WriteString(strings.Repeat("-", 60))
	b.WriteString("\n")
	b.WriteString("Re-run stale steps to bring downstream artifacts up to date.\n")

	return b.String()
}
