package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const maxResponseBodySize = 1 << 20

var urlValidator = validateHTTPTarget

type httpResponse struct {
	OK     bool   `json:"ok"`
	Reason string `json:"reason,omitempty"`
	Action string `json:"action,omitempty"`
}

func executeHTTP(ctx context.Context, hook *LifecycleHookDef, evt HookEvent) HookResult {
	timeout := hook.GetTimeout()
	body, err := json.Marshal(evt)
	if err != nil {
		return HookResult{HookName: hook.Name, Decision: DecisionBlock, Reason: fmt.Sprintf("failed to marshal event: %v", err), Err: err}
	}
	hookURL := os.ExpandEnv(hook.URL)
	if err := urlValidator(hookURL); err != nil {
		return HookResult{HookName: hook.Name, Decision: DecisionBlock, Reason: fmt.Sprintf("blocked URL: %v", err), Err: err}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hookURL, bytes.NewReader(body))
	if err != nil {
		return HookResult{HookName: hook.Name, Decision: DecisionBlock, Reason: fmt.Sprintf("failed to create request: %v", err), Err: err}
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return HookResult{HookName: hook.Name, Decision: DecisionBlock, Reason: fmt.Sprintf("HTTP request failed: %v", err), Err: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return HookResult{HookName: hook.Name, Decision: DecisionBlock, Reason: fmt.Sprintf("HTTP hook returned status %d", resp.StatusCode), Err: fmt.Errorf("non-2xx status: %d", resp.StatusCode)}
	}
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
	if err != nil {
		return HookResult{HookName: hook.Name, Decision: DecisionBlock, Reason: fmt.Sprintf("failed to read response: %v", err), Err: err}
	}
	var hookResp httpResponse
	if err := json.Unmarshal(respBody, &hookResp); err != nil {
		return HookResult{HookName: hook.Name, Decision: DecisionBlock, Reason: fmt.Sprintf("failed to parse response: %v", err), Err: err}
	}
	if hookResp.Action == "skip" {
		return HookResult{HookName: hook.Name, Decision: DecisionSkip, Reason: hookResp.Reason}
	}
	if !hookResp.OK {
		return HookResult{HookName: hook.Name, Decision: DecisionBlock, Reason: hookResp.Reason}
	}
	return HookResult{HookName: hook.Name, Decision: DecisionProceed}
}

func validateHTTPTarget(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("URL has no host")
	}
	lower := strings.ToLower(host)
	if lower == "localhost" || lower == "localhost." {
		return fmt.Errorf("localhost URLs are not allowed")
	}
	ips, err := net.LookupHost(host)
	if err != nil {
		ip := net.ParseIP(host)
		if ip != nil {
			if isBlockedIP(ip) {
				return fmt.Errorf("URL resolves to blocked address %s", ip)
			}
			return nil
		}
		return fmt.Errorf("cannot resolve host %q: %w", host, err)
	}
	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip != nil && isBlockedIP(ip) {
			return fmt.Errorf("URL resolves to blocked address %s", ip)
		}
	}
	return nil
}

func isBlockedIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}
