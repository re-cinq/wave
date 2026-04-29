package webui

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/recinq/wave/internal/onboarding"
)

// TestOnboardSSEPostRoundTrip drives the full SSE → POST round trip:
// 1. Register a fake Service that calls PromptString once.
// 2. Subscribe to the SSE stream.
// 3. Wait for the prompt event.
// 4. POST an answer.
// 5. Assert the Service goroutine returns and emits a `done` event.
func TestOnboardSSEPostRoundTrip(t *testing.T) {
	answerReceived := make(chan string, 1)
	svc := newFakeOnboardingService(func(ctx context.Context, _ string, opts onboarding.StartOptions) (*onboarding.Session, error) {
		ans, err := opts.UI.PromptString(onboarding.Question{ID: "project_name", Prompt: "Project name?"})
		if err != nil {
			return nil, err
		}
		answerReceived <- ans
		return &onboarding.Session{ID: "fake", Status: onboarding.SessionDone}, nil
	})
	srv := onboardTestServer(t, func() onboarding.Service { return svc })

	mux := http.NewServeMux()
	mux.HandleFunc("GET /onboard", srv.handleOnboardPage)
	mux.HandleFunc("GET /onboard/{id}", srv.handleOnboardPage)
	mux.HandleFunc("GET /onboard/{id}/stream", srv.handleOnboardStream)
	mux.HandleFunc("POST /onboard/{id}/answer", srv.handleOnboardAnswer)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	// 1. Hit /onboard to create a session and follow the redirect manually.
	client := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Get(ts.URL + "/onboard")
	if err != nil {
		t.Fatalf("GET /onboard: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected 302, got %d", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	sessID := strings.TrimPrefix(loc, "/onboard/")
	if sessID == "" {
		t.Fatalf("missing session id in %q", loc)
	}
	defer srv.closeOnboardSession(sessID)

	// 2. Subscribe to the SSE stream.
	streamCtx, cancelStream := context.WithCancel(context.Background())
	defer cancelStream()
	streamReq, err := http.NewRequestWithContext(streamCtx, http.MethodGet, ts.URL+"/onboard/"+sessID+"/stream", nil)
	if err != nil {
		t.Fatalf("stream req: %v", err)
	}
	streamResp, err := http.DefaultClient.Do(streamReq)
	if err != nil {
		t.Fatalf("stream do: %v", err)
	}
	defer streamResp.Body.Close()
	if streamResp.StatusCode != http.StatusOK {
		t.Fatalf("stream status: %d", streamResp.StatusCode)
	}

	// 3. Read events until `prompt` and capture id.
	scanner := bufio.NewScanner(streamResp.Body)
	scanner.Buffer(make([]byte, 0, 16*1024), 1<<20)
	deadline := time.Now().Add(5 * time.Second)
	var promptID string
	var sawPrompt bool
	var currentEvent, currentData string
	for !sawPrompt && time.Now().Before(deadline) {
		if !scanner.Scan() {
			break
		}
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "event: "):
			currentEvent = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			currentData = strings.TrimPrefix(line, "data: ")
		case line == "":
			if currentEvent == onboardEventPrompt && currentData != "" {
				var p webOnboardPromptPayload
				if err := json.Unmarshal([]byte(currentData), &p); err != nil {
					t.Fatalf("decode prompt: %v (data=%s)", err, currentData)
				}
				promptID = p.ID
				sawPrompt = true
			}
			currentEvent, currentData = "", ""
		}
	}
	if !sawPrompt {
		t.Fatalf("no prompt event received within 2s")
	}
	if promptID != "project_name" {
		t.Fatalf("prompt id: got %q want project_name", promptID)
	}

	// 4. POST the answer.
	form := url.Values{}
	form.Set("answer", "wave")
	form.Set("prompt_id", promptID)
	answerResp, err := http.PostForm(ts.URL+"/onboard/"+sessID+"/answer", form)
	if err != nil {
		t.Fatalf("post answer: %v", err)
	}
	answerResp.Body.Close()
	if answerResp.StatusCode != http.StatusNoContent {
		t.Fatalf("answer status: %d", answerResp.StatusCode)
	}

	// 5. Service goroutine sees the answer.
	select {
	case ans := <-answerReceived:
		if ans != "wave" {
			t.Fatalf("Service got %q, want wave", ans)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Service did not receive answer")
	}

	// Drain a few more events looking for `done`.
	doneSeen := make(chan struct{})
	go func() {
		var ev string
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "event: ") {
				ev = strings.TrimPrefix(line, "event: ")
			}
			if line == "" && ev == onboardEventDone {
				close(doneSeen)
				return
			}
		}
	}()
	select {
	case <-doneSeen:
	case <-time.After(5 * time.Second):
		t.Fatal("done event not observed within 2s")
	}
}
