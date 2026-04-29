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
	// Step + relay timeouts loosened 2026-04-30 — impl-issue runs that exercise
	// `go test ./...` on the Wave codebase routinely take 6-10 minutes; the old
	// 5-minute step default forced a "canceled" failure class even when the
	// step was making progress. ForgeAPI* loosened so transient GitHub
	// hiccups don't poison long-running pipelines. Override per-step in
	// pipeline yaml or per-runtime in wave.yaml when a tighter bound matters.
	StepDefault      = 30 * time.Minute
	RelayCompaction  = 15 * time.Minute
	MetaDefault      = 60 * time.Minute
	SkillInstall     = 5 * time.Minute
	SkillCLI         = 5 * time.Minute
	SkillHTTP        = 5 * time.Minute
	SkillHTTPHeader  = 60 * time.Second
	SkillPublish     = 60 * time.Second
	ProcessGrace     = 3 * time.Second
	StdoutDrain      = 1 * time.Second
	GateApproval     = 24 * time.Hour
	GatePollInterval = 30 * time.Second
	GatePollTimeout  = 60 * time.Minute
	GitCommand       = 90 * time.Second
	ForgeAPI         = 60 * time.Second
	ForgeAPIList     = 90 * time.Second
	RetryMaxDelay    = 60 * time.Second
)
