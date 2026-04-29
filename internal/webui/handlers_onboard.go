// Package webui — onboarding driver handlers.
//
// SSE event schema (per /onboard/{id}/stream):
//
//	event: message  // onboarding.Service Notify event (kind/message/step_id)
//	event: prompt   // PromptString / PromptChoice question awaiting an answer
//	event: status   // session lifecycle marker (running, …)
//	event: done     // session finished successfully
//	event: error    // session terminated with an error
//
// Each frame's `data:` line is JSON. `id:` is a per-session monotonically
// increasing integer so reconnecting clients can backfill via `Last-Event-ID`.
//
// Form-answer POST shape (`POST /onboard/{id}/answer`):
//
//	Content-Type: application/x-www-form-urlencoded
//	answer=<user reply>&prompt_id=<question id>
//
// `prompt_id` is optional but, when supplied, must match the currently
// pending prompt's ID — otherwise the request is rejected. This guards
// against late-arriving answers from a stale form.
package webui

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/recinq/wave/internal/onboarding"
)

const (
	onboardEventMessage = "message"
	onboardEventPrompt  = "prompt"
	onboardEventStatus  = "status"
	onboardEventDone    = "done"
	onboardEventError   = "error"

	// onboardRingSize bounds the per-session backfill ring buffer. 200 events
	// covers a typical onboarding flow (~20 step events × 10 chunks each).
	onboardRingSize = 200
)

var (
	errOnboardNoPending  = errors.New("no pending prompt")
	errOnboardWrongID    = errors.New("prompt id mismatch")
	errOnboardClosed     = errors.New("session closed")
	errOnboardPromptBusy = errors.New("prompt already pending")
)

// webOnboardEvent is one envelope buffered in a session ring + broadcast to
// SSE subscribers.
type webOnboardEvent struct {
	ID    int64           `json:"id"`
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// webOnboardPromptPayload is the JSON payload published with the `prompt`
// event. Mirrors onboarding.Question plus a Kind discriminator so the form
// can render either a free-text input or a `<select>`.
type webOnboardPromptPayload struct {
	ID       string   `json:"id"`
	Prompt   string   `json:"prompt"`
	Default  string   `json:"default,omitempty"`
	Choices  []string `json:"choices,omitempty"`
	HelpText string   `json:"help_text,omitempty"`
	Kind     string   `json:"kind"` // "string" or "choice"
}

// webOnboardSession holds the in-memory state for one /onboard chat session:
// ring buffer, SSE subscribers, the active prompt slot, and the cancellation
// hook for the goroutine running onboarding.Service.StartSession.
type webOnboardSession struct {
	id        string
	createdAt time.Time

	mu        sync.Mutex
	nextID    int64
	ring      []webOnboardEvent
	subs      map[chan webOnboardEvent]struct{}
	pending   *webOnboardPromptPayload
	pendingCh chan string
	closed    bool

	ctx    context.Context
	cancel context.CancelFunc
}

// webOnboardUI bridges onboarding.UI to the chat session: Notify dumps events
// into the ring buffer + SSE channel; PromptString/PromptChoice publish a
// `prompt` event then block until the matching POST arrives.
type webOnboardUI struct {
	sess *webOnboardSession
}

// Notify pushes an onboarding.Event onto the SSE stream as a `message` event.
func (u *webOnboardUI) Notify(e onboarding.Event) error {
	payload, _ := json.Marshal(struct {
		Kind    string `json:"kind"`
		Message string `json:"message"`
		StepID  string `json:"step_id,omitempty"`
	}{Kind: e.Kind, Message: e.Message, StepID: e.StepID})
	u.sess.publish(onboardEventMessage, payload)
	return nil
}

// PromptString publishes a `prompt` event with kind="string" and blocks
// until the operator POSTs an answer or the session ctx cancels.
func (u *webOnboardUI) PromptString(q onboarding.Question) (string, error) {
	return u.sess.prompt(q, "string")
}

// PromptChoice publishes a `prompt` event with kind="choice" and blocks
// until an answer arrives or ctx cancels.
func (u *webOnboardUI) PromptChoice(q onboarding.Question) (string, error) {
	return u.sess.prompt(q, "choice")
}

// Compile-time assertion: *webOnboardUI satisfies onboarding.UI.
var _ onboarding.UI = (*webOnboardUI)(nil)

// publish appends an event to the ring buffer and fans it out to current
// subscribers. Drops the event for any subscriber whose buffer is full so a
// stuck reader can't stall the producer.
func (s *webOnboardSession) publish(event string, data json.RawMessage) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.nextID++
	ev := webOnboardEvent{ID: s.nextID, Event: event, Data: data}
	s.ring = append(s.ring, ev)
	if len(s.ring) > onboardRingSize {
		s.ring = s.ring[len(s.ring)-onboardRingSize:]
	}
	subs := make([]chan webOnboardEvent, 0, len(s.subs))
	for ch := range s.subs {
		subs = append(subs, ch)
	}
	s.mu.Unlock()
	for _, ch := range subs {
		select {
		case ch <- ev:
		default:
		}
	}
}

// snapshot returns a copy of ring events with id > afterID. Used by SSE
// reconnects to replay missed events without re-running the onboarding flow.
func (s *webOnboardSession) snapshot(afterID int64) []webOnboardEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]webOnboardEvent, 0, len(s.ring))
	for _, ev := range s.ring {
		if ev.ID > afterID {
			out = append(out, ev)
		}
	}
	return out
}

// subscribe returns a new SSE event channel. unsubscribe must be called
// before discarding the returned channel.
func (s *webOnboardSession) subscribe() chan webOnboardEvent {
	ch := make(chan webOnboardEvent, 64)
	s.mu.Lock()
	if s.subs == nil {
		s.subs = make(map[chan webOnboardEvent]struct{})
	}
	s.subs[ch] = struct{}{}
	s.mu.Unlock()
	return ch
}

func (s *webOnboardSession) unsubscribe(ch chan webOnboardEvent) {
	s.mu.Lock()
	if _, ok := s.subs[ch]; ok {
		delete(s.subs, ch)
		close(ch)
	}
	s.mu.Unlock()
}

// prompt registers a pending prompt, publishes the SSE event, and blocks
// until a matching answer arrives or the session ctx is cancelled. Only one
// prompt can be in-flight at a time; concurrent prompts return errOnboardPromptBusy.
func (s *webOnboardSession) prompt(q onboarding.Question, kind string) (string, error) {
	p := webOnboardPromptPayload{
		ID:       q.ID,
		Prompt:   q.Prompt,
		Default:  q.Default,
		Choices:  q.Choices,
		HelpText: q.HelpText,
		Kind:     kind,
	}

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return "", errOnboardClosed
	}
	if s.pending != nil {
		s.mu.Unlock()
		return "", errOnboardPromptBusy
	}
	ch := make(chan string, 1)
	s.pending = &p
	s.pendingCh = ch
	ctx := s.ctx
	s.mu.Unlock()

	payload, _ := json.Marshal(p)
	s.publish(onboardEventPrompt, payload)

	select {
	case ans, ok := <-ch:
		if !ok {
			return "", errOnboardClosed
		}
		return ans, nil
	case <-ctx.Done():
		s.mu.Lock()
		s.pending = nil
		s.pendingCh = nil
		s.mu.Unlock()
		return "", ctx.Err()
	}
}

// deliverAnswer routes a POSTed answer into a pending prompt. Returns
// errOnboardNoPending if no prompt is waiting (409) or errOnboardWrongID if
// the supplied promptID doesn't match the active one.
func (s *webOnboardSession) deliverAnswer(promptID, answer string) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return errOnboardClosed
	}
	if s.pending == nil {
		s.mu.Unlock()
		return errOnboardNoPending
	}
	if promptID != "" && s.pending.ID != promptID {
		s.mu.Unlock()
		return errOnboardWrongID
	}
	ch := s.pendingCh
	s.pending = nil
	s.pendingCh = nil
	s.mu.Unlock()

	select {
	case ch <- answer:
	default:
	}
	return nil
}

// close marks the session as closed, cancels the running goroutine, and
// releases any pending prompt waiter.
func (s *webOnboardSession) close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	cancel := s.cancel
	pendingCh := s.pendingCh
	s.pending = nil
	s.pendingCh = nil
	subs := s.subs
	s.subs = nil
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if pendingCh != nil {
		close(pendingCh)
	}
	for ch := range subs {
		close(ch)
	}
}

// newOnboardSessionID returns a 16-byte hex random ID used as URL slug.
func newOnboardSessionID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// createOnboardSession allocates a session, registers it, and launches the
// onboarding.Service goroutine. The returned session is already running —
// callers redirect the browser to /onboard/{id} immediately.
func (s *Server) createOnboardSession() (*webOnboardSession, error) {
	id, err := newOnboardSessionID()
	if err != nil {
		return nil, fmt.Errorf("session id: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	sess := &webOnboardSession{
		id:        id,
		createdAt: time.Now(),
		ctx:       ctx,
		cancel:    cancel,
	}

	s.onboard.mu.Lock()
	if s.onboard.sessions == nil {
		s.onboard.sessions = make(map[string]*webOnboardSession)
	}
	s.onboard.sessions[id] = sess
	factory := s.onboard.factory
	s.onboard.mu.Unlock()

	if factory == nil {
		factory = func() onboarding.Service { return nil }
	}

	go s.runOnboardSession(sess, factory)
	return sess, nil
}

// runOnboardSession drives onboarding.Service.StartSession on a goroutine
// and translates its outcome into terminal SSE events.
func (s *Server) runOnboardSession(sess *webOnboardSession, factory func() onboarding.Service) {
	svc := factory()
	if svc == nil {
		errPayload, _ := json.Marshal(map[string]string{"error": "onboarding service unavailable"})
		sess.publish(onboardEventError, errPayload)
		return
	}

	statusPayload, _ := json.Marshal(map[string]string{"status": "running"})
	sess.publish(onboardEventStatus, statusPayload)

	projectDir := s.runtime.repoDir
	if projectDir == "" {
		projectDir = "."
	}

	bridge := &webOnboardUI{sess: sess}
	opts := onboarding.StartOptions{
		OutputPath: "wave.yaml",
		UI:         bridge,
	}
	if s.runtime.manifest != nil && s.runtime.manifest.Runtime.WorkspaceRoot != "" {
		opts.Workspace = s.runtime.manifest.Runtime.WorkspaceRoot
	}

	_, err := svc.StartSession(sess.ctx, projectDir, opts)
	if err != nil {
		errPayload, _ := json.Marshal(map[string]string{"error": err.Error()})
		sess.publish(onboardEventError, errPayload)
		return
	}
	donePayload, _ := json.Marshal(map[string]string{"status": "done"})
	sess.publish(onboardEventDone, donePayload)
}

// getOnboardSession returns the session for id, or nil when unknown.
func (s *Server) getOnboardSession(id string) *webOnboardSession {
	s.onboard.mu.Lock()
	defer s.onboard.mu.Unlock()
	if s.onboard.sessions == nil {
		return nil
	}
	return s.onboard.sessions[id]
}

// closeOnboardSession terminates a session and removes it from the registry.
func (s *Server) closeOnboardSession(id string) {
	s.onboard.mu.Lock()
	sess, ok := s.onboard.sessions[id]
	if ok {
		delete(s.onboard.sessions, id)
	}
	s.onboard.mu.Unlock()
	if ok && sess != nil {
		sess.close()
	}
}

// handleOnboardPage handles GET /onboard and GET /onboard/{id} — the chat
// shell. With no id, a new session is created and the browser is redirected
// to the canonical URL so reload works (Service.Resume happens on the SSE
// reconnect, not the page render).
func (s *Server) handleOnboardPage(w http.ResponseWriter, r *http.Request) {
	sessID := r.PathValue("id")
	if sessID == "" {
		sess, err := s.createOnboardSession()
		if err != nil {
			http.Error(w, "failed to start session: "+err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/onboard/"+sess.id, http.StatusFound)
		return
	}

	sess := s.getOnboardSession(sessID)
	if sess == nil {
		http.NotFound(w, r)
		return
	}

	data := struct {
		ActivePage string
		SessionID  string
	}{
		ActivePage: "onboard",
		SessionID:  sess.id,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := s.assets.templates["templates/onboard/index.html"]
	if tmpl == nil {
		http.Error(w, "template missing", http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "templates/layout.html", data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// handleOnboardStream handles GET /onboard/{id}/stream — SSE conversation
// feed. Honors `Last-Event-ID` for backfill on reconnect, then forwards live
// events until the session ends or the client disconnects.
func (s *Server) handleOnboardStream(w http.ResponseWriter, r *http.Request) {
	sessID := r.PathValue("id")
	sess := s.getOnboardSession(sessID)
	if sess == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	fmt.Fprintf(w, "retry: 3000\n\n")
	flusher.Flush()

	var lastID int64
	if h := r.Header.Get("Last-Event-ID"); h != "" {
		if v, err := strconv.ParseInt(h, 10, 64); err == nil {
			lastID = v
		}
	}

	for _, ev := range sess.snapshot(lastID) {
		fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", ev.ID, ev.Event, ev.Data)
	}
	flusher.Flush()

	ch := sess.subscribe()
	defer sess.unsubscribe(ch)

	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

	ctx := r.Context()
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", ev.ID, ev.Event, ev.Data)
			flusher.Flush()
		case <-keepalive.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		case <-ctx.Done():
			return
		}
	}
}

// handleOnboardAnswer handles POST /onboard/{id}/answer — form-encoded reply
// to the active prompt. 404 unknown session, 409 no pending prompt, 400 on
// prompt-id mismatch.
func (s *Server) handleOnboardAnswer(w http.ResponseWriter, r *http.Request) {
	sessID := r.PathValue("id")
	sess := s.getOnboardSession(sessID)
	if sess == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form: "+err.Error(), http.StatusBadRequest)
		return
	}
	answer := r.FormValue("answer")
	promptID := r.FormValue("prompt_id")

	if err := sess.deliverAnswer(promptID, answer); err != nil {
		switch {
		case errors.Is(err, errOnboardNoPending):
			http.Error(w, "no pending prompt", http.StatusConflict)
		case errors.Is(err, errOnboardWrongID):
			http.Error(w, "prompt id mismatch", http.StatusBadRequest)
		case errors.Is(err, errOnboardClosed):
			http.Error(w, "session closed", http.StatusGone)
		default:
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
