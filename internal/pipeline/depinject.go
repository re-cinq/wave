package pipeline

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/recinq/wave/internal/event"
)

// ResolvedArtifact describes one artifact produced by a dependency that
// has been located on disk and is ready for injection into a downstream
// step's workspace.
type ResolvedArtifact struct {
	DepStep  string // ID of the dependency step
	Name     string // artifact name as declared in OutputArtifacts
	Path     string // absolute path to the artifact source file
	Type     string // declared artifact type (json/text/markdown/binary)
	Optional bool   // mirrors ArtifactDef.Required (negated)
}

// ResolveDependencyArtifacts inspects step.Dependencies, reads every declared
// upstream OutputArtifact and locates each one on disk by walking, in order:
//
//  1. execution.ArtifactPaths[<dep>:<name>] (in-memory, parent process).
//  2. e.store.GetArtifacts(runID, dep) (DB, resume-safe, cross-process).
//  3. execution.Context.ArtifactPaths[<dep>.<name>] (composition namespaced).
//  4. <dep_workspace>/.agents/artifacts/<dep>/<name> (filesystem fallback).
//
// Returned map is keyed "<dep>:<name>".
//
// Optional artifacts that cannot be located are silently skipped. Required
// artifacts that cannot be located return an error naming both dep and name.
func (e *DefaultPipelineExecutor) ResolveDependencyArtifacts(execution *PipelineExecution, step *Step) (map[string]ResolvedArtifact, error) {
	resolved := make(map[string]ResolvedArtifact)
	if execution == nil || step == nil || len(step.Dependencies) == 0 || execution.Pipeline == nil {
		return resolved, nil
	}

	for _, depID := range step.Dependencies {
		depStep := findStepByID(execution.Pipeline, depID)
		if depStep == nil {
			// Dependency not declared in pipeline — skip silently.
			// DAG validation already errors on undeclared deps.
			continue
		}

		for _, art := range depStep.OutputArtifacts {
			required := art.Required
			path, found := e.locateDepArtifact(execution, depID, art.Name)
			if !found {
				if required {
					return nil, fmt.Errorf("dependency %q output artifact %q not found", depID, art.Name)
				}
				continue
			}
			key := depID + ":" + art.Name
			resolved[key] = ResolvedArtifact{
				DepStep:  depID,
				Name:     art.Name,
				Path:     path,
				Type:     art.Type,
				Optional: !required,
			}
		}
	}

	return resolved, nil
}

// locateDepArtifact walks the four lookup tiers documented on
// ResolveDependencyArtifacts. Returns the located absolute path and true
// when found and the file exists on disk.
func (e *DefaultPipelineExecutor) locateDepArtifact(execution *PipelineExecution, depID, name string) (string, bool) {
	// Tier 1: in-memory ArtifactPaths.
	execution.mu.Lock()
	path, ok := execution.ArtifactPaths[depID+":"+name]
	execution.mu.Unlock()
	if ok && fileExists(path) {
		return path, true
	}

	// Tier 2: DB.
	if e.store != nil && execution.Status != nil && execution.Status.ID != "" {
		if records, err := e.store.GetArtifacts(execution.Status.ID, depID); err == nil {
			for _, rec := range records {
				if rec.Name == name && fileExists(rec.Path) {
					return rec.Path, true
				}
			}
		}
	}

	// Tier 3: composition-namespaced context.
	if execution.Context != nil {
		if p := execution.Context.GetArtifactPath(depID + "." + name); p != "" && fileExists(p) {
			return p, true
		}
		// Some composition writers register under bare name as well.
		if p := execution.Context.GetArtifactPath(name); p != "" && fileExists(p) {
			return p, true
		}
	}

	// Tier 4: filesystem fallback inside the dep's own workspace.
	execution.mu.Lock()
	depWorkspace := execution.WorkspacePaths[depID]
	execution.mu.Unlock()
	if depWorkspace != "" {
		candidates := []string{
			filepath.Join(depWorkspace, ".agents", "artifacts", depID, name),
			filepath.Join(depWorkspace, ".agents", "artifacts", name),
			filepath.Join(depWorkspace, ".agents", "output", name),
		}
		for _, c := range candidates {
			if fileExists(c) {
				return c, true
			}
		}
	}

	return "", false
}

// injectDependencyArtifacts resolves every dependency artifact for the
// given step and copies (symlinks where possible) the resolved files into
// the step's workspace at canonical locations:
//
//	<workspace>/.agents/artifacts/<dep>/<name>           (canonical)
//	<workspace>/.agents/output/<name>                    (back-compat alias)
//
// It also registers the canonical path in execution.Context under the
// "<dep>.<name>" namespace so {{ artifacts.<dep>.<name> }} resolves.
//
// Failures on optional artifacts are warnings; required artifacts that
// cannot be linked or copied propagate as errors.
//
// The returned map is keyed "<dep>:<name>" with Path rewritten to the
// canonical post-injection path (.agents/artifacts/<dep>/<name>) so
// callers can build env-var or template surfaces from it without
// re-resolving. Returns nil for steps with no dependencies.
func (e *DefaultPipelineExecutor) injectDependencyArtifacts(execution *PipelineExecution, step *Step, workspacePath string) (map[string]ResolvedArtifact, error) {
	if execution == nil || step == nil || workspacePath == "" {
		return nil, nil
	}

	resolved, err := e.ResolveDependencyArtifacts(execution, step)
	if err != nil {
		return nil, err
	}
	if len(resolved) == 0 {
		return nil, nil
	}

	pipelineID := ""
	if execution.Status != nil {
		pipelineID = execution.Status.ID
	}

	artifactsRoot := filepath.Join(workspacePath, ".agents", "artifacts")
	outputRoot := filepath.Join(workspacePath, ".agents", "output")
	if err := os.MkdirAll(artifactsRoot, 0755); err != nil {
		return nil, fmt.Errorf("failed to create artifacts dir: %w", err)
	}
	if err := os.MkdirAll(outputRoot, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output dir: %w", err)
	}

	// Track collisions on the back-compat alias path so we can warn but
	// not fail when two deps both produce the same bare artifact name.
	aliasOwners := make(map[string]string)
	canonical := make(map[string]ResolvedArtifact, len(resolved))

	for _, art := range resolved {
		canonicalDir := filepath.Join(artifactsRoot, art.DepStep)
		if err := os.MkdirAll(canonicalDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create canonical dir %q: %w", canonicalDir, err)
		}
		canonicalPath := filepath.Join(canonicalDir, art.Name)

		if err := linkOrCopy(art.Path, canonicalPath); err != nil {
			if art.Optional {
				e.emit(event.Event{
					Timestamp:  time.Now(),
					PipelineID: pipelineID,
					StepID:     step.ID,
					State:      "step_progress",
					Message:    fmt.Sprintf("optional dep artifact %s/%s skipped: %v", art.DepStep, art.Name, err),
				})
				continue
			}
			return nil, fmt.Errorf("failed to inject %s/%s: %w", art.DepStep, art.Name, err)
		}

		// Back-compat alias at .agents/output/<name>. Warn on collision.
		aliasPath := filepath.Join(outputRoot, art.Name)
		if prev, exists := aliasOwners[art.Name]; exists && prev != art.DepStep {
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      "step_progress",
				Message:    fmt.Sprintf("dep artifact name collision on %q: %s vs %s — alias .agents/output/%s won by %s; canonical paths remain unambiguous", art.Name, prev, art.DepStep, art.Name, art.DepStep),
			})
		}
		_ = os.Remove(aliasPath)
		if err := linkOrCopy(canonicalPath, aliasPath); err != nil {
			// Alias failure is non-fatal — canonical path still works.
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      "step_progress",
				Message:    fmt.Sprintf("alias .agents/output/%s could not be created: %v", art.Name, err),
			})
		} else {
			aliasOwners[art.Name] = art.DepStep
		}

		// Register canonical path under both {{ artifacts.<dep>.<name> }}
		// (existing namespace) and {{ deps.<dep>.<name> }} (new, dep-scoped
		// namespace introduced by issue #1452 phase 3).
		if execution.Context != nil {
			execution.Context.SetArtifactPath(art.DepStep+"."+art.Name, canonicalPath)
			execution.Context.SetCustomVariable("deps."+art.DepStep+"."+art.Name, canonicalPath)
		}

		canonical[art.DepStep+":"+art.Name] = ResolvedArtifact{
			DepStep:  art.DepStep,
			Name:     art.Name,
			Path:     canonicalPath,
			Type:     art.Type,
			Optional: art.Optional,
		}
	}

	return canonical, nil
}

// BuildDepEnvVars returns the WAVE_DEP_<DEP>_<NAME> + WAVE_DEPS_DIR env
// entries (KEY=VALUE strings) for the given resolved-dep map and
// workspace. Names are uppercased; non-alphanumerics become underscores.
// Empty map / empty workspace returns an empty slice.
func BuildDepEnvVars(resolved map[string]ResolvedArtifact, workspacePath string) []string {
	if workspacePath == "" {
		return nil
	}
	out := make([]string, 0, len(resolved)+1)
	out = append(out, "WAVE_DEPS_DIR="+filepath.Join(workspacePath, ".agents", "artifacts"))
	for _, art := range resolved {
		out = append(out, "WAVE_DEP_"+envSlug(art.DepStep)+"_"+envSlug(art.Name)+"="+art.Path)
	}
	return out
}

// envSlug uppercases s and replaces every non-alphanumeric byte with `_`,
// producing a token safe for use in environment-variable names.
func envSlug(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			b.WriteByte(c)
		case c >= 'a' && c <= 'z':
			b.WriteByte(c - 32)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

// findStepByID returns the step in p whose ID matches id, or nil.
func findStepByID(p *Pipeline, id string) *Step {
	if p == nil {
		return nil
	}
	for i := range p.Steps {
		if p.Steps[i].ID == id {
			return &p.Steps[i]
		}
	}
	return nil
}

// fileExists reports whether path refers to an existing filesystem entry.
func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

// linkOrCopy attempts to symlink dest → src (cheap, atomic). Falls back to
// a hard copy when the filesystem rejects symlinks (e.g. Windows CI) or
// when src and dest live on filesystems that disagree.
func linkOrCopy(src, dest string) error {
	if src == dest {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	// If dest already exists pointing to src, leave it.
	if existing, err := os.Readlink(dest); err == nil && existing == src {
		return nil
	}
	_ = os.Remove(dest)
	if err := os.Symlink(src, dest); err == nil {
		return nil
	}
	// Fallback: copy.
	srcF, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcF.Close()
	destF, err := os.Create(dest)
	if err != nil {
		return err
	}
	if _, err := io.Copy(destF, srcF); err != nil {
		destF.Close()
		return err
	}
	return destF.Close()
}
