package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// httpResponse is the expected JSON response from HTTP hooks.
type httpResponse struct {
	OK     bool   `json:"ok"`
	Reason string `json:"reason,omitempty"`
	Action string `json:"action,omitempty"`
}

// executeHTTP sends a POST request with the hook event as JSON.
func executeHTTP(ctx context.Context, hook *LifecycleHookDef, evt HookEvent) HookResult {
	timeout := hook.GetTimeout()

	body, err := json.Marshal(evt)
	if err != nil {
		return HookResult{
			HookName: hook.Name,
			Decision: DecisionBlock,
			Reason:   fmt.Sprintf("failed to marshal event: %v", err),
			Err:      err,
		}
	}

	url := os.ExpandEnv(hook.URL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return HookResult{
			HookName: hook.Name,
			Decision: DecisionBlock,
			Reason:   fmt.Sprintf("failed to create request: %v", err),
			Err:      err,
		}
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return HookResult{
			HookName: hook.Name,
			Decision: DecisionBlock,
			Reason:   fmt.Sprintf("HTTP request failed: %v", err),
			Err:      err,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return HookResult{
			HookName: hook.Name,
			Decision: DecisionBlock,
			Reason:   fmt.Sprintf("HTTP hook returned status %d", resp.StatusCode),
			Err:      fmt.Errorf("non-2xx status: %d", resp.StatusCode),
		}
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return HookResult{
			HookName: hook.Name,
			Decision: DecisionBlock,
			Reason:   fmt.Sprintf("failed to read response: %v", err),
			Err:      err,
		}
	}

	var hookResp httpResponse
	if err := json.Unmarshal(respBody, &hookResp); err != nil {
		return HookResult{
			HookName: hook.Name,
			Decision: DecisionBlock,
			Reason:   fmt.Sprintf("failed to parse response: %v", err),
			Err:      err,
		}
	}

	if hookResp.Action == "skip" {
		return HookResult{
			HookName: hook.Name,
			Decision: DecisionSkip,
			Reason:   hookResp.Reason,
		}
	}

	if !hookResp.OK {
		return HookResult{
			HookName: hook.Name,
			Decision: DecisionBlock,
			Reason:   hookResp.Reason,
		}
	}

	return HookResult{
		HookName: hook.Name,
		Decision: DecisionProceed,
	}
}
