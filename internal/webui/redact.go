//go:build webui

package webui

import (
	"regexp"
)

var credentialPatterns = []*regexp.Regexp{
	regexp.MustCompile(`AKIA[A-Z0-9]{16}`),                                    // AWS access keys
	regexp.MustCompile(`(?i)(aws_secret_access_key|secret_key)\s*[=:]\s*\S+`), // AWS secret keys
	regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}`),                                 // OpenAI/Anthropic keys
	regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`),                                 // GitHub PATs
	regexp.MustCompile(`gho_[a-zA-Z0-9]{36}`),                                 // GitHub OAuth tokens
	regexp.MustCompile(`github_pat_[a-zA-Z0-9_]{82}`),                         // GitHub fine-grained PATs
	regexp.MustCompile(`(?i)password\s*[=:]\s*\S+`),                            // Inline passwords
	regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9._\-]{20,}`),                   // Bearer tokens
	regexp.MustCompile(`(?i)(api[_-]?key|apikey)\s*[=:]\s*\S+`),               // Generic API keys
	regexp.MustCompile(`(?i)(token)\s*[=:]\s*[a-zA-Z0-9._\-]{20,}`),           // Generic tokens
}

const redactedPlaceholder = "[REDACTED]"

// RedactCredentials replaces known credential patterns in content with a redaction placeholder.
func RedactCredentials(content string) string {
	result := content
	for _, pattern := range credentialPatterns {
		result = pattern.ReplaceAllString(result, redactedPlaceholder)
	}
	return result
}
