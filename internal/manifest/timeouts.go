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

func (t *Timeouts) GetSkillInstall() time.Duration {
	if t != nil && t.SkillInstallSec > 0 {
		return time.Duration(t.SkillInstallSec) * time.Second
	}
	return timeouts.SkillInstall
}

func (t *Timeouts) GetSkillCLI() time.Duration {
	if t != nil && t.SkillCLISec > 0 {
		return time.Duration(t.SkillCLISec) * time.Second
	}
	return timeouts.SkillCLI
}

func (t *Timeouts) GetSkillHTTP() time.Duration {
	if t != nil && t.SkillHTTPSec > 0 {
		return time.Duration(t.SkillHTTPSec) * time.Second
	}
	return timeouts.SkillHTTP
}

func (t *Timeouts) GetSkillHTTPHeader() time.Duration {
	if t != nil && t.SkillHTTPHeaderSec > 0 {
		return time.Duration(t.SkillHTTPHeaderSec) * time.Second
	}
	return timeouts.SkillHTTPHeader
}

func (t *Timeouts) GetSkillPublish() time.Duration {
	if t != nil && t.SkillPublishSec > 0 {
		return time.Duration(t.SkillPublishSec) * time.Second
	}
	return timeouts.SkillPublish
}

func (t *Timeouts) GetProcessGrace() time.Duration {
	if t != nil && t.ProcessGraceSec > 0 {
		return time.Duration(t.ProcessGraceSec) * time.Second
	}
	return timeouts.ProcessGrace
}

func (t *Timeouts) GetStdoutDrain() time.Duration {
	if t != nil && t.StdoutDrainSec > 0 {
		return time.Duration(t.StdoutDrainSec) * time.Second
	}
	return timeouts.StdoutDrain
}

func (t *Timeouts) GetGateApproval() time.Duration {
	if t != nil && t.GateApprovalHours > 0 {
		return time.Duration(t.GateApprovalHours) * time.Hour
	}
	return timeouts.GateApproval
}

func (t *Timeouts) GetGatePollInterval() time.Duration {
	if t != nil && t.GatePollIntervalSec > 0 {
		return time.Duration(t.GatePollIntervalSec) * time.Second
	}
	return timeouts.GatePollInterval
}

func (t *Timeouts) GetGatePollTimeout() time.Duration {
	if t != nil && t.GatePollTimeoutMin > 0 {
		return time.Duration(t.GatePollTimeoutMin) * time.Minute
	}
	return timeouts.GatePollTimeout
}

func (t *Timeouts) GetGitCommand() time.Duration {
	if t != nil && t.GitCommandSec > 0 {
		return time.Duration(t.GitCommandSec) * time.Second
	}
	return timeouts.GitCommand
}

func (t *Timeouts) GetForgeAPI() time.Duration {
	if t != nil && t.ForgeAPISec > 0 {
		return time.Duration(t.ForgeAPISec) * time.Second
	}
	return timeouts.ForgeAPI
}

func (t *Timeouts) GetRetryMaxDelay() time.Duration {
	if t != nil && t.RetryMaxDelaySec > 0 {
		return time.Duration(t.RetryMaxDelaySec) * time.Second
	}
	return timeouts.RetryMaxDelay
}
