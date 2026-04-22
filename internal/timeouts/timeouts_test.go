package timeouts

import (
	"testing"
	"time"
)

// TestConstantsPositive guards against a constant being accidentally zeroed or
// set negative — all timeout defaults must be positive durations.
func TestConstantsPositive(t *testing.T) {
	tests := []struct {
		name string
		v    time.Duration
	}{
		{"StepDefault", StepDefault},
		{"RelayCompaction", RelayCompaction},
		{"MetaDefault", MetaDefault},
		{"SkillInstall", SkillInstall},
		{"SkillCLI", SkillCLI},
		{"SkillHTTP", SkillHTTP},
		{"SkillHTTPHeader", SkillHTTPHeader},
		{"SkillPublish", SkillPublish},
		{"ProcessGrace", ProcessGrace},
		{"StdoutDrain", StdoutDrain},
		{"GateApproval", GateApproval},
		{"GatePollInterval", GatePollInterval},
		{"GatePollTimeout", GatePollTimeout},
		{"GitCommand", GitCommand},
		{"ForgeAPI", ForgeAPI},
		{"ForgeAPIList", ForgeAPIList},
		{"RetryMaxDelay", RetryMaxDelay},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.v <= 0 {
				t.Errorf("%s = %v, want > 0", tt.name, tt.v)
			}
		})
	}
}

// TestOrderingInvariants pins relationships that the rest of the codebase relies
// on — e.g. MetaDefault must exceed StepDefault, poll interval must be shorter
// than poll timeout.
func TestOrderingInvariants(t *testing.T) {
	if MetaDefault <= StepDefault {
		t.Errorf("MetaDefault (%v) must exceed StepDefault (%v)", MetaDefault, StepDefault)
	}
	if GatePollInterval >= GatePollTimeout {
		t.Errorf("GatePollInterval (%v) must be less than GatePollTimeout (%v)", GatePollInterval, GatePollTimeout)
	}
	if SkillHTTPHeader >= SkillHTTP {
		t.Errorf("SkillHTTPHeader (%v) must be less than SkillHTTP (%v)", SkillHTTPHeader, SkillHTTP)
	}
	if GateApproval <= GatePollTimeout {
		t.Errorf("GateApproval (%v) must exceed GatePollTimeout (%v)", GateApproval, GatePollTimeout)
	}
}
