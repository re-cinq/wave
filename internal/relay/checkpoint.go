package relay

// Checkpoint validation errors
var ()

// CheckpointFilename is the workspace-relative checkpoint path. It lives
// under .agents/ so that, in worktree workspaces, Wave's bookkeeping never
// lands in the project tree (and in a `git add -A` of the step's changes).
const CheckpointFilename = ".agents/checkpoint.md"
