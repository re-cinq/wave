package mission

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/recinq/wave/internal/state"
)

// PollInterval is the default polling interval for SQLite state reads.
const PollInterval = 2 * time.Second

// StatePolledMsg carries run records fetched from SQLite.
type StatePolledMsg struct {
	Records []storeRecord
	Err     error
}

// StepDataMsg carries per-step state data for a run from the state store.
type StepDataMsg struct {
	RunID    string
	StepData []state.StepProgressRecord
	Events   []state.LogRecord
	Progress *state.PipelineProgressRecord
}

// PollState returns a tea.Cmd that reads run records from the state store.
func PollState(store state.StateStore) tea.Cmd {
	return tea.Tick(PollInterval, func(t time.Time) tea.Msg {
		if store == nil {
			return StatePolledMsg{}
		}

		runs, err := store.ListRuns(state.ListRunsOptions{
			Limit: 50,
		})
		if err != nil {
			return StatePolledMsg{Err: err}
		}

		records := make([]storeRecord, len(runs))
		for i, r := range runs {
			records[i] = storeRecord{
				RunID:        r.RunID,
				PipelineName: r.PipelineName,
				Status:       r.Status,
				CurrentStep:  r.CurrentStep,
				TotalTokens:  r.TotalTokens,
				ErrorMessage: r.ErrorMessage,
				StartedAt:    r.StartedAt,
				CompletedAt:  r.CompletedAt,
			}
		}

		return StatePolledMsg{Records: records}
	})
}

// InitialPoll does a one-time immediate poll of the state store.
func InitialPoll(store state.StateStore) tea.Cmd {
	return func() tea.Msg {
		if store == nil {
			return StatePolledMsg{}
		}

		runs, err := store.ListRuns(state.ListRunsOptions{
			Limit: 50,
		})
		if err != nil {
			return StatePolledMsg{Err: err}
		}

		records := make([]storeRecord, len(runs))
		for i, r := range runs {
			records[i] = storeRecord{
				RunID:        r.RunID,
				PipelineName: r.PipelineName,
				Status:       r.Status,
				CurrentStep:  r.CurrentStep,
				TotalTokens:  r.TotalTokens,
				ErrorMessage: r.ErrorMessage,
				StartedAt:    r.StartedAt,
				CompletedAt:  r.CompletedAt,
			}
		}

		return StatePolledMsg{Records: records}
	}
}

// LoadRunStepData queries the state store for step-level detail.
func LoadRunStepData(store state.StateStore, runID string) tea.Cmd {
	return func() tea.Msg {
		if store == nil {
			return StepDataMsg{RunID: runID}
		}
		stepData, _ := store.GetAllStepProgress(runID)
		events, _ := store.GetEvents(runID, state.EventQueryOptions{Limit: 200})
		progress, _ := store.GetPipelineProgress(runID)
		return StepDataMsg{
			RunID:    runID,
			StepData: stepData,
			Events:   events,
			Progress: progress,
		}
	}
}
