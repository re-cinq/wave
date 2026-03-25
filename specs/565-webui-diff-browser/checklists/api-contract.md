# API Contract Quality Checklist: WebUI Changed-Files Browser

## Endpoint Design

- [ ] CHK029 - Are the HTTP methods for both endpoints explicitly stated in the requirements (GET only), and is it clear that POST/PUT/DELETE return 405? [Completeness]
- [ ] CHK030 - Is the `{path...}` wildcard behavior in FR-002 specified for edge cases like empty path, path with query parameters, or URL-encoded characters? [Completeness]
- [ ] CHK031 - Is the Content-Type header specified for API responses (application/json), and is it consistent with existing webui API conventions? [Clarity]
- [ ] CHK032 - Are pagination or result limits specified for the file list endpoint when a run touches hundreds of files? [Completeness]

## Schema Completeness

- [ ] CHK033 - Does the diff-summary-api.json schema include the `message` field as optional (not required) to match the "omitempty" behavior in the data model? [Consistency]
- [ ] CHK034 - Does the file-diff-api.json schema's `old_path` field match the omitempty semantics in the Go struct definition? [Consistency]
- [ ] CHK035 - Are the `binary` field semantics identical between FileSummary (in DiffSummary response) and FileDiff (in single-file response)? [Consistency]
- [ ] CHK036 - Is the `status` enum ("added", "modified", "deleted", "renamed") identical in both API contract schemas? [Consistency]

## Error Handling

- [ ] CHK037 - Is the error response format (`{"error": "..."}`) specified as a requirement, or only mentioned in the edge cases section? [Clarity]
- [ ] CHK038 - Are error HTTP status codes defined for each failure mode (404 for not found, 400 for invalid path, 500 for git errors), or left implicit? [Completeness]
- [ ] CHK039 - Is it specified whether `available: false` responses return HTTP 200 or a 4xx status? [Clarity]
- [ ] CHK040 - Is error behavior specified when the state store is unavailable (database error during RunRecord lookup)? [Coverage]

## Data Integrity

- [ ] CHK041 - Is the relationship between `total_files` and `len(files)` mandated to be consistent in the response schema? [Consistency]
- [ ] CHK042 - Is it specified whether `total_additions` and `total_deletions` include binary files (which have 0/0 counts) in their totals? [Clarity]
- [ ] CHK043 - Is the base branch resolution algorithm's output (the chosen branch name) visible to the API consumer, and is this useful for debugging? [Completeness]
