package onboarding

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// State represents the persisted onboarding state.
type State struct {
	Completed   bool      `json:"completed"`
	CompletedAt time.Time `json:"completed_at"`
	Version     int       `json:"version"`
}

// stateFile returns the path to the onboarding state file.
func stateFile(waveDir string) string {
	return filepath.Join(waveDir, ".onboarded")
}

// IsOnboarded checks whether onboarding has been completed by reading
// the state file from waveDir. Returns false if the file is missing,
// unreadable, or contains invalid data.
func IsOnboarded(waveDir string) bool {
	data, err := os.ReadFile(stateFile(waveDir))
	if err != nil {
		return false
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return false
	}

	return state.Completed
}

// ClearOnboarding removes the onboarding state file from waveDir.
// If the file does not exist, nil is returned.
func ClearOnboarding(waveDir string) error {
	err := os.Remove(stateFile(waveDir))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
