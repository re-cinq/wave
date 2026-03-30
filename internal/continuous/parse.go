package continuous

import (
	"fmt"
	"strings"
)

// ParseSourceURI parses a source URI string into a SourceConfig.
// Format: "<provider>:<params>" where params are key=value pairs separated by commas.
// For file sources, the format is "file:<path>".
func ParseSourceURI(uri string) (*SourceConfig, error) {
	if uri == "" {
		return nil, fmt.Errorf("source URI cannot be empty")
	}

	parts := strings.SplitN(uri, ":", 2)
	if len(parts) != 2 || parts[1] == "" {
		return nil, fmt.Errorf("invalid source URI %q: expected format <provider>:<params>", uri)
	}

	provider := parts[0]
	rawParams := parts[1]

	switch provider {
	case "github":
		params, err := parseKeyValueParams(rawParams)
		if err != nil {
			return nil, fmt.Errorf("invalid github source params: %w", err)
		}
		return &SourceConfig{
			Provider: provider,
			Params:   params,
			}, nil
	case "file":
		// For file sources, the entire params section is the path
		return &SourceConfig{
			Provider: provider,
			Params:   map[string]string{"path": rawParams},
			}, nil
	default:
		return nil, fmt.Errorf("unknown source provider %q: supported providers are github, file", provider)
	}
}

// parseKeyValueParams parses comma-separated key=value pairs.
func parseKeyValueParams(raw string) (map[string]string, error) {
	params := make(map[string]string)
	pairs := strings.Split(raw, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 || kv[0] == "" {
			return nil, fmt.Errorf("invalid parameter %q: expected key=value", pair)
		}
		params[kv[0]] = kv[1]
	}
	return params, nil
}
