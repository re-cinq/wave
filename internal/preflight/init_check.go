package preflight

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// onboardingState mirrors the onboarding.State struct to avoid an import cycle
// between preflight and onboarding packages (onboarding -> tui -> pipeline -> preflight).
type onboardingState struct {
	Completed   bool      `json:"completed"`
	CompletedAt time.Time `json:"completed_at"`
}

// readOnboardingState reads the onboarding state file from the .wave directory.
// Returns (nil, nil) if the file does not exist.
func readOnboardingState(waveDir string) (*onboardingState, error) {
	data, err := os.ReadFile(filepath.Join(waveDir, ".onboarded"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var state onboardingState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// CheckWaveInit verifies that Wave has been initialized by reading
// the onboarding state from the given .wave directory.
func (c *Checker) CheckWaveInit(waveDir string) ([]Result, error) {
	state, err := readOnboardingState(waveDir)
	if err != nil {
		return []Result{{
			Name:        "wave-init",
			Kind:        "init",
			OK:          false,
			Message:     fmt.Sprintf("failed to read Wave initialization state: %v", err),
			Remediation: "Check that the .wave directory is accessible and not corrupted",
		}}, nil
	}

	if state == nil {
		return []Result{{
			Name:        "wave-init",
			Kind:        "init",
			OK:          false,
			Message:     "Wave has not been initialized",
			Remediation: "Run 'wave init' to initialize the project",
		}}, nil
	}

	if state.Completed {
		return []Result{{
			Name:    "wave-init",
			Kind:    "init",
			OK:      true,
			Message: fmt.Sprintf("Wave initialized (completed: %s)", state.CompletedAt.Format(time.RFC3339)),
		}}, nil
	}

	return []Result{{
		Name:        "wave-init",
		Kind:        "init",
		OK:          false,
		Message:     "Wave initialization started but not completed",
		Remediation: "Run 'wave init' to complete project setup",
	}}, nil
}
