import { useApi } from "../hooks/useApi";
import { StatusBadge } from "./StatusBadge";
import { StepDetail } from "./StepDetail";

interface RunDetail {
  run: {
    run_id: string;
    pipeline_name: string;
    status: string;
    input?: string;
    current_step?: string;
    total_tokens: number;
    started_at: string;
    completed_at?: string;
    duration_ms: number;
    error?: string;
    tags?: string[];
  };
  steps?: StepProgress[];
  pipeline_progress?: {
    total_steps: number;
    completed_steps: number;
    current_step_index: number;
    overall_progress: number;
  };
}

interface StepProgress {
  step_id: string;
  run_id: string;
  persona?: string;
  state: string;
  progress: number;
  current_action?: string;
  message?: string;
  started_at?: string;
  updated_at: string;
  tokens_used?: number;
}

interface EventEntry {
  id: number;
  timestamp: string;
  step_id?: string;
  state: string;
  persona?: string;
  message?: string;
  tokens_used?: number;
  duration_ms?: number;
}

interface EventListData {
  events: EventEntry[];
}

interface PipelineDetailProps {
  runID: string;
  onBack: () => void;
  refreshKey: number;
}

export function PipelineDetail({
  runID,
  onBack,
  refreshKey,
}: PipelineDetailProps) {
  const { data, loading, error } = useApi<RunDetail>(
    `/api/runs/${runID}`,
    refreshKey,
  );

  const { data: eventsData } = useApi<EventListData>(
    `/api/runs/${runID}/events`,
    refreshKey,
  );

  if (loading && !data) return <div class="loading">Loading...</div>;
  if (error) return <div class="error-msg">Error: {error}</div>;
  if (!data) return null;

  const { run, steps, pipeline_progress } = data;
  const events = eventsData?.events ?? [];

  return (
    <div class="pipeline-detail">
      <button class="back-btn" onClick={onBack}>
        Back
      </button>

      <div class="run-header">
        <h2>{run.run_id}</h2>
        <StatusBadge status={run.status} />
      </div>

      <div class="run-info">
        <dl>
          <dt>Pipeline</dt>
          <dd>{run.pipeline_name}</dd>
          <dt>Started</dt>
          <dd>{new Date(run.started_at).toLocaleString()}</dd>
          {run.completed_at && (
            <>
              <dt>Completed</dt>
              <dd>{new Date(run.completed_at).toLocaleString()}</dd>
            </>
          )}
          <dt>Duration</dt>
          <dd>{formatDuration(run.duration_ms)}</dd>
          <dt>Tokens</dt>
          <dd>{run.total_tokens.toLocaleString()}</dd>
          {run.input && (
            <>
              <dt>Input</dt>
              <dd class="run-input">{run.input}</dd>
            </>
          )}
          {run.error && (
            <>
              <dt>Error</dt>
              <dd class="run-error">{run.error}</dd>
            </>
          )}
        </dl>
      </div>

      {pipeline_progress && (
        <div class="progress-section">
          <h3>Pipeline Progress</h3>
          <div class="progress-bar-container">
            <div
              class="progress-bar-fill"
              style={{ width: `${pipeline_progress.overall_progress}%` }}
            />
          </div>
          <span class="progress-text">
            {pipeline_progress.completed_steps}/{pipeline_progress.total_steps}{" "}
            steps ({pipeline_progress.overall_progress}%)
          </span>
        </div>
      )}

      {steps && steps.length > 0 && (
        <div class="steps-section">
          <h3>Steps</h3>
          {steps.map((step) => (
            <StepDetail key={step.step_id} step={step} />
          ))}
        </div>
      )}

      {events.length > 0 && (
        <div class="events-section">
          <h3>Event Log</h3>
          <table class="events-table">
            <thead>
              <tr>
                <th>Time</th>
                <th>State</th>
                <th>Step</th>
                <th>Persona</th>
                <th>Message</th>
              </tr>
            </thead>
            <tbody>
              {events.map((ev) => (
                <tr key={ev.id}>
                  <td class="mono">
                    {new Date(ev.timestamp).toLocaleTimeString()}
                  </td>
                  <td>
                    <StatusBadge status={ev.state} />
                  </td>
                  <td>{ev.step_id || "-"}</td>
                  <td>{ev.persona || "-"}</td>
                  <td>{ev.message || "-"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  const s = Math.floor(ms / 1000);
  if (s < 60) return `${s}s`;
  const m = Math.floor(s / 60);
  const rem = s % 60;
  if (m < 60) return `${m}m ${rem}s`;
  const h = Math.floor(m / 60);
  return `${h}h ${m % 60}m`;
}
