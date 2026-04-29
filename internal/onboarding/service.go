package onboarding

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// SentinelFile marks a project as fully onboarded. Both the CLI driver
// (cmd/wave/commands/init.go) and the future webui driver
// (internal/webui/handlers_onboard.go) write this on completion and check
// for it at boot to decide whether to route to the onboarding flow.
//
// Path is relative to the project root: `.agents/.onboarding-done`.
const SentinelFile = ".agents/.onboarding-done"

// Service is the Domain-layer onboarding contract per ADR-016 D2. The CLI
// and webui drivers each implement their own UI but consume this single
// service so the actual onboarding logic — flavour detection, manifest
// scaffolding, sentinel write — lives in one place.
//
// Phase 0 PRE-2 ships the service skeleton + non-interactive baseline.
// Phase 1 (1.2 / 1.3) wires the per-driver UI implementations and replaces
// the legacy RunWizard machinery.
type Service interface {
	// IsOnboarded reports whether the project at projectDir has the
	// SentinelFile written. Empty projectDir defaults to the working
	// directory.
	IsOnboarded(projectDir string) bool

	// StartSession initialises an onboarding session for projectDir. The
	// returned Session is the handle the driver pumps via its UI.
	// Drivers that already produce a manifest non-interactively (e.g.
	// `wave init --yes`) can call StartSession + Resume + MarkDone in a
	// single synchronous sequence.
	StartSession(ctx context.Context, projectDir string, opts StartOptions) (*Session, error)

	// Resume continues an existing session by ID. Used by the webui driver
	// when an SSE stream reconnects after a page reload.
	Resume(ctx context.Context, sessionID string) (*Session, error)

	// Status returns a snapshot of session state without mutating it.
	Status(sessionID string) (*Status, error)

	// MarkDone writes the sentinel file at projectDir/.agents/.onboarding-done.
	// Idempotent — repeated calls leave the sentinel intact.
	MarkDone(projectDir string) error
}

// UI is the per-driver surface the Service uses to ask the operator
// questions and emit progress events. The CLI implements it via terminal
// prompts; the webui implements it via HTTP form POSTs and SSE streams.
//
// Implementations must be safe to call from a non-TTY context — the
// non-interactive baseline supplies a NoopUI that never asks anything and
// surfaces every Question through its default value.
type UI interface {
	PromptString(question Question) (string, error)
	PromptChoice(question Question) (string, error)
	Notify(event Event) error
}

// StartOptions feeds StartSession with driver-supplied configuration. The
// fields mirror the legacy WizardConfig surface but without the ad-hoc
// fields that were specific to the old wizard machinery.
type StartOptions struct {
	// Adapter is the default LLM adapter recorded in the generated
	// manifest. CLI's --adapter flag and the webui form populate this.
	Adapter string
	// Workspace is the workspace root recorded in the manifest's runtime
	// section (e.g. ".agents/workspaces").
	Workspace string
	// OutputPath is where the wave.yaml manifest will be written.
	OutputPath string
	// All toggles whether every embedded pipeline ships in the project,
	// or only the release-ready set.
	All bool
	// UI is the per-driver UI hook. May be nil for non-interactive baseline.
	UI UI
}

// Question is what the Service asks the UI for. Drivers translate it to
// their native widget (terminal prompt, HTML form field, …).
type Question struct {
	ID       string
	Prompt   string
	Default  string
	Choices  []string
	HelpText string
}

// Event is what the Service notifies the UI about during a session.
// Drivers translate to a native progress channel (stdout banner, SSE
// frame, etc).
type Event struct {
	Kind    string // "step_start" | "step_done" | "warning" | "info"
	Message string
	StepID  string
}

// SessionStatus is the lifecycle state of an onboarding session.
type SessionStatus string

const (
	SessionPending  SessionStatus = "pending"
	SessionRunning  SessionStatus = "running"
	SessionDone     SessionStatus = "done"
	SessionAborted  SessionStatus = "aborted"
	SessionFailed   SessionStatus = "failed"
)

// Session is the in-flight onboarding handle returned by StartSession.
// Drivers either pump it synchronously (CLI) or hand the ID to a webui
// SSE stream that pulls Status snapshots.
type Session struct {
	ID         string
	ProjectDir string
	StartedAt  time.Time
	Status     SessionStatus
	Flavour    *FlavourInfo
	// AssetSet is the discovered assets the manifest was built from.
	// Nil until the session reaches SessionRunning.
	Assets *AssetSet
}

// Status is a snapshot returned by Service.Status. Currently mirrors the
// public Session fields; kept as a separate type so future webui-only
// fields (e.g. progress percentage, last-event timestamp) can live here
// without bloating Session.
type Status struct {
	SessionID string
	Status    SessionStatus
	UpdatedAt time.Time
}

// NoopUI is the UI returned when no driver-specific UI is supplied. Every
// PromptString returns the question's Default; every PromptChoice picks
// the first choice (or Default if it appears in Choices); Notify
// discards events.
type NoopUI struct{ Out io.Writer }

func (u NoopUI) PromptString(q Question) (string, error) {
	return q.Default, nil
}
func (u NoopUI) PromptChoice(q Question) (string, error) {
	if q.Default != "" {
		for _, c := range q.Choices {
			if c == q.Default {
				return q.Default, nil
			}
		}
	}
	if len(q.Choices) > 0 {
		return q.Choices[0], nil
	}
	return q.Default, nil
}
func (u NoopUI) Notify(e Event) error {
	if u.Out != nil {
		fmt.Fprintf(u.Out, "[%s] %s\n", e.Kind, e.Message)
	}
	return nil
}

// IsOnboardedAt reports whether a sentinel file exists at the conventional
// path under projectDir. Used by the baseline service and standalone by the
// webui boot path.
func IsOnboardedAt(projectDir string) bool {
	if projectDir == "" {
		projectDir = "."
	}
	_, err := os.Stat(filepath.Join(projectDir, SentinelFile))
	return err == nil
}

// MarkDoneAt writes the sentinel file. Idempotent. The parent directory
// is created if missing.
func MarkDoneAt(projectDir string) error {
	if projectDir == "" {
		projectDir = "."
	}
	sentinel := filepath.Join(projectDir, SentinelFile)
	if err := os.MkdirAll(filepath.Dir(sentinel), 0o755); err != nil {
		return fmt.Errorf("create .agents dir: %w", err)
	}
	if _, err := os.Stat(sentinel); err == nil {
		return nil
	}
	f, err := os.Create(sentinel)
	if err != nil {
		return fmt.Errorf("write sentinel: %w", err)
	}
	return f.Close()
}

// ClearSentinel removes the sentinel file. Used by `wave init --reconfigure`
// so the next boot routes back to the onboarding flow.
func ClearSentinel(projectDir string) error {
	if projectDir == "" {
		projectDir = "."
	}
	err := os.Remove(filepath.Join(projectDir, SentinelFile))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
