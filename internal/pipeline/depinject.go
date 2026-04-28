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
	if execution == nil || step == nil || execution.Pipeline == nil {
		return resolved, nil
	}

	// Walk both the live deps and any deps stripped by the resume
	// subpipeline rewriter so artifacts from already-completed steps
	// still get auto-injected on a resumed run.
	depList := step.Dependencies
	for _, dep := range step.ResumeOriginalDeps {
		seen := false
		for _, d := range depList {
			if d == dep {
				seen = true
				break
			}
		}
		if !seen {
			depList = append(depList, dep)
		}
	}
	if len(depList) == 0 {
		return resolved, nil
	}

	for _, depID := range depList {
		depStep := findStepByID(execution.Pipeline, depID)
		// depStep may be nil after resume — the resume subpipeline only
		// contains steps from fromStep onwards. Fall through to the
		// implicit-ArtifactPaths scan with an empty declared set so every
		// "<dep>:*" registration still resolves.
		declared := make(map[string]struct{})
		if depStep != nil {
			declared = make(map[string]struct{}, len(depStep.OutputArtifacts))
			for _, art := range depStep.OutputArtifacts {
				declared[art.Name] = struct{}{}
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

		// Implicit fallback: aggregate / iterate / sub_pipeline steps do
		// not declare OutputArtifacts but register their outputs in
		// execution.ArtifactPaths under "<dep>:<name>" or in the DB.
		// Treat anything registered there but not already declared as an
		// optional dep artifact so downstream steps see it via the same
		// canonical injection path.
		depPrefix := depID + ":"
		execution.mu.Lock()
		var implicit []string
		for k := range execution.ArtifactPaths {
			if !strings.HasPrefix(k, depPrefix) {
				continue
			}
			name := k[len(depPrefix):]
			if _, ok := declared[name]; ok {
				continue
			}
			implicit = append(implicit, name)
		}
		execution.mu.Unlock()
		for _, name := range implicit {
			path, found := e.locateDepArtifact(execution, depID, name)
			if !found {
				continue
			}
			resolved[depID+":"+name] = ResolvedArtifact{
				DepStep:  depID,
				Name:     name,
				Path:     path,
				Optional: true,
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

	// Absolutize workspacePath once for "is the src already inside this
	// workspace?" comparisons below. Worktree workspaces are reused
	// across steps on the same branch so the upstream archive often
	// lives inside the same workspace tree, in which case we'd create a
	// no-op self-symlink (worse: a symlink loop with the alias pass).
	absWorkspace := workspacePath
	if !filepath.IsAbs(absWorkspace) {
		if a, err := filepath.Abs(absWorkspace); err == nil {
			absWorkspace = a
		}
	}

	for _, art := range resolved {
		// If the upstream archive is already inside this step's workspace
		// (shared worktree case — synthesize, create-pr both run on the
		// same `branch: pipeline_id`), the file is reachable at its
		// recorded path. Skip both canonical and alias creation; legacy
		// injectArtifacts and direct path references still work because
		// the file already exists where it was archived.
		absArt := art.Path
		if !filepath.IsAbs(absArt) {
			if a, err := filepath.Abs(absArt); err == nil {
				absArt = a
			}
		}
		if strings.HasPrefix(absArt, absWorkspace+string(filepath.Separator)) {
			// Still register canonical-name templates so prompts using
			// {{ artifacts.<dep>.<name> }} or {{ deps.<dep>.<name> }}
			// resolve to the existing path.
			if execution.Context != nil {
				execution.Context.SetArtifactPath(art.DepStep+"."+art.Name, absArt)
				execution.Context.SetCustomVariable("deps."+art.DepStep+"."+art.Name, absArt)
			}
			canonical[art.DepStep+":"+art.Name] = ResolvedArtifact{
				DepStep:  art.DepStep,
				Name:     art.Name,
				Path:     absArt,
				Type:     art.Type,
				Optional: art.Optional,
			}
			continue
		}

		canonicalDir := filepath.Join(artifactsRoot, art.DepStep)
		if err := os.MkdirAll(canonicalDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create canonical dir %q: %w", canonicalDir, err)
		}
		// Filename on disk preserves the source basename (e.g.
		// `pr-context.json`) so scripts that reference the original
		// extension keep working transparently. Logical artifact name
		// (`pr-context`) remains the key for env vars and templates.
		filename := filepath.Base(art.Path)
		if filename == "" || filename == "." || filename == "/" {
			filename = art.Name
		}
		canonicalPath := filepath.Join(canonicalDir, filename)

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

		// Back-compat alias at .agents/output/<filename>. Warn on collision.
		aliasPath := filepath.Join(outputRoot, filename)
		if prev, exists := aliasOwners[filename]; exists && prev != art.DepStep {
			e.emit(event.Event{
				Timestamp:  time.Now(),
				PipelineID: pipelineID,
				StepID:     step.ID,
				State:      "step_progress",
				Message:    fmt.Sprintf("dep artifact name collision on %q: %s vs %s — alias .agents/output/%s won by %s; canonical paths remain unambiguous", filename, prev, art.DepStep, filename, art.DepStep),
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
				Message:    fmt.Sprintf("alias .agents/output/%s could not be created: %v", filename, err),
			})
		} else {
			aliasOwners[filename] = art.DepStep
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
//
// Symlink targets are absolutized first because the dest's parent
// directory differs from the process CWD, and a relative src would
// silently dangle. The absolutize uses os.Getwd at link time and is
// best-effort: if it fails, fall through to copy mode.
func linkOrCopy(src, dest string) error {
	if src == dest {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	absSrc := src
	if !filepath.IsAbs(absSrc) {
		if a, err := filepath.Abs(absSrc); err == nil {
			absSrc = a
		}
	}
	absDest := dest
	if !filepath.IsAbs(absDest) {
		if a, err := filepath.Abs(absDest); err == nil {
			absDest = a
		}
	}
	// Same target after absolutization (e.g. archive path inside a
	// shared worktree equals the canonical injection path). Skip — any
	// symlink we create here would form a loop because os.Remove(dest)
	// would delete the file we are trying to link to.
	if absSrc == absDest {
		return nil
	}
	// If dest already exists pointing to absSrc, leave it.
	if existing, err := os.Readlink(dest); err == nil && existing == absSrc {
		return nil
	}
	_ = os.Remove(dest)
	if err := os.Symlink(absSrc, dest); err == nil {
		return nil
	}
	// Fallback: copy from the absolutized source.
	srcF, err := os.Open(absSrc)
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
