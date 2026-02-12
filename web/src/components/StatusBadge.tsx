interface StatusBadgeProps {
  status: string;
}

const statusStyles: Record<string, string> = {
  pending: "badge-pending",
  running: "badge-running",
  completed: "badge-completed",
  failed: "badge-failed",
  cancelled: "badge-cancelled",
  retrying: "badge-retrying",
};

export function StatusBadge({ status }: StatusBadgeProps) {
  const className = statusStyles[status] || "badge-pending";
  return <span class={`badge ${className}`}>{status}</span>;
}
