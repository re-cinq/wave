package onboarding

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

// BaselineService is the non-interactive Service implementation that wraps
// the existing Greenfield + AssetSet pipeline. It reuses flavour.go and
// metadata.go for detection so the surface stays in lock-step with whatever
// `wave init --yes` already produces today.
//
// Phase 1 (1.2) supplies an InteractiveService that consumes a UI; until
// then BaselineService is the only Service in the binary.
type BaselineService struct {
	mu       sync.Mutex
	sessions map[string]*Session
	stderr   io.Writer
}

// NewBaselineService returns a BaselineService wired with the supplied
// stderr writer. Pass io.Discard in tests that don't care about the boot
// banner.
func NewBaselineService(stderr io.Writer) *BaselineService {
	if stderr == nil {
		stderr = io.Discard
	}
	return &BaselineService{
		sessions: make(map[string]*Session),
		stderr:   stderr,
	}
}

// IsOnboarded reports whether the project at projectDir has the sentinel.
// Empty projectDir defaults to ".".
func (s *BaselineService) IsOnboarded(projectDir string) bool {
	return IsOnboardedAt(projectDir)
}

// StartSession runs the non-interactive Greenfield path synchronously and
// returns the completed Session. The UI on opts is honoured for Notify
// events but PromptString/PromptChoice are never invoked — every question
// resolves to its default.
func (s *BaselineService) StartSession(ctx context.Context, projectDir string, opts StartOptions) (*Session, error) {
	if projectDir == "" {
		projectDir = "."
	}
	id, err := newSessionID()
	if err != nil {
		return nil, fmt.Errorf("session id: %w", err)
	}

	sess := &Session{
		ID:         id,
		ProjectDir: projectDir,
		StartedAt:  time.Now(),
		Status:     SessionRunning,
	}
	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()

	ui := opts.UI
	if ui == nil {
		ui = NoopUI{}
	}

	if err := ctx.Err(); err != nil {
		s.markStatus(id, SessionAborted)
		return sess, err
	}

	_ = ui.Notify(Event{Kind: "info", Message: "starting non-interactive baseline onboarding"})

	assets, flavour, err := Greenfield(GreenfieldOpts{
		Adapter:    opts.Adapter,
		Workspace:  opts.Workspace,
		OutputPath: opts.OutputPath,
		All:        opts.All,
		Stderr:     s.stderr,
	})
	if err != nil {
		s.markStatus(id, SessionFailed)
		return sess, fmt.Errorf("greenfield scaffold: %w", err)
	}

	sess.Flavour = flavour
	sess.Assets = assets

	if err := MarkDoneAt(projectDir); err != nil {
		s.markStatus(id, SessionFailed)
		return sess, fmt.Errorf("mark sentinel: %w", err)
	}

	s.markStatus(id, SessionDone)
	_ = ui.Notify(Event{Kind: "info", Message: "onboarding complete"})
	return sess, nil
}

// Resume returns the existing session for the supplied id. The baseline
// service runs synchronously so Resume only ever returns a snapshot of an
// already-terminal session — it never reanimates a paused flow.
func (s *BaselineService) Resume(ctx context.Context, sessionID string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[sessionID]
	if !ok {
		return nil, errors.New("session not found")
	}
	return sess, nil
}

// Status returns a Status snapshot for the supplied session id.
func (s *BaselineService) Status(sessionID string) (*Status, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[sessionID]
	if !ok {
		return nil, errors.New("session not found")
	}
	return &Status{
		SessionID: sess.ID,
		Status:    sess.Status,
		UpdatedAt: time.Now(),
	}, nil
}

// MarkDone delegates to the package-level helper. Exposed on the interface
// so drivers that don't share the BaselineService instance can still write
// the sentinel via the same code path.
func (s *BaselineService) MarkDone(projectDir string) error {
	return MarkDoneAt(projectDir)
}

func (s *BaselineService) markStatus(id string, status SessionStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ok := s.sessions[id]; ok {
		sess.Status = status
	}
}

// newSessionID returns a 16-byte hex random id. Sessions are short-lived —
// 64 bits would suffice — but 128 bits removes any risk of collision when
// the same project boots multiple onboarding flows in parallel (e.g. CLI
// while webui is also running).
func newSessionID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// Compile-time assertion: BaselineService satisfies Service.
var _ Service = (*BaselineService)(nil)
