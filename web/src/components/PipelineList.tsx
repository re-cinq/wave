import { useState } from "preact/hooks";
import { useApi } from "../hooks/useApi";
import { StatusBadge } from "./StatusBadge";

interface Run {
  run_id: string;
  pipeline_name: string;
  status: string;
  current_step?: string;
  total_tokens: number;
  started_at: string;
  completed_at?: string;
  duration_ms: number;
  error?: string;
  tags?: string[];
}

interface RunListData {
  runs: Run[];
  total: number;
}

interface PipelineListProps {
  onSelectRun: (runID: string) => void;
  refreshKey: number;
}

export function PipelineList({ onSelectRun, refreshKey }: PipelineListProps) {
  const [statusFilter, setStatusFilter] = useState("");
  const [search, setSearch] = useState("");

  const queryParams = new URLSearchParams();
  queryParams.set("limit", "50");
  if (statusFilter) queryParams.set("status", statusFilter);

  const { data, loading, error } = useApi<RunListData>(
    `/api/runs?${queryParams}`,
    refreshKey,
  );

  const runs = data?.runs ?? [];
  const filtered = search
    ? runs.filter(
        (r) =>
          r.run_id.includes(search) ||
          r.pipeline_name.includes(search),
      )
    : runs;

  return (
    <div class="pipeline-list">
      <div class="list-controls">
        <input
          type="text"
          class="search-input"
          placeholder="Search runs..."
          value={search}
          onInput={(e) => setSearch((e.target as HTMLInputElement).value)}
        />
        <select
          class="status-filter"
          value={statusFilter}
          onChange={(e) =>
            setStatusFilter((e.target as HTMLSelectElement).value)
          }
        >
          <option value="">All statuses</option>
          <option value="running">Running</option>
          <option value="completed">Completed</option>
          <option value="failed">Failed</option>
          <option value="pending">Pending</option>
          <option value="cancelled">Cancelled</option>
        </select>
      </div>

      {loading && runs.length === 0 && <div class="loading">Loading...</div>}
      {error && <div class="error-msg">Error: {error}</div>}

      {!loading && filtered.length === 0 && (
        <div class="empty-state">No pipeline runs found</div>
      )}

      <table class="runs-table">
        <thead>
          <tr>
            <th>Run ID</th>
            <th>Pipeline</th>
            <th>Status</th>
            <th>Step</th>
            <th>Duration</th>
            <th>Tokens</th>
            <th>Started</th>
          </tr>
        </thead>
        <tbody>
          {filtered.map((run) => (
            <tr
              key={run.run_id}
              class="run-row"
              onClick={() => onSelectRun(run.run_id)}
            >
              <td class="mono">{run.run_id}</td>
              <td>{run.pipeline_name}</td>
              <td>
                <StatusBadge status={run.status} />
              </td>
              <td>{run.current_step || "-"}</td>
              <td>{formatDuration(run.duration_ms)}</td>
              <td>{formatTokens(run.total_tokens)}</td>
              <td>{formatTime(run.started_at)}</td>
            </tr>
          ))}
        </tbody>
      </table>
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

function formatTokens(n: number): string {
  if (n < 1000) return `${n}`;
  if (n < 1000000) return `${(n / 1000).toFixed(1)}k`;
  return `${(n / 1000000).toFixed(1)}M`;
}

function formatTime(iso: string): string {
  try {
    const d = new Date(iso);
    return d.toLocaleString(undefined, {
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return iso;
  }
}
