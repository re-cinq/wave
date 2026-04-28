package pipeline

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/recinq/wave/internal/event"
)

func (e *DefaultPipelineExecutor) resolveWorkspaceStepRefs(tmpl string, execution *PipelineExecution) (string, error) {
	var resolveErr error

	result := templatePattern.ReplaceAllStringFunc(tmpl, func(match string) string {
		if resolveErr != nil {
			return match
		}

		expr := strings.TrimSpace(match[2 : len(match)-2])

		// Only handle {{ steps.* }} references here.
		if !strings.HasPrefix(expr, "steps.") {
			return match
		}

		// steps.STEP_ID.artifacts.ARTIFACT_NAME[.JSON_PATH]
		// steps.STEP_ID.output[.JSON_PATH]
		rest := expr[len("steps."):]
		parts := strings.SplitN(rest, ".", 3) // [STEP_ID, "artifacts"|"output", rest]
		if len(parts) < 2 {
			resolveErr = fmt.Errorf("workspace template %q: expected steps.<step-id>.artifacts.<name> or steps.<step-id>.output.<field>", match)
			return match
		}

		stepID := parts[0]
		segment := parts[1]

		execution.mu.Lock()
		artifactsCopy := make(map[string]string, len(execution.ArtifactPaths))
		for k, v := range execution.ArtifactPaths {
			artifactsCopy[k] = v
		}
		execution.mu.Unlock()

		switch segment {
		case "artifacts":
			// steps.STEP_ID.artifacts.ARTIFACT_NAME[.JSON_PATH]
			if len(parts) < 3 {
				resolveErr = fmt.Errorf("workspace template %q: missing artifact name after 'artifacts'", match)
				return match
			}
			// parts[2] = "ARTIFACT_NAME" or "ARTIFACT_NAME.json.path"
			artAndPath := parts[2]
			dotIdx := strings.Index(artAndPath, ".")
			var artifactName, jsonPath string
			if dotIdx == -1 {
				artifactName = artAndPath
				jsonPath = ""
			} else {
				artifactName = artAndPath[:dotIdx]
				jsonPath = artAndPath[dotIdx+1:]
			}

			key := stepID + ":" + artifactName
			artPath, ok := artifactsCopy[key]
			if !ok {
				resolveErr = fmt.Errorf("workspace template %q: artifact %q from step %q not found (step may not have completed yet)", match, artifactName, stepID)
				return match
			}

			data, err := os.ReadFile(artPath)
			if err != nil {
				resolveErr = fmt.Errorf("workspace template %q: failed to read artifact %q: %w", match, artPath, err)
				return match
			}

			if jsonPath == "" {
				return strings.TrimSpace(string(data))
			}

			val, err := ExtractJSONPath(data, "."+jsonPath)
			if err != nil {
				resolveErr = fmt.Errorf("workspace template %q: JSON path %q in artifact %q: %w", match, jsonPath, artifactName, err)
				return match
			}
			return val

		case "output":
			// steps.STEP_ID.output[.JSON_PATH]
			// Find the first artifact for this step.
			var artPath string
			for k, v := range artifactsCopy {
				if strings.HasPrefix(k, stepID+":") {
					artPath = v
					break
				}
			}
			if artPath == "" {
				resolveErr = fmt.Errorf("workspace template %q: no output found for step %q (step may not have completed yet)", match, stepID)
				return match
			}

			data, err := os.ReadFile(artPath)
			if err != nil {
				resolveErr = fmt.Errorf("workspace template %q: failed to read output for step %q: %w", match, stepID, err)
				return match
			}

			if len(parts) < 3 {
				return strings.TrimSpace(string(data))
			}

			jsonPath := parts[2]
			val, err := ExtractJSONPath(data, "."+jsonPath)
			if err != nil {
				resolveErr = fmt.Errorf("workspace template %q: JSON path %q in step %q output: %w", match, jsonPath, stepID, err)
				return match
			}
			return val

		default:
			resolveErr = fmt.Errorf("workspace template %q: unknown segment %q (expected 'artifacts' or 'output')", match, segment)
			return match
		}
	})

	if resolveErr != nil {
		return "", resolveErr
	}
	return result, nil
}

// resolveStepOutputRef resolves step output references in template strings.
// It supports two forms:
//
//   - Legacy (ADR-010): {{ stepID.output }} / {{ stepID.output.field }}.
//     Resolves by prefix-scanning execution.ArtifactPaths for any key starting
//     with "<stepID>:" — non-deterministic when a step has multiple outputs.
//     The executor emits an ADR-011 rule-4 deprecation warning when this form
//     resolves successfully.
//
//   - Typed (ADR-011 rule 4): {{ stepID.out.<name> }} / {{ stepID.out.<name>.field }}.
//     Looks up exactly "<stepID>:<name>" in execution.ArtifactPaths. This is
//     deterministic — a single step:name binding, no map scan.
//
// This bridges composition steps (which use TemplateContext-style references)
// with the DAG executor (which stores artifacts in execution.ArtifactPaths).
func (e *DefaultPipelineExecutor) resolveStepOutputRef(tmpl string, execution *PipelineExecution) string {
	return templatePattern.ReplaceAllStringFunc(tmpl, func(match string) string {
		expr := strings.TrimSpace(match[2 : len(match)-2])

		// Must be stepID.output(.field)? OR stepID.out.<name>(.field)?
		parts := strings.SplitN(expr, ".", 4)
		if len(parts) < 2 {
			return match
		}

		stepID := parts[0]

		switch parts[1] {
		case "out":
			// Typed named-output addressing — ADR-011 rule 4.
			// {{ stepID.out.<name> }} or {{ stepID.out.<name>.field }}
			if len(parts) < 3 {
				return match // malformed: stepID.out with no name
			}
			outName := parts[2]
			key := stepID + ":" + outName

			execution.mu.Lock()
			path, ok := execution.ArtifactPaths[key]
			execution.mu.Unlock()
			if !ok {
				return match
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return match
			}

			if len(parts) == 3 {
				return string(data)
			}
			// len(parts) == 4 — field extraction
			val, err := ExtractJSONPath(data, "."+parts[3])
			if err != nil {
				return match
			}
			return val

		case "output":
			// Legacy addressing — ADR-010 / deprecated by ADR-011 rule 4.
			// Gather all artifacts registered for this step. Multiple
			// artifacts can live under the same step prefix when the step
			// is a sub-pipeline composition and its child pipeline_outputs
			// were all propagated.
			execution.mu.Lock()
			candidates := make([]string, 0, 4)
			for key, path := range execution.ArtifactPaths {
				if strings.HasPrefix(key, stepID+":") {
					candidates = append(candidates, path)
				}
			}
			execution.mu.Unlock()

			if len(candidates) == 0 {
				return match // no artifact found
			}

			var resolved string
			// {{ stepID.output }} → full file content from first candidate.
			if len(parts) == 2 {
				data, err := os.ReadFile(candidates[0])
				if err != nil {
					return match
				}
				resolved = string(data)
			} else {
				// {{ stepID.output.field }} — parts[2] holds "field" or
				// "field.subfield..." (SplitN capped at 4, so anything past
				// the third dot is in parts[3]; reassemble for JSON path).
				field := parts[2]
				if len(parts) == 4 && parts[3] != "" {
					field = field + "." + parts[3]
				}
				var val string
				found := false
				for _, p := range candidates {
					data, err := os.ReadFile(p)
					if err != nil {
						continue
					}
					v, err := ExtractJSONPath(data, "."+field)
					if err == nil {
						val = v
						found = true
						break
					}
				}
				if !found {
					return match
				}
				resolved = val
			}

			// Emit ADR-011 rule-4 deprecation warning the first time a
			// legacy reference resolves inside this execution.
			e.warnLegacyStepOutputOnce(execution, stepID)
			return resolved

		default:
			return match
		}
	})
}

// warnLegacyStepOutputOnce emits a single WLP rule-4 deprecation warning per
// (execution, stepID) for legacy `{{ stepID.output }}` references. The
// execution tracks emitted warnings via its Results map under the reserved
// key "__wlp_legacy_output_warnings__" to avoid event spam when the same
// template is resolved many times.
func (e *DefaultPipelineExecutor) warnLegacyStepOutputOnce(execution *PipelineExecution, stepID string) {
	if execution == nil {
		return
	}
	const bucket = "__wlp_legacy_output_warnings__"
	execution.mu.Lock()
	if execution.Results == nil {
		execution.Results = make(map[string]map[string]interface{})
	}
	seen, ok := execution.Results[bucket]
	if !ok {
		seen = make(map[string]interface{})
		execution.Results[bucket] = seen
	}
	if _, already := seen[stepID]; already {
		execution.mu.Unlock()
		return
	}
	seen[stepID] = true
	execution.mu.Unlock()

	e.emit(event.Event{
		Timestamp: time.Now(),
		State:     "warning",
		Message: fmt.Sprintf(
			"deprecated: use {{ %s.out.<name> }} instead of {{ %s.output }} — see ADR-011 rule 4",
			stepID, stepID,
		),
	})
}
