package state

// GetAllMigrations returns all available migrations in chronological order
func GetAllMigrations() []Migration {
	return []Migration{
		{
			Version:     1,
			Description: "Create initial pipeline and step state tables",
			Up: `
CREATE TABLE IF NOT EXISTS pipeline_state (
    pipeline_id TEXT PRIMARY KEY,
    pipeline_name TEXT NOT NULL,
    status TEXT NOT NULL,
    input TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS step_state (
    step_id TEXT PRIMARY KEY,
    pipeline_id TEXT NOT NULL,
    state TEXT NOT NULL,
    retry_count INTEGER NOT NULL DEFAULT 0,
    started_at INTEGER,
    completed_at INTEGER,
    workspace_path TEXT,
    error_message TEXT,
    FOREIGN KEY (pipeline_id) REFERENCES pipeline_state(pipeline_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_step_pipeline_id ON step_state(pipeline_id);
`,
			Down: "",
		},
		{
			Version:     2,
			Description: "Add ops commands tables for run tracking (spec 016)",
			Up: `
-- Track individual pipeline runs for ops commands
CREATE TABLE IF NOT EXISTS pipeline_run (
    run_id TEXT PRIMARY KEY,
    pipeline_name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
    input TEXT,
    current_step TEXT,
    total_tokens INTEGER DEFAULT 0,
    started_at INTEGER NOT NULL,
    completed_at INTEGER,
    cancelled_at INTEGER,
    error_message TEXT
);

CREATE INDEX IF NOT EXISTS idx_run_pipeline ON pipeline_run(pipeline_name);
CREATE INDEX IF NOT EXISTS idx_run_status ON pipeline_run(status);
CREATE INDEX IF NOT EXISTS idx_run_started ON pipeline_run(started_at);

-- Store event log entries for pipeline runs
CREATE TABLE IF NOT EXISTS event_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    step_id TEXT,
    state TEXT NOT NULL,
    persona TEXT,
    message TEXT,
    tokens_used INTEGER,
    duration_ms INTEGER,
    FOREIGN KEY (run_id) REFERENCES pipeline_run(run_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_event_run ON event_log(run_id);
CREATE INDEX IF NOT EXISTS idx_event_timestamp ON event_log(timestamp);

-- Track artifacts produced by pipeline runs
CREATE TABLE IF NOT EXISTS artifact (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id TEXT NOT NULL,
    step_id TEXT NOT NULL,
    name TEXT NOT NULL,
    path TEXT NOT NULL,
    type TEXT,
    size_bytes INTEGER,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (run_id) REFERENCES pipeline_run(run_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_artifact_run ON artifact(run_id);

-- Cancellation flags for pipeline runs
CREATE TABLE IF NOT EXISTS cancellation (
    run_id TEXT PRIMARY KEY,
    requested_at INTEGER NOT NULL,
    force BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (run_id) REFERENCES pipeline_run(run_id) ON DELETE CASCADE
);
`,
			Down: "",
		},
		{
			Version:     3,
			Description: "Add performance metrics tables (spec 018 - part 1)",
			Up: `
-- Track historical performance metrics for steps
CREATE TABLE IF NOT EXISTS performance_metric (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id TEXT NOT NULL,
    step_id TEXT NOT NULL,
    pipeline_name TEXT NOT NULL,
    persona TEXT,
    started_at INTEGER NOT NULL,
    completed_at INTEGER,
    duration_ms INTEGER,
    tokens_used INTEGER DEFAULT 0,
    files_modified INTEGER DEFAULT 0,
    artifacts_generated INTEGER DEFAULT 0,
    memory_bytes INTEGER,
    success BOOLEAN DEFAULT TRUE,
    error_message TEXT,
    FOREIGN KEY (run_id) REFERENCES pipeline_run(run_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_perf_run ON performance_metric(run_id);
CREATE INDEX IF NOT EXISTS idx_perf_step ON performance_metric(step_id);
CREATE INDEX IF NOT EXISTS idx_perf_pipeline ON performance_metric(pipeline_name);
CREATE INDEX IF NOT EXISTS idx_perf_started ON performance_metric(started_at);
`,
			Down: "",
		},
		{
			Version:     4,
			Description: "Add progress tracking tables (spec 018 - part 2)",
			Up: `
-- Progress snapshots for step-level granular tracking
CREATE TABLE IF NOT EXISTS progress_snapshot (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id TEXT NOT NULL,
    step_id TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    progress INTEGER NOT NULL CHECK (progress >= 0 AND progress <= 100),
    current_action TEXT,
    estimated_time_ms INTEGER,
    validation_phase TEXT,
    compaction_stats TEXT,
    FOREIGN KEY (run_id) REFERENCES pipeline_run(run_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_progress_run ON progress_snapshot(run_id);
CREATE INDEX IF NOT EXISTS idx_progress_step ON progress_snapshot(step_id);
CREATE INDEX IF NOT EXISTS idx_progress_timestamp ON progress_snapshot(timestamp);

-- Step progress tracking for real-time updates
CREATE TABLE IF NOT EXISTS step_progress (
    step_id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL,
    persona TEXT,
    state TEXT NOT NULL,
    progress INTEGER DEFAULT 0 CHECK (progress >= 0 AND progress <= 100),
    current_action TEXT,
    message TEXT,
    started_at INTEGER,
    updated_at INTEGER NOT NULL,
    estimated_completion_ms INTEGER,
    tokens_used INTEGER DEFAULT 0,
    FOREIGN KEY (run_id) REFERENCES pipeline_run(run_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_step_progress_run ON step_progress(run_id);
CREATE INDEX IF NOT EXISTS idx_step_progress_state ON step_progress(state);
CREATE INDEX IF NOT EXISTS idx_step_progress_updated ON step_progress(updated_at);

-- Pipeline-level progress aggregation
CREATE TABLE IF NOT EXISTS pipeline_progress (
    run_id TEXT PRIMARY KEY,
    total_steps INTEGER NOT NULL,
    completed_steps INTEGER DEFAULT 0,
    current_step_index INTEGER DEFAULT 0,
    overall_progress INTEGER DEFAULT 0 CHECK (overall_progress >= 0 AND overall_progress <= 100),
    estimated_completion_ms INTEGER,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (run_id) REFERENCES pipeline_run(run_id) ON DELETE CASCADE
);
`,
			Down: "",
		},
		{
			Version:     5,
			Description: "Add artifact metadata extension (spec 018 - part 3)",
			Up: `
-- Artifact metadata extension for progress visualization
CREATE TABLE IF NOT EXISTS artifact_metadata (
    artifact_id INTEGER PRIMARY KEY,
    run_id TEXT NOT NULL,
    step_id TEXT NOT NULL,
    preview_text TEXT,
    mime_type TEXT,
    encoding TEXT,
    metadata_json TEXT,
    indexed_at INTEGER NOT NULL,
    FOREIGN KEY (artifact_id) REFERENCES artifact(id) ON DELETE CASCADE,
    FOREIGN KEY (run_id) REFERENCES pipeline_run(run_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_artifact_meta_run ON artifact_metadata(run_id);
CREATE INDEX IF NOT EXISTS idx_artifact_meta_step ON artifact_metadata(step_id);
`,
			Down: "",
		},
		{
			Version:     6,
			Description: "Add tags support for pipeline runs",
			Up: `
-- Add tags column to pipeline_run for categorization and filtering
-- Stores JSON array of tag strings e.g. ["production", "critical", "deploy"]
ALTER TABLE pipeline_run ADD COLUMN tags_json TEXT DEFAULT '[]';

-- Create index for efficient tag-based filtering
-- Note: SQLite's json_each can be used for searching within tags
CREATE INDEX IF NOT EXISTS idx_run_tags ON pipeline_run(tags_json);
`,
			Down: "",
		},
		{
			Version:     7,
			Description: "Add branch_name to pipeline_run for TUI header branch display",
			Up: `
ALTER TABLE pipeline_run ADD COLUMN branch_name TEXT DEFAULT '';
`,
			Down: "",
		},
		{
			Version:     8,
			Description: "Add pid column to pipeline_run for detached subprocess tracking",
			Up: `
ALTER TABLE pipeline_run ADD COLUMN pid INTEGER DEFAULT 0;
`,
			Down: "",
		},
		{
			Version:     9,
			Description: "Add step_attempt table for retry/recovery tracking",
			Up: `
CREATE TABLE IF NOT EXISTS step_attempt (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id TEXT NOT NULL,
    step_id TEXT NOT NULL,
    attempt INTEGER NOT NULL,
    state TEXT NOT NULL,
    error_message TEXT DEFAULT '',
    failure_class TEXT DEFAULT '',
    stdout_tail TEXT DEFAULT '',
    tokens_used INTEGER DEFAULT 0,
    duration_ms INTEGER DEFAULT 0,
    started_at INTEGER NOT NULL,
    completed_at INTEGER,
    FOREIGN KEY (run_id) REFERENCES pipeline_run(run_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_attempt_run ON step_attempt(run_id);
CREATE INDEX IF NOT EXISTS idx_attempt_step ON step_attempt(step_id);
`,
			Down: "",
		},
		{
			Version:     10,
			Description: "Add chat_session table for bidirectional chat persistence",
			Up: `
CREATE TABLE IF NOT EXISTS chat_session (
    session_id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL,
    step_filter TEXT DEFAULT '',
    workspace_path TEXT NOT NULL,
    model TEXT DEFAULT '',
    created_at INTEGER NOT NULL,
    last_resumed_at INTEGER,
    FOREIGN KEY (run_id) REFERENCES pipeline_run(run_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_chat_session_run ON chat_session(run_id);
`,
			Down: "",
		},
		{
			Version:     11,
			Description: "Add ontology_usage table for decision lineage tracking",
			Up: `CREATE TABLE IF NOT EXISTS ontology_usage (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id TEXT NOT NULL,
    step_id TEXT NOT NULL,
    context_name TEXT NOT NULL,
    invariant_count INTEGER NOT NULL DEFAULT 0,
    step_status TEXT NOT NULL,
    contract_passed INTEGER,
    created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    FOREIGN KEY (run_id) REFERENCES pipeline_run(run_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_ontology_usage_context ON ontology_usage(context_name);
CREATE INDEX IF NOT EXISTS idx_ontology_usage_run ON ontology_usage(run_id);`,
			Down: `DROP TABLE IF EXISTS ontology_usage;
DROP INDEX IF EXISTS idx_ontology_usage_context;
DROP INDEX IF EXISTS idx_ontology_usage_run;`,
		},
		{
			Version:     12,
			Description: "Add visit_count column to step_state for graph-mode loop tracking",
			Up:          `ALTER TABLE step_state ADD COLUMN visit_count INTEGER NOT NULL DEFAULT 0;`,
			Down:        "",
		},
		{
			Version:     13,
			Description: "Add retrospective table for run retrospective indexing",
			Up: `CREATE TABLE IF NOT EXISTS retrospective (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id TEXT NOT NULL,
    pipeline_name TEXT NOT NULL,
    smoothness TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'quantitative',
    file_path TEXT NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    FOREIGN KEY (run_id) REFERENCES pipeline_run(run_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_retro_run ON retrospective(run_id);
CREATE INDEX IF NOT EXISTS idx_retro_pipeline ON retrospective(pipeline_name);
CREATE INDEX IF NOT EXISTS idx_retro_created ON retrospective(created_at);`,
			Down: `DROP TABLE IF EXISTS retrospective;`,
		},
		{
			Version:     14,
			Description: "Add parent_run_id and parent_step_id to pipeline_run for sub-pipeline composition",
			Up: `ALTER TABLE pipeline_run ADD COLUMN parent_run_id TEXT;
ALTER TABLE pipeline_run ADD COLUMN parent_step_id TEXT;
CREATE INDEX IF NOT EXISTS idx_run_parent ON pipeline_run(parent_run_id);`,
			Down: "",
		},
		{
			Version:     15,
			Description: "Add checkpoint table and forked_from column for fork/rewind support",
			Up: `CREATE TABLE IF NOT EXISTS checkpoint (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id TEXT NOT NULL,
    step_id TEXT NOT NULL,
    step_index INTEGER NOT NULL,
    workspace_path TEXT NOT NULL DEFAULT '',
    workspace_commit_sha TEXT DEFAULT '',
    artifact_snapshot TEXT NOT NULL DEFAULT '{}',
    created_at INTEGER NOT NULL,
    FOREIGN KEY (run_id) REFERENCES pipeline_run(run_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_checkpoint_run ON checkpoint(run_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_checkpoint_run_step ON checkpoint(run_id, step_id);

ALTER TABLE pipeline_run ADD COLUMN forked_from_run_id TEXT DEFAULT '';`,
			Down: `DROP TABLE IF EXISTS checkpoint;
DROP INDEX IF EXISTS idx_checkpoint_run;
DROP INDEX IF EXISTS idx_checkpoint_run_step;`,
		},
		{
			Version:     16,
			Description: "Fix step_state primary key to composite (step_id, pipeline_id) — prevents cross-run collisions",
			Up: `CREATE TABLE IF NOT EXISTS step_state_new (
    step_id TEXT NOT NULL,
    pipeline_id TEXT NOT NULL,
    state TEXT NOT NULL,
    retry_count INTEGER NOT NULL DEFAULT 0,
    started_at INTEGER,
    completed_at INTEGER,
    workspace_path TEXT,
    error_message TEXT,
    visit_count INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (step_id, pipeline_id),
    FOREIGN KEY (pipeline_id) REFERENCES pipeline_state(pipeline_id) ON DELETE CASCADE
);
INSERT OR IGNORE INTO step_state_new SELECT * FROM step_state;
DROP TABLE step_state;
ALTER TABLE step_state_new RENAME TO step_state;
CREATE INDEX IF NOT EXISTS idx_step_pipeline_id ON step_state(pipeline_id);`,
			Down: "",
		},
		{
			Version:     17,
			Description: "Add decision log table for structured decision tracking",
			Up: `CREATE TABLE IF NOT EXISTS decision_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id TEXT NOT NULL,
    step_id TEXT NOT NULL DEFAULT '',
    timestamp INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    category TEXT NOT NULL,
    decision TEXT NOT NULL,
    rationale TEXT NOT NULL DEFAULT '',
    context_json TEXT NOT NULL DEFAULT '{}',
    FOREIGN KEY (run_id) REFERENCES pipeline_run(run_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_decision_run ON decision_log(run_id);
CREATE INDEX IF NOT EXISTS idx_decision_step ON decision_log(run_id, step_id);
CREATE INDEX IF NOT EXISTS idx_decision_category ON decision_log(category);`,
			Down: `DROP TABLE IF EXISTS decision_log;`,
		},
		{
			Version:     18,
			Description: "Add webhook and webhook_deliveries tables for event notification",
			Up: `CREATE TABLE IF NOT EXISTS webhooks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    events TEXT NOT NULL DEFAULT '[]',
    matcher TEXT NOT NULL DEFAULT '',
    headers TEXT NOT NULL DEFAULT '{}',
    secret TEXT NOT NULL DEFAULT '',
    active INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    webhook_id INTEGER NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    run_id TEXT NOT NULL,
    event TEXT NOT NULL,
    status_code INTEGER,
    response_time_ms INTEGER,
    error TEXT,
    delivered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook ON webhook_deliveries(webhook_id);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_run ON webhook_deliveries(run_id);`,
			Down: `DROP TABLE IF EXISTS webhook_deliveries;
DROP TABLE IF EXISTS webhooks;`,
		},
		{
			Version:     19,
			Description: "Add model and adapter columns to event_log for step tracking",
			Up: `ALTER TABLE event_log ADD COLUMN model TEXT;
ALTER TABLE event_log ADD COLUMN adapter TEXT;`,
			Down: `ALTER TABLE event_log DROP COLUMN model;
ALTER TABLE event_log DROP COLUMN adapter;`,
		},
		{
			Version:     20,
			Description: "Add configured_model column to event_log for tier tracking",
			Up:          `ALTER TABLE event_log ADD COLUMN configured_model TEXT DEFAULT '';`,
			Down:        `ALTER TABLE event_log DROP COLUMN configured_model;`,
		},
		{
			Version:     21,
			Description: "Add pipeline_outcome table for persistent outcome tracking",
			Up: `CREATE TABLE IF NOT EXISTS pipeline_outcome (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id TEXT NOT NULL,
    step_id TEXT NOT NULL,
    type TEXT NOT NULL,
    label TEXT NOT NULL,
    value TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (run_id) REFERENCES pipeline_run(run_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_outcome_run ON pipeline_outcome(run_id);
CREATE INDEX IF NOT EXISTS idx_outcome_type_value ON pipeline_outcome(type, value);`,
			Down: `DROP TABLE IF EXISTS pipeline_outcome;`,
		},
		{
			Version:     22,
			Description: "Add orchestration_decisions table for task classification feedback loop",
			Up: `CREATE TABLE IF NOT EXISTS orchestration_decision (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id TEXT NOT NULL,
    input_text TEXT NOT NULL,
    domain TEXT NOT NULL,
    complexity TEXT NOT NULL,
    pipeline_name TEXT NOT NULL,
    model_tier TEXT NOT NULL DEFAULT 'balanced',
    reason TEXT NOT NULL DEFAULT '',
    outcome TEXT NOT NULL DEFAULT 'pending',
    tokens_used INTEGER NOT NULL DEFAULT 0,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    completed_at INTEGER,
    FOREIGN KEY (run_id) REFERENCES pipeline_run(run_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_orchestration_pipeline ON orchestration_decision(pipeline_name);
CREATE INDEX IF NOT EXISTS idx_orchestration_domain ON orchestration_decision(domain, complexity);
CREATE INDEX IF NOT EXISTS idx_orchestration_outcome ON orchestration_decision(outcome);`,
			Down: `DROP TABLE IF EXISTS orchestration_decision;`,
		},
	}
}
