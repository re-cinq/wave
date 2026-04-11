package webui

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// encodeCursor encodes a pagination cursor to a base64 string.
func encodeCursor(t time.Time, runID string) string {
	c := PaginationCursor{
		Timestamp: t.Unix(),
		RunID:     runID,
	}
	data, _ := json.Marshal(c)
	return base64.URLEncoding.EncodeToString(data)
}

// decodeCursor decodes a base64 pagination cursor string.
func decodeCursor(s string) (*PaginationCursor, error) {
	if s == "" {
		return nil, nil
	}
	data, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor encoding: %w", err)
	}
	var c PaginationCursor
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("invalid cursor format: %w", err)
	}
	return &c, nil
}

// parsePageSize parses and validates the limit query parameter.
func parsePageSize(r *http.Request) int {
	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		return defaultPageSize
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		return defaultPageSize
	}
	if limit > maxPageSize {
		return maxPageSize
	}
	return limit
}

// parsePageNumber parses and validates the page query parameter.
// Returns 1 for missing, non-numeric, zero, or negative values.
func parsePageNumber(r *http.Request) int {
	pageStr := r.URL.Query().Get("page")
	if pageStr == "" {
		return 1
	}
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		return 1
	}
	return page
}

// validateStateFilter validates a state filter value.
// Returns the validated state or "open" as default for invalid values.
func validateStateFilter(state string) string {
	switch state {
	case "open", "closed", "all":
		return state
	default:
		return "open"
	}
}
