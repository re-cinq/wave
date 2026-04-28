package tui

import (
	"github.com/recinq/wave/internal/health"
	"github.com/recinq/wave/internal/manifest"
	"github.com/recinq/wave/internal/state"
)

// HealthCheckStatus aliases health.CheckStatus for use in the TUI layer.
type HealthCheckStatus = health.CheckStatus

// Status constants re-exported from internal/health so TUI code keeps its
// existing identifiers.
const (
	HealthCheckOK       = health.StatusOK
	HealthCheckWarn     = health.StatusWarn
	HealthCheckErr      = health.StatusErr
	HealthCheckChecking = health.StatusChecking
)

// HealthCheckResultMsg is the bubbletea message form of a health check result.
// It is a type alias of health.CheckResult so providers from the health
// package can be used directly as TUI message sources.
type HealthCheckResultMsg = health.CheckResult

// HealthDataProvider aliases health.DataProvider.
type HealthDataProvider = health.DataProvider

// DefaultHealthDataProvider aliases health.DefaultDataProvider.
type DefaultHealthDataProvider = health.DefaultDataProvider

// NewDefaultHealthDataProvider creates a new health data provider.
// Thin wrapper preserved for call-site compatibility.
func NewDefaultHealthDataProvider(m *manifest.Manifest, store state.RunStore, pipelinesDir string) *DefaultHealthDataProvider {
	return health.NewDefaultDataProvider(m, store, pipelinesDir)
}
