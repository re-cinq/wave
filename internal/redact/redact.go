// Package redact provides a single, canonical credential-redaction
// implementation shared by webui, audit, and any other consumer that needs
// to scrub secrets from user-visible text.
//
// Covered secret shapes:
//
//   - AWS access key IDs (AKIA...)
//   - AWS secret keys (aws_secret_access_key=..., secret_key=...)
//   - OpenAI / Anthropic style keys (sk-...)
//   - GitHub PATs (ghp_...), OAuth tokens (gho_...), fine-grained PATs (github_pat_...)
//   - GitLab PATs (glpat-...)
//   - Slack tokens (xoxb-/xoxp-/xoxo-/xoxr-/xoxa-/xoxs-)
//   - Inline passwords (password=..., password:...)
//   - Bearer tokens (Authorization: Bearer ...)
//   - Generic api_key / apikey assignments
//   - Generic long token assignments (token=<20+ chars>)
//   - Generic credential keyword assignments matching
//     API_KEY / TOKEN / SECRET / PASSWORD / CREDENTIAL / AUTH /
//     PRIVATE_KEY / ACCESS_KEY in KEY=VALUE or KEY:VALUE form
//
// All matches are replaced with the literal string "[REDACTED]".
package redact

import (
	"regexp"
	"strings"
)

// Placeholder is the literal string that replaces redacted credentials.
const Placeholder = "[REDACTED]"

// maxRedactSize caps the input size for credential scanning. Running many
// regex patterns on multi-megabyte content is too slow for a request handler.
// Credential tokens are short (<200 chars), so scanning beyond 256 KB is
// wasteful. Inputs larger than this are returned unchanged.
const maxRedactSize = 256 * 1024

// genericKeywords are credential keywords matched in a loose KEY[=:]VALUE
// form. They subsume the historical internal/audit keyword set and catch
// short-value cases the precise patterns deliberately ignore.
var genericKeywords = []string{
	`API[_-]?KEY`,
	`TOKEN`,
	`SECRET`,
	`PASSWORD`,
	`CREDENTIAL`,
	`AUTH`,
	`PRIVATE[_-]?KEY`,
	`ACCESS[_-]?KEY`,
}

var credentialPatterns = func() []*regexp.Regexp {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`AKIA[A-Z0-9]{16}`),                                    // AWS access keys
		regexp.MustCompile(`(?i)(aws_secret_access_key|secret_key)\s*[=:]\s*\S+`), // AWS secret keys
		regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}`),                                 // OpenAI/Anthropic keys
		regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`),                                 // GitHub PATs
		regexp.MustCompile(`gho_[a-zA-Z0-9]{36}`),                                 // GitHub OAuth tokens
		regexp.MustCompile(`github_pat_[a-zA-Z0-9_]{82}`),                         // GitHub fine-grained PATs
		regexp.MustCompile(`(?i)password\s*[=:]\s*\S+`),                           // Inline passwords
		regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9._\-]{20,}`),                   // Bearer tokens
		regexp.MustCompile(`(?i)(api[_-]?key|apikey)\s*[=:]\s*\S+`),               // Generic API keys
		regexp.MustCompile(`(?i)(token)\s*[=:]\s*[a-zA-Z0-9._\-]{20,}`),           // Generic tokens
		regexp.MustCompile(`glpat-[a-zA-Z0-9_\-]{20,}`),                           // GitLab PATs
		regexp.MustCompile(`xox[bporas]-[a-zA-Z0-9\-]{10,}`),                      // Slack tokens
	}
	// Generic keyword fallback (preserves historical audit-package coverage).
	keyword := `(?i)(` + strings.Join(genericKeywords, `|`) + `)[=:]?\s*[\w\-]+`
	patterns = append(patterns, regexp.MustCompile(keyword))
	return patterns
}()

// Redact returns content with all known credential patterns replaced by
// Placeholder. Inputs larger than 256 KB are returned unchanged to bound
// scanning cost; no other transformation is applied.
func Redact(content string) string {
	if len(content) > maxRedactSize {
		return content
	}
	result := content
	for _, pattern := range credentialPatterns {
		result = pattern.ReplaceAllString(result, Placeholder)
	}
	return result
}
