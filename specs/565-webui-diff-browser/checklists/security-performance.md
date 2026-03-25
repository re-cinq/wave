# Security & Performance Quality Checklist: WebUI Diff Browser

## Path Sanitization (FR-013)

- [ ] CHK062 - Is the path validation algorithm fully specified — does "reject `..`" mean rejecting any path containing the literal `..` substring, or only `../` traversal sequences? [Clarity]
- [ ] CHK063 - Is it specified whether symlinks within the repository are followed or rejected during diff computation? [Coverage]
- [ ] CHK064 - Is URL encoding of the `{path...}` parameter handled — can an attacker bypass validation with `%2e%2e` encoding? [Coverage]
- [ ] CHK065 - Is the maximum path length specified for the `{path...}` parameter to prevent buffer/resource exhaustion? [Completeness]

## Git Subprocess Security

- [ ] CHK066 - Is it specified how git command arguments are constructed to prevent shell injection (e.g., branch names containing shell metacharacters)? [Completeness]
- [ ] CHK067 - Is the git working directory for subprocess execution specified — does it use the project root, or a specific git repo path? [Clarity]
- [ ] CHK068 - Is timeout behavior specified for git subprocesses that hang (e.g., network-mounted repos, corrupted objects)? [Coverage]
- [ ] CHK069 - Is it specified that git commands run in a read-only capacity (no `git checkout`, `git merge`, etc. side effects)? [Completeness]

## Resource Limits

- [ ] CHK070 - Is the 100KB truncation limit (FR-005) applied BEFORE or AFTER the response is marshaled to JSON? [Clarity]
- [ ] CHK071 - Is there a limit on the total number of files returned by the summary endpoint, or is it unbounded? [Completeness]
- [ ] CHK072 - Is the memory budget (SC-004, 200MB) testable — is a test methodology specified, or is it an aspirational goal? [Clarity]
- [ ] CHK073 - Is rate limiting or request throttling specified for the diff endpoints to prevent abuse? [Coverage]

## Performance Criteria Testability

- [ ] CHK074 - Is the 3-second SLA (SC-001) measured from request receipt to response sent, or from user click to content rendered? [Clarity]
- [ ] CHK075 - Is the 1-second SLA (SC-002) specified under what conditions — cold git cache, warm cache, SSD vs HDD? [Clarity]
- [ ] CHK076 - Is the 500ms render SLA (SC-003) measured with or without syntax highlighting applied? [Clarity]
- [ ] CHK077 - Are the performance SLAs (SC-001 through SC-004) specified with a test environment baseline (CPU, RAM, disk type)? [Completeness]
