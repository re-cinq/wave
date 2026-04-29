package webui

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/recinq/wave/internal/onboarding"
)

// fakeOnboardingService lets tests script Service.StartSession without
// touching the real Greenfield filesystem flow. The hook is invoked with the
// supplied UI so tests can drive PromptString / Notify directly.
type fakeOnboardingService struct {
	startHook func(ctx context.Context, projectDir string, opts onboarding.StartOptions) (*onboarding.Session, error)
	mu        sync.Mutex
	sessions  map[string]*onboarding.Session
}

func newFakeOnboardingService(hook func(ctx context.Context, projectDir string, opts onboarding.StartOptions) (*onboarding.Session, error)) *fakeOnboardingService {
	return &fakeOnboardingService{startHook: hook, sessions: make(map[string]*onboarding.Session)}
}

func (f *fakeOnboardingService) IsOnboarded(_ string) bool { return false }
func (f *fakeOnboardingService) StartSession(ctx context.Context, projectDir string, opts onboarding.StartOptions) (*onboarding.Session, error) {
	if f.startHook != nil {
		return f.startHook(ctx, projectDir, opts)
	}
	return &onboarding.Session{ID: "fake", ProjectDir: projectDir, Status: onboarding.SessionDone}, nil
}
func (f *fakeOnboardingService) Resume(_ context.Context, sessionID string) (*onboarding.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.sessions[sessionID]
	if !ok {
		return nil, errors.New("not found")
	}
	return s, nil
}
func (f *fakeOnboardingService) Status(_ string) (*onboarding.Status, error) {
	return &onboarding.Status{}, nil
}
func (f *fakeOnboardingService) MarkDone(_ string) error { return nil }

// onboardTestServer is testServer + an in-memory onboard registry seeded
// with a fake onboarding.Service factory.
func onboardTestServer(t *testing.T, factory func() onboarding.Service) *Server {
	t.Helper()
	srv, _ := testServer(t)
	srv.assets.templates["templates/onboard/index.html"] = template.Must(template.New("templates/layout.html").Parse(
		`<!doctype html><html><body><div class="onboard-shell" data-session-id="{{.SessionID}}">chat-shell</div></body></html>`,
	))
	srv.onboard = serverOnboard{
		sessions: make(map[string]*webOnboardSession),
		factory:  factory,
	}
	return srv
}

func newTestOnboardSession(t *testing.T) *webOnboardSession {
	t.Helper()
	id, err := newOnboardSessionID()
	if err != nil {
		t.Fatalf("session id: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &webOnboardSession{
		id:        id,
		createdAt: time.Now(),
		ctx:       ctx,
		cancel:    cancel,
	}
}

func TestWebOnboardSessionPublishAndSnapshot(t *testing.T) {
	sess := newTestOnboardSession(t)
	defer sess.close()

	sess.publish(onboardEventMessage, json.RawMessage(`{"kind":"info","message":"a"}`))
	sess.publish(onboardEventMessage, json.RawMessage(`{"kind":"info","message":"b"}`))

	all := sess.snapshot(0)
	if len(all) != 2 {
		t.Fatalf("snapshot 0: expected 2 events, got %d", len(all))
	}
	if all[0].ID != 1 || all[1].ID != 2 {
		t.Fatalf("snapshot ids: got %d, %d", all[0].ID, all[1].ID)
	}

	after := sess.snapshot(1)
	if len(after) != 1 || after[0].ID != 2 {
		t.Fatalf("snapshot afterID=1: got %+v", after)
	}
}

func TestWebOnboardSessionPromptUnblocksOnAnswer(t *testing.T) {
	sess := newTestOnboardSession(t)
	defer sess.close()

	type result struct {
		ans string
		err error
	}
	resCh := make(chan result, 1)
	go func() {
		ans, err := sess.prompt(onboarding.Question{ID: "name", Prompt: "What"}, "string")
		resCh <- result{ans, err}
	}()

	// Wait for prompt to register.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		sess.mu.Lock()
		ready := sess.pending != nil
		sess.mu.Unlock()
		if ready {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}

	if err := sess.deliverAnswer("name", "wave"); err != nil {
		t.Fatalf("deliverAnswer: %v", err)
	}

	select {
	case r := <-resCh:
		if r.err != nil {
			t.Fatalf("prompt err: %v", r.err)
		}
		if r.ans != "wave" {
			t.Fatalf("answer: got %q, want %q", r.ans, "wave")
		}
	case <-time.After(time.Second):
		t.Fatal("prompt did not unblock within 1s")
	}
}

func TestWebOnboardSessionPromptCtxCancel(t *testing.T) {
	sess := newTestOnboardSession(t)

	resCh := make(chan error, 1)
	go func() {
		_, err := sess.prompt(onboarding.Question{ID: "x", Prompt: "?"}, "string")
		resCh <- err
	}()

	// Wait for pending to register.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		sess.mu.Lock()
		ready := sess.pending != nil
		sess.mu.Unlock()
		if ready {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}

	sess.cancel()

	select {
	case err := <-resCh:
		if err == nil {
			t.Fatal("expected ctx error, got nil")
		}
	case <-time.After(time.Second):
		t.Fatal("prompt did not unblock on ctx cancel")
	}
}

func TestWebOnboardSessionDeliverAnswerNoPending(t *testing.T) {
	sess := newTestOnboardSession(t)
	defer sess.close()

	if err := sess.deliverAnswer("", "x"); !errors.Is(err, errOnboardNoPending) {
		t.Fatalf("expected errOnboardNoPending, got %v", err)
	}
}

func TestHandleOnboardPageCreatesAndRedirects(t *testing.T) {
	srv := onboardTestServer(t, func() onboarding.Service {
		return newFakeOnboardingService(func(ctx context.Context, _ string, _ onboarding.StartOptions) (*onboarding.Session, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/onboard", nil)
	rec := httptest.NewRecorder()
	srv.handleOnboardPage(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, "/onboard/") {
		t.Fatalf("expected /onboard/<id> redirect, got %q", loc)
	}

	id := strings.TrimPrefix(loc, "/onboard/")
	if srv.getOnboardSession(id) == nil {
		t.Fatalf("session %s not registered", id)
	}
	srv.closeOnboardSession(id)
}

func TestHandleOnboardPageRendersForExistingSession(t *testing.T) {
	srv := onboardTestServer(t, func() onboarding.Service {
		return newFakeOnboardingService(func(ctx context.Context, _ string, _ onboarding.StartOptions) (*onboarding.Session, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		})
	})
	sess, err := srv.createOnboardSession()
	if err != nil {
		t.Fatalf("createOnboardSession: %v", err)
	}
	defer srv.closeOnboardSession(sess.id)

	req := httptest.NewRequest(http.MethodGet, "/onboard/"+sess.id, nil)
	req.SetPathValue("id", sess.id)
	rec := httptest.NewRecorder()
	srv.handleOnboardPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "chat-shell") {
		t.Fatalf("expected chat shell markup, got %q", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), sess.id) {
		t.Fatalf("session id not echoed in body")
	}
}

func TestHandleOnboardPageUnknownSession(t *testing.T) {
	srv := onboardTestServer(t, func() onboarding.Service { return nil })

	req := httptest.NewRequest(http.MethodGet, "/onboard/missing", nil)
	req.SetPathValue("id", "missing")
	rec := httptest.NewRecorder()
	srv.handleOnboardPage(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleOnboardStreamReplaysRingBuffer(t *testing.T) {
	srv := onboardTestServer(t, func() onboarding.Service { return nil })
	sess := newTestOnboardSession(t)
	srv.onboard.sessions[sess.id] = sess
	defer sess.close()

	sess.publish(onboardEventMessage, json.RawMessage(`{"kind":"info","message":"hello"}`))
	sess.publish(onboardEventMessage, json.RawMessage(`{"kind":"info","message":"world"}`))

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/onboard/"+sess.id+"/stream", nil).WithContext(ctx)
	req.SetPathValue("id", sess.id)
	req.Header.Set("Last-Event-ID", "1")
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		srv.handleOnboardStream(rec, req)
		close(done)
	}()

	// Give the handler time to flush the backfill, then close.
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("stream did not close on ctx cancel")
	}

	body := rec.Body.String()
	if rec.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatalf("content-type: got %q", rec.Header().Get("Content-Type"))
	}
	if strings.Contains(body, `"hello"`) {
		t.Fatalf("hello (id=1) should be skipped via Last-Event-ID; body=%s", body)
	}
	if !strings.Contains(body, `"world"`) {
		t.Fatalf("world should be replayed; body=%s", body)
	}
	if !strings.Contains(body, "id: 2") {
		t.Fatalf("expected id: 2 in stream; body=%s", body)
	}
}

func TestHandleOnboardStreamUnknownSession(t *testing.T) {
	srv := onboardTestServer(t, func() onboarding.Service { return nil })

	req := httptest.NewRequest(http.MethodGet, "/onboard/missing/stream", nil)
	req.SetPathValue("id", "missing")
	rec := httptest.NewRecorder()
	srv.handleOnboardStream(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleOnboardAnswerNoPending(t *testing.T) {
	srv := onboardTestServer(t, func() onboarding.Service { return nil })
	sess := newTestOnboardSession(t)
	srv.onboard.sessions[sess.id] = sess
	defer sess.close()

	form := strings.NewReader("answer=foo")
	req := httptest.NewRequest(http.MethodPost, "/onboard/"+sess.id+"/answer", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", sess.id)
	rec := httptest.NewRecorder()
	srv.handleOnboardAnswer(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

func TestHandleOnboardAnswerUnknownSession(t *testing.T) {
	srv := onboardTestServer(t, func() onboarding.Service { return nil })

	req := httptest.NewRequest(http.MethodPost, "/onboard/missing/answer", strings.NewReader("answer=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", "missing")
	rec := httptest.NewRecorder()
	srv.handleOnboardAnswer(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleOnboardAnswerSuccess(t *testing.T) {
	srv := onboardTestServer(t, func() onboarding.Service { return nil })
	sess := newTestOnboardSession(t)
	srv.onboard.sessions[sess.id] = sess
	defer sess.close()

	resCh := make(chan string, 1)
	go func() {
		ans, err := sess.prompt(onboarding.Question{ID: "name", Prompt: "?"}, "string")
		if err != nil {
			t.Errorf("prompt: %v", err)
			resCh <- ""
			return
		}
		resCh <- ans
	}()

	// Wait for prompt to register.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		sess.mu.Lock()
		ready := sess.pending != nil
		sess.mu.Unlock()
		if ready {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}

	form := strings.NewReader("answer=hello&prompt_id=name")
	req := httptest.NewRequest(http.MethodPost, "/onboard/"+sess.id+"/answer", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", sess.id)
	rec := httptest.NewRecorder()
	srv.handleOnboardAnswer(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d (body=%s)", rec.Code, rec.Body.String())
	}

	select {
	case ans := <-resCh:
		if ans != "hello" {
			t.Fatalf("answer: got %q want %q", ans, "hello")
		}
	case <-time.After(time.Second):
		t.Fatal("prompt did not receive answer")
	}
}

func TestHandleOnboardAnswerWrongID(t *testing.T) {
	srv := onboardTestServer(t, func() onboarding.Service { return nil })
	sess := newTestOnboardSession(t)
	srv.onboard.sessions[sess.id] = sess
	defer sess.close()

	go func() {
		_, _ = sess.prompt(onboarding.Question{ID: "name", Prompt: "?"}, "string")
	}()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		sess.mu.Lock()
		ready := sess.pending != nil
		sess.mu.Unlock()
		if ready {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}

	form := strings.NewReader("answer=x&prompt_id=other")
	req := httptest.NewRequest(http.MethodPost, "/onboard/"+sess.id+"/answer", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", sess.id)
	rec := httptest.NewRecorder()
	srv.handleOnboardAnswer(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on prompt id mismatch, got %d", rec.Code)
	}
}
