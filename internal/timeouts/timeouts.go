// Package timeouts provides default timeout constants used across Wave.
// This is a leaf package with zero internal dependencies, breaking import
// cycles between manifest, adapter, skill, and other packages.
//
// All values are configurable via runtime.timeouts in wave.yaml.
// The manifest.Timeouts struct reads from YAML and falls back to these
// constants when fields are zero.
package timeouts

import "time"

const (
	StepDefault      = 5 * time.Minute
	RelayCompaction  = 5 * time.Minute
	MetaDefault      = 30 * time.Minute
	SkillInstall     = 2 * time.Minute
	SkillCLI         = 2 * time.Minute
	SkillHTTP        = 2 * time.Minute
	SkillHTTPHeader  = 30 * time.Second
	SkillPublish     = 30 * time.Second
	ProcessGrace     = 3 * time.Second
	StdoutDrain      = 1 * time.Second
	GateApproval     = 24 * time.Hour
	GatePollInterval = 30 * time.Second
	GatePollTimeout  = 30 * time.Minute
	GitCommand       = 30 * time.Second
	ForgeAPI         = 15 * time.Second
	ForgeAPIList     = 30 * time.Second
	RetryMaxDelay    = 60 * time.Second
)
