// Package security enforces input validation and access control throughout
// Wave's pipeline execution. It prevents path traversal attacks through
// directory allowlisting and symlink validation, detects prompt injection
// attempts via pattern matching, sanitizes HTML and script content, and
// validates persona configurations. Security violations are logged with
// structured severity levels and violation types for audit trails.
package security
