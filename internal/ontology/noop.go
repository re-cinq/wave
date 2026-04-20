package ontology

import "github.com/recinq/wave/internal/manifest"

// NoOp is the null implementation of Service. It is returned by New when
// Config.Enabled is false so call sites can invoke Service methods
// unconditionally.
type NoOp struct{}

func (NoOp) Enabled() bool                                                 { return false }
func (NoOp) CheckStaleness() string                                        { return "" }
func (NoOp) BuildStepSection(_, _ string, _ []string) string               { return "" }
func (NoOp) RecordUsage(_, _ string, _ []string, _ bool, _ string)         {}
func (NoOp) ValidateManifest(_ *manifest.Manifest) []error                 { return nil }
func (NoOp) InstallStalenessHook() error                                   { return nil }
