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

// blockResult builds a blocking HookResult with a formatted reason and cause.
func blockResult(hookName, reason string, err error) HookResult {
	return HookResult{HookName: hookName, Decision: DecisionBlock, Reason: reason, Err: err}
}

func executeHTTP(ctx context.Context, hook *LifecycleHookDef, evt HookEvent) HookResult {
	timeout := hook.GetTimeout()
	body, err := json.Marshal(evt)
	if err != nil {
		return blockResult(hook.Name, fmt.Sprintf("failed to marshal event: %v", err), err)
	}
	hookURL := os.ExpandEnv(hook.URL)
	if err := urlValidator(hookURL); err != nil {
		return blockResult(hook.Name, fmt.Sprintf("blocked URL: %v", err), err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hookURL, bytes.NewReader(body))
	if err != nil {
		return blockResult(hook.Name, fmt.Sprintf("failed to create request: %v", err), err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return blockResult(hook.Name, fmt.Sprintf("HTTP request failed: %v", err), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		statusErr := fmt.Errorf("non-2xx status: %d", resp.StatusCode)
		return blockResult(hook.Name, fmt.Sprintf("HTTP hook returned status %d", resp.StatusCode), statusErr)
	}
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
	if err != nil {
		return blockResult(hook.Name, fmt.Sprintf("failed to read response: %v", err), err)
	}
	var hookResp httpResponse
	if err := json.Unmarshal(respBody, &hookResp); err != nil {
		return blockResult(hook.Name, fmt.Sprintf("failed to parse response: %v", err), err)
	}
	if hookResp.Action == "skip" {
		return HookResult{HookName: hook.Name, Decision: DecisionSkip, Reason: hookResp.Reason}
	}
	if !hookResp.OK {
		return blockResult(hook.Name, hookResp.Reason, nil)
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
