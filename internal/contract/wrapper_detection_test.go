package contract

import (
	"encoding/json"
	"testing"
)

func TestDetectErrorWrapper_NonJSON(t *testing.T) {
	result, err := DetectErrorWrapper([]byte("not json at all"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsWrapper {
		t.Error("non-JSON input should not be detected as wrapper")
	}
	if result.Confidence != "low" {
		t.Errorf("expected confidence 'low', got %q", result.Confidence)
	}
}

func TestDetectErrorWrapper_ValidNonWrapper(t *testing.T) {
	// Valid JSON that isn't an error wrapper
	input := []byte(`{"name": "test", "value": 42}`)
	result, err := DetectErrorWrapper(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsWrapper {
		t.Error("regular JSON should not be detected as wrapper")
	}
	if len(result.FieldsMatched) != 0 {
		t.Errorf("expected no matched fields, got %v", result.FieldsMatched)
	}
}

func TestDetectErrorWrapper_TableDriven(t *testing.T) {
	tests := []struct {
		name             string
		input            ErrorWrapper
		wantIsWrapper    bool
		wantConfidence   string
		wantFieldsMin    int
		wantExtractedLen int
	}{
		{
			name: "full wrapper with valid raw_output JSON",
			input: ErrorWrapper{
				ErrorType:    "contract_validation",
				RawOutput:    `{"key": "value"}`,
				ContractType: "json_schema",
				StepID:       "step-1",
				FinalError:   "validation failed",
				Attempts:     3,
			},
			wantIsWrapper:    true,
			wantConfidence:   "high",
			wantFieldsMin:    6,
			wantExtractedLen: len(`{"key": "value"}`),
		},
		{
			name: "wrapper with non-JSON raw_output lowers confidence from high to medium",
			input: ErrorWrapper{
				ErrorType:    "contract_validation",
				RawOutput:    "plain text output",
				ContractType: "json_schema",
				StepID:       "step-1",
				FinalError:   "validation failed",
				Attempts:     3,
			},
			wantIsWrapper:  true,
			wantConfidence: "medium",
			wantFieldsMin:  6,
		},
		{
			name: "wrapper with 4 fields gets medium confidence",
			input: ErrorWrapper{
				ErrorType:    "adapter_error",
				RawOutput:    `{"data": true}`,
				ContractType: "test_suite",
				StepID:       "step-2",
			},
			wantIsWrapper:  true,
			wantConfidence: "medium",
			wantFieldsMin:  4,
		},
		{
			name: "wrapper with 3 fields (minimum) gets low confidence",
			input: ErrorWrapper{
				ErrorType:    "adapter_error",
				RawOutput:    `{"data": true}`,
				ContractType: "json_schema",
			},
			wantIsWrapper:  true,
			wantConfidence: "low",
			wantFieldsMin:  3,
		},
		{
			name: "wrapper with 4 fields but non-JSON raw_output lowers to low",
			input: ErrorWrapper{
				ErrorType:    "adapter_error",
				RawOutput:    "not json",
				ContractType: "json_schema",
				StepID:       "step-1",
			},
			wantIsWrapper:  true,
			wantConfidence: "low",
			wantFieldsMin:  4,
		},
		{
			name: "error_type only - not a wrapper",
			input: ErrorWrapper{
				ErrorType: "some_error",
			},
			wantIsWrapper: false,
		},
		{
			name: "raw_output only - not a wrapper",
			input: ErrorWrapper{
				RawOutput: `{"key": "value"}`,
			},
			wantIsWrapper: false,
		},
		{
			name: "error_type + raw_output but no third field - not a wrapper",
			input: ErrorWrapper{
				ErrorType: "error",
				RawOutput: `{"key": "value"}`,
			},
			wantIsWrapper: false,
		},
		{
			name:          "empty wrapper",
			input:         ErrorWrapper{},
			wantIsWrapper: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputBytes, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("failed to marshal input: %v", err)
			}

			result, err := DetectErrorWrapper(inputBytes)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.IsWrapper != tt.wantIsWrapper {
				t.Errorf("IsWrapper = %v, want %v", result.IsWrapper, tt.wantIsWrapper)
			}

			if tt.wantIsWrapper {
				if result.Confidence != tt.wantConfidence {
					t.Errorf("Confidence = %q, want %q", result.Confidence, tt.wantConfidence)
				}
				if len(result.FieldsMatched) < tt.wantFieldsMin {
					t.Errorf("FieldsMatched count = %d, want >= %d (fields: %v)", len(result.FieldsMatched), tt.wantFieldsMin, result.FieldsMatched)
				}
				if result.ExtractedFrom != "raw_output" {
					t.Errorf("ExtractedFrom = %q, want 'raw_output'", result.ExtractedFrom)
				}
				if result.ErrorWrapper == nil {
					t.Error("ErrorWrapper should not be nil for detected wrapper")
				}
				if tt.wantExtractedLen > 0 && len(result.RawContent) != tt.wantExtractedLen {
					t.Errorf("RawContent length = %d, want %d", len(result.RawContent), tt.wantExtractedLen)
				}
			}
		})
	}
}

func TestDetectErrorWrapper_EmptyInput(t *testing.T) {
	result, err := DetectErrorWrapper([]byte{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsWrapper {
		t.Error("empty input should not be detected as wrapper")
	}
}

func TestWrapperDetectionResult_GetDebugInfo(t *testing.T) {
	tests := []struct {
		name        string
		result      WrapperDetectionResult
		inputLen    int
		wantWrapper bool
	}{
		{
			name: "non-wrapper debug info",
			result: WrapperDetectionResult{
				IsWrapper:     false,
				Confidence:    "low",
				FieldsMatched: []string{},
			},
			inputLen:    100,
			wantWrapper: false,
		},
		{
			name: "wrapper debug info includes extraction details",
			result: WrapperDetectionResult{
				IsWrapper:     true,
				Confidence:    "high",
				FieldsMatched: []string{"error_type", "raw_output", "contract_type"},
				RawContent:    []byte(`{"key": "value"}`),
				ExtractedFrom: "raw_output",
			},
			inputLen:    500,
			wantWrapper: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			debug := tt.result.GetDebugInfo(tt.inputLen)

			if debug.InputLength != tt.inputLen {
				t.Errorf("InputLength = %d, want %d", debug.InputLength, tt.inputLen)
			}
			if !debug.DetectionAttempted {
				t.Error("DetectionAttempted should always be true")
			}
			if debug.WrapperDetected != tt.wantWrapper {
				t.Errorf("WrapperDetected = %v, want %v", debug.WrapperDetected, tt.wantWrapper)
			}
			if debug.Confidence != tt.result.Confidence {
				t.Errorf("Confidence = %q, want %q", debug.Confidence, tt.result.Confidence)
			}

			if tt.wantWrapper {
				if debug.ExtractedLength != len(tt.result.RawContent) {
					t.Errorf("ExtractedLength = %d, want %d", debug.ExtractedLength, len(tt.result.RawContent))
				}
				if debug.ExtractionMethod != tt.result.ExtractedFrom {
					t.Errorf("ExtractionMethod = %q, want %q", debug.ExtractionMethod, tt.result.ExtractedFrom)
				}
			} else if debug.ExtractedLength != 0 {
				t.Errorf("ExtractedLength should be 0 for non-wrapper, got %d", debug.ExtractedLength)
			}
		})
	}
}
