package onboarding

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

	res, err := Greenfield(GreenfieldOpts{
		Adapter:    opts.Adapter,
		Model:      opts.Model,
		Workspace:  opts.Workspace,
		OutputPath: opts.OutputPath,
		All:        opts.All,
		Commands:   opts.Commands,
		Stderr:     s.stderr,
	})
	if err != nil {
		s.markStatus(id, SessionFailed)
		return sess, fmt.Errorf("greenfield scaffold: %w", err)
	}

	sess.Flavour = res.Flavour
	sess.Assets = res.Assets
	sess.Result = res

	if err := MarkDoneAt(projectDir); err != nil {
		s.markStatus(id, SessionFailed)
		return sess, fmt.Errorf("mark sentinel: %w", err)
	}

	s.markStatus(id, SessionDone)
	_ = ui.Notify(Event{Kind: "info", Message: "onboarding complete"})
	return sess, nil
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
