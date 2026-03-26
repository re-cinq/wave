package manifest

import (
	"testing"
	"time"

	"github.com/recinq/wave/internal/timeouts"
)

func TestTimeoutsGetters(t *testing.T) {
	type getter struct {
		name     string
		get      func(*Timeouts) time.Duration
		field    func(*Timeouts, int)
		fallback time.Duration
	}

	getters := []getter{
		{"StepDefault", (*Timeouts).GetStepDefault, func(t *Timeouts, v int) { t.StepDefaultMin = v }, timeouts.StepDefault},
		{"RelayCompaction", (*Timeouts).GetRelayCompaction, func(t *Timeouts, v int) { t.RelayCompactionMin = v }, timeouts.RelayCompaction},
		{"MetaDefault", (*Timeouts).GetMetaDefault, func(t *Timeouts, v int) { t.MetaDefaultMin = v }, timeouts.MetaDefault},
		{"SkillInstall", (*Timeouts).GetSkillInstall, func(t *Timeouts, v int) { t.SkillInstallSec = v }, timeouts.SkillInstall},
		{"SkillCLI", (*Timeouts).GetSkillCLI, func(t *Timeouts, v int) { t.SkillCLISec = v }, timeouts.SkillCLI},
		{"SkillHTTP", (*Timeouts).GetSkillHTTP, func(t *Timeouts, v int) { t.SkillHTTPSec = v }, timeouts.SkillHTTP},
		{"SkillHTTPHeader", (*Timeouts).GetSkillHTTPHeader, func(t *Timeouts, v int) { t.SkillHTTPHeaderSec = v }, timeouts.SkillHTTPHeader},
		{"SkillPublish", (*Timeouts).GetSkillPublish, func(t *Timeouts, v int) { t.SkillPublishSec = v }, timeouts.SkillPublish},
		{"ProcessGrace", (*Timeouts).GetProcessGrace, func(t *Timeouts, v int) { t.ProcessGraceSec = v }, timeouts.ProcessGrace},
		{"StdoutDrain", (*Timeouts).GetStdoutDrain, func(t *Timeouts, v int) { t.StdoutDrainSec = v }, timeouts.StdoutDrain},
		{"GateApproval", (*Timeouts).GetGateApproval, func(t *Timeouts, v int) { t.GateApprovalHours = v }, timeouts.GateApproval},
		{"GatePollInterval", (*Timeouts).GetGatePollInterval, func(t *Timeouts, v int) { t.GatePollIntervalSec = v }, timeouts.GatePollInterval},
		{"GatePollTimeout", (*Timeouts).GetGatePollTimeout, func(t *Timeouts, v int) { t.GatePollTimeoutMin = v }, timeouts.GatePollTimeout},
		{"GitCommand", (*Timeouts).GetGitCommand, func(t *Timeouts, v int) { t.GitCommandSec = v }, timeouts.GitCommand},
		{"ForgeAPI", (*Timeouts).GetForgeAPI, func(t *Timeouts, v int) { t.ForgeAPISec = v }, timeouts.ForgeAPI},
		{"RetryMaxDelay", (*Timeouts).GetRetryMaxDelay, func(t *Timeouts, v int) { t.RetryMaxDelaySec = v }, timeouts.RetryMaxDelay},
	}

	for _, g := range getters {
		t.Run(g.name, func(t *testing.T) {
			// nil receiver returns fallback
			t.Run("nil_receiver", func(t *testing.T) {
				var nilT *Timeouts
				got := g.get(nilT)
				if got != g.fallback {
					t.Errorf("nil receiver: got %v, want %v", got, g.fallback)
				}
			})

			// zero field returns fallback
			t.Run("zero_field", func(t *testing.T) {
				tt := &Timeouts{}
				got := g.get(tt)
				if got != g.fallback {
					t.Errorf("zero field: got %v, want %v", got, g.fallback)
				}
			})

			// positive field returns configured value
			t.Run("positive_field", func(t *testing.T) {
				tt := &Timeouts{}
				g.field(tt, 42)
				got := g.get(tt)
				if got <= 0 {
					t.Errorf("positive field: got %v, want positive duration", got)
				}
				if got == g.fallback {
					t.Errorf("positive field: got fallback %v, want configured value", got)
				}
			})

			// negative field returns fallback (negative is treated as zero/unset)
			t.Run("negative_field", func(t *testing.T) {
				tt := &Timeouts{}
				g.field(tt, -5)
				got := g.get(tt)
				if got != g.fallback {
					t.Errorf("negative field: got %v, want fallback %v", got, g.fallback)
				}
			})
		})
	}
}

func TestValidateTimeouts_RejectsNegative(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*Timeouts)
		field string
	}{
		{"negative step_default", func(t *Timeouts) { t.StepDefaultMin = -1 }, "runtime.timeouts.step_default_minutes"},
		{"negative forge_api", func(t *Timeouts) { t.ForgeAPISec = -10 }, "runtime.timeouts.forge_api_seconds"},
		{"negative retry_max_delay", func(t *Timeouts) { t.RetryMaxDelaySec = -1 }, "runtime.timeouts.retry_max_delay_seconds"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tt := &Timeouts{}
			tc.setup(tt)
			errs := validateTimeouts(tt, "")
			if len(errs) == 0 {
				t.Fatal("expected validation error for negative timeout")
			}
			ve, ok := errs[0].(*ValidationError)
			if !ok {
				t.Fatalf("expected *ValidationError, got %T", errs[0])
			}
			if ve.Field != tc.field {
				t.Errorf("field: got %q, want %q", ve.Field, tc.field)
			}
		})
	}
}

func TestValidateTimeouts_AcceptsZeroAndPositive(t *testing.T) {
	// All zeros — no errors
	tt := &Timeouts{}
	if errs := validateTimeouts(tt, ""); len(errs) > 0 {
		t.Errorf("zero timeouts should not produce errors, got %v", errs)
	}

	// All positive — no errors
	tt = &Timeouts{
		StepDefaultMin:     5,
		RelayCompactionMin: 3,
		ForgeAPISec:        15,
		RetryMaxDelaySec:   60,
	}
	if errs := validateTimeouts(tt, ""); len(errs) > 0 {
		t.Errorf("positive timeouts should not produce errors, got %v", errs)
	}
}
