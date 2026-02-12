import { StatusBadge } from "./StatusBadge";

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

interface StepDetailProps {
  step: StepProgress;
}

export function StepDetail({ step }: StepDetailProps) {
  return (
    <div class="step-card">
      <div class="step-header">
        <span class="step-id">{step.step_id}</span>
        <StatusBadge status={step.state} />
        {step.persona && <span class="step-persona">{step.persona}</span>}
      </div>

      <div class="step-progress-bar-container">
        <div
          class="step-progress-bar-fill"
          style={{ width: `${step.progress}%` }}
        />
      </div>
      <div class="step-meta">
        <span>{step.progress}%</span>
        {step.current_action && (
          <span class="step-action">{step.current_action}</span>
        )}
        {step.tokens_used !== undefined && step.tokens_used > 0 && (
          <span>{step.tokens_used.toLocaleString()} tokens</span>
        )}
      </div>

      {step.message && <div class="step-message">{step.message}</div>}
    </div>
  );
}
