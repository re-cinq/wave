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
		{
			Version:     23,
			Description: "Add completed_empty to pipeline_run status CHECK constraint",
			Up: `
-- SQLite cannot ALTER CHECK constraints directly, so drop and recreate.
-- The CHECK constraint on status prevented inserting 'completed_empty'.
CREATE TABLE pipeline_run_new (
    run_id TEXT PRIMARY KEY,
    pipeline_name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'completed_empty', 'failed', 'cancelled')),
    input TEXT,
    current_step TEXT,
    total_tokens INTEGER DEFAULT 0,
    started_at INTEGER NOT NULL,
    completed_at INTEGER,
    cancelled_at INTEGER,
    error_message TEXT,
    tags_json TEXT DEFAULT '[]',
    branch_name TEXT DEFAULT '',
    pid INTEGER DEFAULT 0,
    parent_run_id TEXT,
    parent_step_id TEXT,
    forked_from_run_id TEXT DEFAULT ''
);
INSERT INTO pipeline_run_new SELECT * FROM pipeline_run;
DROP TABLE pipeline_run;
ALTER TABLE pipeline_run_new RENAME TO pipeline_run;
CREATE INDEX IF NOT EXISTS idx_run_pipeline ON pipeline_run(pipeline_name);
CREATE INDEX IF NOT EXISTS idx_run_status ON pipeline_run(status);
CREATE INDEX IF NOT EXISTS idx_run_started ON pipeline_run(started_at);
CREATE INDEX IF NOT EXISTS idx_run_tags ON pipeline_run(tags_json);
CREATE INDEX IF NOT EXISTS idx_run_parent ON pipeline_run(parent_run_id);`,
			Down: `
UPDATE pipeline_run SET status = 'completed' WHERE status = 'completed_empty';`,
		},
		{
			Version:     24,
			Description: "Backfill pipeline_run.total_tokens from event_log for legacy runs",
			Up: `UPDATE pipeline_run SET total_tokens = (
    SELECT COALESCE(SUM(el.tokens_used), 0)
    FROM event_log el
    WHERE el.run_id = pipeline_run.run_id AND el.tokens_used > 0
)
WHERE total_tokens = 0
AND status IN ('completed', 'failed', 'cancelled');`,
			Down: "",
		},
		{
			Version:     25,
			Description: "Add description and metadata columns to pipeline_outcome (merge of internal/deliverable into state.OutcomeRecord)",
			Up: `ALTER TABLE pipeline_outcome ADD COLUMN description TEXT NOT NULL DEFAULT '';
ALTER TABLE pipeline_outcome ADD COLUMN metadata TEXT NOT NULL DEFAULT '';`,
			Down: `ALTER TABLE pipeline_outcome DROP COLUMN description;
ALTER TABLE pipeline_outcome DROP COLUMN metadata;`,
		},
		{
			Version:     26,
			Description: "Add last_heartbeat column to pipeline_run for liveness tracking",
			Up: `ALTER TABLE pipeline_run ADD COLUMN last_heartbeat INTEGER NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_run_heartbeat ON pipeline_run(last_heartbeat) WHERE status = 'running';`,
			Down: `DROP INDEX IF EXISTS idx_run_heartbeat;
ALTER TABLE pipeline_run DROP COLUMN last_heartbeat;`,
		},
		{
			Version:     27,
			Description: "Add iterate metadata + run_kind + sub_pipeline_ref to pipeline_run for composition tree rendering (issue #1450)",
			Up: `ALTER TABLE pipeline_run ADD COLUMN iterate_index INTEGER;
ALTER TABLE pipeline_run ADD COLUMN iterate_total INTEGER;
ALTER TABLE pipeline_run ADD COLUMN iterate_mode TEXT NOT NULL DEFAULT '';
ALTER TABLE pipeline_run ADD COLUMN run_kind TEXT NOT NULL DEFAULT '';
ALTER TABLE pipeline_run ADD COLUMN sub_pipeline_ref TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_run_kind ON pipeline_run(run_kind) WHERE run_kind != '';`,
			Down: `DROP INDEX IF EXISTS idx_run_kind;
ALTER TABLE pipeline_run DROP COLUMN sub_pipeline_ref;
ALTER TABLE pipeline_run DROP COLUMN run_kind;
ALTER TABLE pipeline_run DROP COLUMN iterate_mode;
ALTER TABLE pipeline_run DROP COLUMN iterate_total;
ALTER TABLE pipeline_run DROP COLUMN iterate_index;`,
		},
		{
			Version:     28,
			Description: "Drop ontology_usage table (feature removed)",
			Up: `DROP INDEX IF EXISTS idx_ontology_usage_run;
DROP INDEX IF EXISTS idx_ontology_usage_context;
DROP TABLE IF EXISTS ontology_usage;`,
			Down: "",
		},
		{
			Version:     29,
			Description: "Add pipeline_eval table for evolution signal aggregation (epic #1565 PRE-5)",
			Up: `CREATE TABLE IF NOT EXISTS pipeline_eval (
    pipeline_name TEXT NOT NULL,
    run_id TEXT NOT NULL,
    judge_score REAL,
    contract_pass BOOLEAN,
    retry_count INTEGER,
    failure_class TEXT,
    human_override BOOLEAN,
    duration_ms INTEGER,
    cost_dollars REAL,
    recorded_at INTEGER NOT NULL,
    PRIMARY KEY (pipeline_name, run_id)
);
CREATE INDEX IF NOT EXISTS idx_pipeline_eval_recorded ON pipeline_eval(pipeline_name, recorded_at DESC);`,
			Down: `DROP INDEX IF EXISTS idx_pipeline_eval_recorded;
DROP TABLE IF EXISTS pipeline_eval;`,
		},
		{
			Version:     30,
			Description: "Add pipeline_version table for active-version tracking (epic #1565 PRE-5)",
			Up: `CREATE TABLE IF NOT EXISTS pipeline_version (
    pipeline_name TEXT NOT NULL,
    version INTEGER NOT NULL,
    sha256 TEXT NOT NULL,
    yaml_path TEXT NOT NULL,
    active BOOLEAN NOT NULL,
    created_at INTEGER NOT NULL,
    PRIMARY KEY (pipeline_name, version)
);
CREATE INDEX IF NOT EXISTS idx_pipeline_version_active ON pipeline_version(pipeline_name) WHERE active = 1;`,
			Down: `DROP INDEX IF EXISTS idx_pipeline_version_active;
DROP TABLE IF EXISTS pipeline_version;`,
		},
		{
			Version:     31,
			Description: "Add evolution_proposal table for human-approve gate on auto-evolved pipelines (epic #1565 PRE-5)",
			Up: `CREATE TABLE IF NOT EXISTS evolution_proposal (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pipeline_name TEXT NOT NULL,
    version_before INTEGER NOT NULL,
    version_after INTEGER NOT NULL,
    diff_path TEXT NOT NULL,
    reason TEXT NOT NULL,
    signal_summary TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('proposed','approved','rejected','superseded')),
    proposed_at INTEGER NOT NULL,
    decided_at INTEGER,
    decided_by TEXT
);
CREATE INDEX IF NOT EXISTS idx_evolution_proposal_status ON evolution_proposal(status, proposed_at DESC);
CREATE INDEX IF NOT EXISTS idx_evolution_proposal_pipeline ON evolution_proposal(pipeline_name);`,
			Down: `DROP INDEX IF EXISTS idx_evolution_proposal_pipeline;
DROP INDEX IF EXISTS idx_evolution_proposal_status;
DROP TABLE IF EXISTS evolution_proposal;`,
		},
		{
			Version:     32,
			Description: "Add worksource_binding table for issue→pipeline dispatch (epic #1565 PRE-5)",
			Up: `CREATE TABLE IF NOT EXISTS worksource_binding (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    forge TEXT NOT NULL,
    repo TEXT NOT NULL,
    selector TEXT NOT NULL,
    pipeline_name TEXT NOT NULL,
    trigger TEXT NOT NULL CHECK (trigger IN ('on_demand','on_label','on_open','scheduled')),
    config TEXT,
    active BOOLEAN NOT NULL,
    created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_worksource_binding_active ON worksource_binding(forge, repo) WHERE active = 1;`,
			Down: `DROP INDEX IF EXISTS idx_worksource_binding_active;
DROP TABLE IF EXISTS worksource_binding;`,
		},
		{
			Version:     33,
			Description: "Add schedule table for cron-driven pipeline runs (epic #1565 PRE-5)",
			Up: `CREATE TABLE IF NOT EXISTS schedule (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pipeline_name TEXT NOT NULL,
    cron_expr TEXT NOT NULL,
    input_ref TEXT,
    active BOOLEAN NOT NULL,
    next_fire_at INTEGER,
    last_run_id TEXT,
    created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_schedule_due ON schedule(next_fire_at) WHERE active = 1;`,
			Down: `DROP INDEX IF EXISTS idx_schedule_due;
DROP TABLE IF EXISTS schedule;`,
		},
	}
}
