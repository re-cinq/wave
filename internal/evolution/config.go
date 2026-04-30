package evolution

// Config holds the threshold parameters that govern Service.ShouldEvolve.
// Zero values on any non-Enabled field fall back to the corresponding
// DefaultConfig value, so a partially populated override (e.g. only
// drift_pass_drop) keeps the unspecified fields sane.
type Config struct {
	Enabled           bool
	EveryNWindow      int     // rows per half-window for the every-N median compare
	EveryNJudgeDrop   float64 // min median judge_score drop to fire every-N
	DriftWindow       int     // rows for contract_pass drift heuristic
	DriftPassDrop     float64 // min absolute pass-rate drop to fire drift
	RetryWindow       int     // rows for retry-rate heuristic
	RetryAvgThreshold float64 // avg retry_count over RetryWindow to fire retry-rate
}

// DefaultConfig returns the compiled-in thresholds documented in the
// acceptance criteria for issue #1612.
func DefaultConfig() Config {
	return Config{
		Enabled:           true,
		EveryNWindow:      10,
		EveryNJudgeDrop:   0.1,
		DriftWindow:       20,
		DriftPassDrop:     0.15,
		RetryWindow:       10,
		RetryAvgThreshold: 2.0,
	}
}

// YAMLOverrides mirrors the wave.yaml `evolution:` block. Only the fields
// the operator sets carry over; zero / unset numeric fields fall back to
// DefaultConfig. Enabled defaults to true when nil.
type YAMLOverrides struct {
	Enabled           *bool
	EveryNWindow      int
	EveryNJudgeDrop   float64
	DriftWindow       int
	DriftPassDrop     float64
	RetryWindow       int
	RetryAvgThreshold float64
}

// Apply layers o on top of the defaults. A nil receiver returns DefaultConfig
// untouched so callers can pass a *YAMLOverrides loaded from manifest without
// nil-check boilerplate.
func (o *YAMLOverrides) Apply() Config {
	cfg := DefaultConfig()
	if o == nil {
		return cfg
	}
	if o.Enabled != nil {
		cfg.Enabled = *o.Enabled
	}
	if o.EveryNWindow > 0 {
		cfg.EveryNWindow = o.EveryNWindow
	}
	if o.EveryNJudgeDrop > 0 {
		cfg.EveryNJudgeDrop = o.EveryNJudgeDrop
	}
	if o.DriftWindow > 0 {
		cfg.DriftWindow = o.DriftWindow
	}
	if o.DriftPassDrop > 0 {
		cfg.DriftPassDrop = o.DriftPassDrop
	}
	if o.RetryWindow > 0 {
		cfg.RetryWindow = o.RetryWindow
	}
	if o.RetryAvgThreshold > 0 {
		cfg.RetryAvgThreshold = o.RetryAvgThreshold
	}
	return cfg
}

// maxWindow is the largest row count any heuristic needs. Used to cap the
// state-store query so hot pipelines do not scan unbounded history.
func (c Config) maxWindow() int {
	n := c.EveryNWindow * 2
	if c.DriftWindow > n {
		n = c.DriftWindow
	}
	if c.RetryWindow > n {
		n = c.RetryWindow
	}
	return n
}
