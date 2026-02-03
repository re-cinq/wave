package contract

import (
	"encoding/json"
)

// ErrorWrapper represents the structure used to wrap AI output when validation fails
type ErrorWrapper struct {
	Attempts        int      `json:"attempts,omitempty"`
	ContractType    string   `json:"contract_type,omitempty"`
	ErrorType       string   `json:"error_type,omitempty"`
	ExitCode        int      `json:"exit_code,omitempty"`
	FinalError      string   `json:"final_error,omitempty"`
	RawOutput       string   `json:"raw_output,omitempty"`
	Recommendations []string `json:"recommendations,omitempty"`
	StepID          string   `json:"step_id,omitempty"`
	Timestamp       string   `json:"timestamp,omitempty"`
	TokensUsed      int      `json:"tokens_used,omitempty"`
	MustPass        bool     `json:"must_pass,omitempty"`
	Persona         string   `json:"persona,omitempty"`
}

// WrapperDetectionResult contains the results of error wrapper detection
type WrapperDetectionResult struct {
	IsWrapper      bool
	ErrorWrapper   *ErrorWrapper
	RawContent     []byte
	ExtractedFrom  string // source field name
	Confidence     string // "high", "medium", "low"
	FieldsMatched  []string
}

// DetectErrorWrapper analyzes input to determine if it's an error wrapper structure
// and extracts the raw content if found
func DetectErrorWrapper(input []byte) (*WrapperDetectionResult, error) {
	result := &WrapperDetectionResult{
		IsWrapper:     false,
		Confidence:    "low",
		FieldsMatched: make([]string, 0),
	}

	// Try to parse as potential error wrapper
	var wrapper ErrorWrapper
	if err := json.Unmarshal(input, &wrapper); err != nil {
		// Not valid JSON at all, return as non-wrapper
		return result, nil
	}

	// Check for error wrapper indicators and count matches
	indicators := 0
	var matchedFields []string

	if wrapper.ErrorType != "" {
		indicators++
		matchedFields = append(matchedFields, "error_type")
	}
	if wrapper.RawOutput != "" {
		indicators++
		matchedFields = append(matchedFields, "raw_output")
	}
	if wrapper.ContractType != "" {
		indicators++
		matchedFields = append(matchedFields, "contract_type")
	}
	if wrapper.StepID != "" {
		indicators++
		matchedFields = append(matchedFields, "step_id")
	}
	if wrapper.FinalError != "" {
		indicators++
		matchedFields = append(matchedFields, "final_error")
	}
	if wrapper.Attempts > 0 {
		indicators++
		matchedFields = append(matchedFields, "attempts")
	}

	result.FieldsMatched = matchedFields

	// Must have error_type and raw_output plus at least one other field to be considered a wrapper
	if wrapper.ErrorType != "" && wrapper.RawOutput != "" && indicators >= 3 {
		result.IsWrapper = true
		result.ErrorWrapper = &wrapper
		result.RawContent = []byte(wrapper.RawOutput)
		result.ExtractedFrom = "raw_output"

		// Determine confidence based on number of matched fields
		if indicators >= 6 {
			result.Confidence = "high"
		} else if indicators >= 4 {
			result.Confidence = "medium"
		} else {
			result.Confidence = "low"
		}

		// Validate that raw_output contains parseable JSON
		var testJSON interface{}
		if err := json.Unmarshal(result.RawContent, &testJSON); err != nil {
			// Raw output is not valid JSON, lower confidence
			if result.Confidence == "high" {
				result.Confidence = "medium"
			} else if result.Confidence == "medium" {
				result.Confidence = "low"
			}
		}
	}

	return result, nil
}

// WrapperDetectionDebug contains debugging information about wrapper detection
type WrapperDetectionDebug struct {
	InputLength        int      `json:"input_length"`
	DetectionAttempted bool     `json:"detection_attempted"`
	WrapperDetected    bool     `json:"wrapper_detected"`
	FieldsMatched      []string `json:"fields_matched"`
	ExtractedLength    int      `json:"extracted_length,omitempty"`
	ExtractionMethod   string   `json:"extraction_method,omitempty"`
	Confidence         string   `json:"confidence"`
}

// GetDebugInfo returns debugging information about the wrapper detection result
func (w *WrapperDetectionResult) GetDebugInfo(inputLength int) WrapperDetectionDebug {
	debug := WrapperDetectionDebug{
		InputLength:        inputLength,
		DetectionAttempted: true,
		WrapperDetected:    w.IsWrapper,
		FieldsMatched:      w.FieldsMatched,
		Confidence:         w.Confidence,
	}

	if w.IsWrapper {
		debug.ExtractedLength = len(w.RawContent)
		debug.ExtractionMethod = w.ExtractedFrom
	}

	return debug
}