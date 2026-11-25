package contract

import (
	"testing"
)

func TestNewValidator_SpecDerivedTest(t *testing.T) {
	validator := NewValidator(ContractConfig{Type: "spec_derived_test"})
	if validator != nil {
		t.Error("expected nil validator for spec_derived_test (runner-dependent)")
	}
}
