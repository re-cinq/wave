// Package contract validates pipeline step outputs against structured
// contracts before marking steps as successful. It supports multiple
// validation backends including JSON Schema, TypeScript interfaces, test
// suites, markdown specifications, format checks, non-empty file validation,
// LLM-based judging, source diff validation, and agent review. The package provides
// automatic JSON recovery and repair, configurable retry strategies, and
// detailed failure reporting. Hard validation failures block pipeline
// progression while soft failures log warnings.
package contract
