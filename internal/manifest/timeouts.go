package manifest

import (
	"time"

	"github.com/recinq/wave/internal/timeouts"
)

// Timeouts centralizes all configurable timeout and duration defaults.
// Every operational timeout in Wave reads from this config via getter methods.
// Zero values fall back to the constants in internal/timeouts.
//
// Configure in wave.yaml under runtime.timeouts:
//
//	runtime:
//	  timeouts:
//	    step_default_minutes: 30
//	    relay_compaction_minutes: 5
//	    meta_default_minutes: 30
//	    skill_install_seconds: 120
//	    skill_cli_seconds: 120
//	    skill_http_seconds: 120
//	    skill_http_header_seconds: 30
//	    skill_publish_seconds: 30
//	    process_grace_seconds: 3
//	    stdout_drain_seconds: 1
//	    gate_approval_hours: 24
//	    gate_poll_interval_seconds: 30
//	    gate_poll_timeout_minutes: 30
//	    git_command_seconds: 30
//	    forge_api_seconds: 15
//	    retry_max_delay_seconds: 60
type Timeouts struct {
	StepDefaultMin      int `yaml:"step_default_minutes,omitempty"`
	RelayCompactionMin  int `yaml:"relay_compaction_minutes,omitempty"`
	MetaDefaultMin      int `yaml:"meta_default_minutes,omitempty"`
	SkillInstallSec     int `yaml:"skill_install_seconds,omitempty"`
	SkillCLISec         int `yaml:"skill_cli_seconds,omitempty"`
	SkillHTTPSec        int `yaml:"skill_http_seconds,omitempty"`
	SkillHTTPHeaderSec  int `yaml:"skill_http_header_seconds,omitempty"`
	SkillPublishSec     int `yaml:"skill_publish_seconds,omitempty"`
	ProcessGraceSec     int `yaml:"process_grace_seconds,omitempty"`
	StdoutDrainSec      int `yaml:"stdout_drain_seconds,omitempty"`
	GateApprovalHours   int `yaml:"gate_approval_hours,omitempty"`
	GatePollIntervalSec int `yaml:"gate_poll_interval_seconds,omitempty"`
	GatePollTimeoutMin  int `yaml:"gate_poll_timeout_minutes,omitempty"`
	GitCommandSec       int `yaml:"git_command_seconds,omitempty"`
	ForgeAPISec         int `yaml:"forge_api_seconds,omitempty"`
	RetryMaxDelaySec    int `yaml:"retry_max_delay_seconds,omitempty"`
}

func (t *Timeouts) GetStepDefault() time.Duration {
	if t != nil && t.StepDefaultMin > 0 {
		return time.Duration(t.StepDefaultMin) * time.Minute
	}
	return timeouts.StepDefault
}

func (t *Timeouts) GetRelayCompaction() time.Duration {
	if t != nil && t.RelayCompactionMin > 0 {
		return time.Duration(t.RelayCompactionMin) * time.Minute
	}
	return timeouts.RelayCompaction
}

func (t *Timeouts) GetMetaDefault() time.Duration {
	if t != nil && t.MetaDefaultMin > 0 {
		return time.Duration(t.MetaDefaultMin) * time.Minute
	}
	return timeouts.MetaDefault
}

func (t *Timeouts) GetGateApproval() time.Duration {
	if t != nil && t.GateApprovalHours > 0 {
		return time.Duration(t.GateApprovalHours) * time.Hour
	}
	return timeouts.GateApproval
}
