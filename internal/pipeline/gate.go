package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/recinq/wave/internal/event"
	"github.com/recinq/wave/internal/forge"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/state"
)

// commandRunner is a function that runs a command and returns its combined output.
// It is a field on GateExecutor so tests can inject a fake implementation.
type commandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

// defaultCommandRunner executes a real subprocess and returns stdout.
func defaultCommandRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

// GateExecutor handles blocking gate steps.
type GateExecutor struct {
	emitter  event.EventEmitter
	store    state.StateStore
	runner   commandRunner // injectable for tests
	timeouts *manifest.Timeouts
}

// NewGateExecutor creates a gate executor.
func NewGateExecutor(emitter event.EventEmitter, store state.StateStore, timeouts *manifest.Timeouts) *GateExecutor {
	return &GateExecutor{
		emitter:  emitter,
		store:    store,
		runner:   defaultCommandRunner,
		timeouts: timeouts,
	}
}

// Execute blocks until the gate condition is met, times out, or context is cancelled.
func (g *GateExecutor) Execute(ctx context.Context, gate *GateConfig, tmplCtx *TemplateContext) error {
	if gate == nil {
		return fmt.Errorf("gate config is nil")
	}

	g.emit(event.Event{
		Timestamp: time.Now(),
		State:     event.StateGateWaiting,
		Message:   fmt.Sprintf("gate: %s — %s", gate.Type, gate.Message),
	})

	switch gate.Type {
	case "approval":
		return g.executeApproval(ctx, gate)
	case "timer":
		return g.executeTimer(ctx, gate)
	case "pr_merge":
		return g.executePRMerge(ctx, gate)
	case "ci_pass":
		return g.executeCIPass(ctx, gate)
	default:
		return fmt.Errorf("unknown gate type: %q", gate.Type)
	}
}

// executeApproval waits for manual approval or auto-approves.
func (g *GateExecutor) executeApproval(ctx context.Context, gate *GateConfig) error {
	if gate.Auto {
		g.emit(event.Event{
			Timestamp: time.Now(),
			State:     event.StateGateResolved,
			Message:   "gate auto-approved",
		})
		return nil
	}

	// Parse timeout
	timeout := g.timeouts.GetGateApproval()
	if gate.Timeout != "" {
		d, err := time.ParseDuration(gate.Timeout)
		if err != nil {
			return fmt.Errorf("invalid gate timeout %q: %w", gate.Timeout, err)
		}
		timeout = d
	}

	// In non-interactive mode, wait for context cancellation or timeout.
	// The TUI or external system is expected to resolve the gate by cancelling
	// the context or using the state store.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(timeout):
		return fmt.Errorf("gate timed out after %s", timeout)
	}
}

// executeTimer waits for a specified duration.
func (g *GateExecutor) executeTimer(ctx context.Context, gate *GateConfig) error {
	if gate.Timeout == "" {
		return fmt.Errorf("timer gate requires a timeout duration")
	}

	d, err := time.ParseDuration(gate.Timeout)
	if err != nil {
		return fmt.Errorf("invalid timer duration %q: %w", gate.Timeout, err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		g.emit(event.Event{
			Timestamp: time.Now(),
			State:     event.StateGateResolved,
			Message:   fmt.Sprintf("timer gate elapsed: %s", d),
		})
		return nil
	}
}

// parsePollGateTiming returns the poll interval and timeout for a poll gate.
func parsePollGateTiming(gate *GateConfig) (interval, timeout time.Duration, err error) {
	interval = manifest.DefaultGatePollInterval
	timeout = manifest.DefaultGatePollTimeout

	if gate.Interval != "" {
		interval, err = time.ParseDuration(gate.Interval)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid gate interval %q: %w", gate.Interval, err)
		}
	}

	if gate.Timeout != "" {
		timeout, err = time.ParseDuration(gate.Timeout)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid gate timeout %q: %w", gate.Timeout, err)
		}
	}

	return interval, timeout, nil
}

// resolveRepo returns the "owner/repo" slug from gate config or git remotes.
func (g *GateExecutor) resolveRepo(ctx context.Context, gate *GateConfig) (string, error) {
	if gate.Repo != "" {
		return gate.Repo, nil
	}

	info, err := forge.DetectFromGitRemotes()
	if err != nil {
		return "", fmt.Errorf("could not detect forge from git remotes: %w", err)
	}

	slug := info.Slug()
	if slug == "" {
		return "", fmt.Errorf("could not determine repo slug from git remotes; set gate.repo in your pipeline")
	}

	return slug, nil
}

// resolveForge returns the detected forge CLI tool and info, preferring the gate's repo if set.
func (g *GateExecutor) resolveForge() (forge.ForgeInfo, error) {
	info, err := forge.DetectFromGitRemotes()
	if err != nil {
		return forge.ForgeInfo{}, fmt.Errorf("could not detect forge: %w", err)
	}
	if info.Type == forge.ForgeUnknown {
		// Default to GitHub when detection fails (most common case).
		info = forge.ForgeInfo{
			Type:       forge.ForgeGitHub,
			CLITool:    "gh",
			PRCommand:  "pr",
			PRTerm:     "Pull Request",
		}
	}
	return info, nil
}

// prMergeStatus is the JSON shape returned by `gh pr view --json merged,state`.
type prMergeStatus struct {
	Merged bool   `json:"merged"`
	State  string `json:"state"` // "open", "closed"
}

// executePRMerge polls until the specified PR is merged or closed without merging.
func (g *GateExecutor) executePRMerge(ctx context.Context, gate *GateConfig) error {
	if gate.Auto {
		g.emit(event.Event{
			Timestamp: time.Now(),
			State:     event.StateGateResolved,
			Message:   "pr_merge gate auto-resolved",
		})
		return nil
	}

	interval, timeout, err := parsePollGateTiming(gate)
	if err != nil {
		return err
	}

	if gate.PRNumber <= 0 {
		return fmt.Errorf("pr_merge gate requires a pr_number > 0")
	}

	repo, err := g.resolveRepo(ctx, gate)
	if err != nil {
		return err
	}

	fi, err := g.resolveForge()
	if err != nil {
		return err
	}

	cli := fi.CLITool
	prCmd := fi.PRCommand
	prNum := fmt.Sprintf("%d", gate.PRNumber)

	deadline := time.After(timeout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("pr_merge gate timed out after %s waiting for PR #%d to merge", timeout, gate.PRNumber)
		case <-time.After(interval):
		}

		out, runErr := g.runner(ctx, cli, prCmd, "view", prNum, "--repo", repo, "--json", "merged,state")
		if runErr != nil {
			// Transient CLI error — log and keep polling.
			g.emit(event.Event{
				Timestamp: time.Now(),
				State:     event.StateGateWaiting,
				Message:   fmt.Sprintf("pr_merge: CLI error polling PR #%d: %v — retrying", gate.PRNumber, runErr),
			})
			continue
		}

		var status prMergeStatus
		if jsonErr := json.Unmarshal(out, &status); jsonErr != nil {
			g.emit(event.Event{
				Timestamp: time.Now(),
				State:     event.StateGateWaiting,
				Message:   fmt.Sprintf("pr_merge: unexpected output from CLI for PR #%d — retrying", gate.PRNumber),
			})
			continue
		}

		if status.Merged {
			g.emit(event.Event{
				Timestamp: time.Now(),
				State:     event.StateGateResolved,
				Message:   fmt.Sprintf("pr_merge gate resolved: PR #%d is merged", gate.PRNumber),
			})
			return nil
		}

		if strings.ToLower(status.State) == "closed" {
			return fmt.Errorf("pr_merge gate failed: PR #%d was closed without merging", gate.PRNumber)
		}

		g.emit(event.Event{
			Timestamp: time.Now(),
			State:     event.StateGateWaiting,
			Message:   fmt.Sprintf("polling pr_merge: PR #%d is %s, not yet merged...", gate.PRNumber, status.State),
		})
	}
}

// ciRunStatus is one entry in the JSON array returned by `gh run list --json status,conclusion`.
type ciRunStatus struct {
	Status     string `json:"status"`     // "completed", "in_progress", "queued", "waiting", ...
	Conclusion string `json:"conclusion"` // "success", "failure", "cancelled", "skipped", ...
}

// executeCIPass polls until CI checks for a branch pass or fail.
func (g *GateExecutor) executeCIPass(ctx context.Context, gate *GateConfig) error {
	if gate.Auto {
		g.emit(event.Event{
			Timestamp: time.Now(),
			State:     event.StateGateResolved,
			Message:   "ci_pass gate auto-resolved",
		})
		return nil
	}

	interval, timeout, err := parsePollGateTiming(gate)
	if err != nil {
		return err
	}

	// Resolve branch: use gate config, fall back to current git branch.
	branch := gate.Branch
	if branch == "" {
		branch, err = getCurrentGitBranch()
		if err != nil {
			return fmt.Errorf("ci_pass gate: could not determine current branch (set gate.branch): %w", err)
		}
	}

	repo, err := g.resolveRepo(ctx, gate)
	if err != nil {
		return err
	}

	fi, err := g.resolveForge()
	if err != nil {
		return err
	}

	cli := fi.CLITool
	deadline := time.After(timeout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("ci_pass gate timed out after %s waiting for CI on branch %q", timeout, branch)
		case <-time.After(interval):
		}

		out, runErr := g.runner(ctx, cli, "run", "list", "--branch", branch, "--repo", repo, "--limit", "5", "--json", "status,conclusion")
		if runErr != nil {
			g.emit(event.Event{
				Timestamp: time.Now(),
				State:     event.StateGateWaiting,
				Message:   fmt.Sprintf("ci_pass: CLI error polling branch %q: %v — retrying", branch, runErr),
			})
			continue
		}

		var runs []ciRunStatus
		if jsonErr := json.Unmarshal(out, &runs); jsonErr != nil {
			g.emit(event.Event{
				Timestamp: time.Now(),
				State:     event.StateGateWaiting,
				Message:   fmt.Sprintf("ci_pass: unexpected output from CLI for branch %q — retrying", branch),
			})
			continue
		}

		if len(runs) == 0 {
			g.emit(event.Event{
				Timestamp: time.Now(),
				State:     event.StateGateWaiting,
				Message:   fmt.Sprintf("ci_pass: no CI runs found for branch %q — retrying", branch),
			})
			continue
		}

		// Evaluate the most recent run (index 0 from --limit 5).
		latest := runs[0]
		switch strings.ToLower(latest.Status) {
		case "completed":
			conclusion := strings.ToLower(latest.Conclusion)
			switch conclusion {
			case "success", "skipped":
				g.emit(event.Event{
					Timestamp: time.Now(),
					State:     event.StateGateResolved,
					Message:   fmt.Sprintf("ci_pass gate resolved: CI on branch %q completed with %q", branch, latest.Conclusion),
				})
				return nil
			case "failure", "cancelled", "timed_out", "action_required":
				return fmt.Errorf("ci_pass gate failed: CI on branch %q completed with %q", branch, latest.Conclusion)
			default:
				// Neutral conclusion (e.g. "neutral") — treat as pass.
				g.emit(event.Event{
					Timestamp: time.Now(),
					State:     event.StateGateResolved,
					Message:   fmt.Sprintf("ci_pass gate resolved: CI on branch %q completed with %q", branch, latest.Conclusion),
				})
				return nil
			}
		default:
			g.emit(event.Event{
				Timestamp: time.Now(),
				State:     event.StateGateWaiting,
				Message:   fmt.Sprintf("polling ci_pass: branch %q run status=%q — waiting for completion...", branch, latest.Status),
			})
		}
	}
}

func (g *GateExecutor) emit(ev event.Event) {
	if g.emitter != nil {
		g.emitter.Emit(ev)
	}
}
