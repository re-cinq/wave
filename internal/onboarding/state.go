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

// MarkOnboarded writes the onboarding state file to waveDir, recording
// that onboarding has been completed. The parent directory is created
// if it does not already exist.
func MarkOnboarded(waveDir string) error {
	if err := os.MkdirAll(waveDir, 0755); err != nil {
		return err
	}

	state := State{
		Completed:   true,
		CompletedAt: time.Now(),
		Version:     1,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(stateFile(waveDir), data, 0644)
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

// ReadState reads and parses the onboarding state file from waveDir.
// If the file does not exist, it returns (nil, nil).
func ReadState(waveDir string) (*State, error) {
	data, err := os.ReadFile(stateFile(waveDir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}
