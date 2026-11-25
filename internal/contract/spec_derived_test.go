package contract

import (
	"testing"
)

// --- Config validation tests (Task 3.1) ---

// --- Persona separation tests (Task 3.2) ---

// --- Spec artifact loading tests (Task 3.3) ---

// --- NewValidator returns nil for spec_derived_test (Task 3.4) ---

func TestNewValidator_SpecDerivedTest(t *testing.T) {
	validator := NewValidator(ContractConfig{Type: "spec_derived_test"})
	if validator != nil {
		t.Error("expected nil validator for spec_derived_test (runner-dependent)")
	}
}

// --- Validate() returns runner-required error (Task 3.5) ---

// --- parseSpecDerivedResult tests ---

// --- ValidateSpecDerived integration tests ---

// --- buildSpecDerivedPrompt tests ---
