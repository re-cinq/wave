// fork.go exposes a thin adapter around pipeline.ForkManager so callers
// outside the pipeline package (notably internal/webui) can drive fork and
// fork-point operations without taking a direct dependency on the executor
// package. The runner package already imports internal/pipeline and is the
// canonical bridge between transport layers and the executor.
package runner
