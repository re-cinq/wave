// Package contract validates pipeline step outputs against structured
// contracts before marking steps as successful. It supports multiple
// validation backends including JSON Schema, TypeScript interfaces, test
// suites, markdown specifications, and format checks. The package provides
// automatic JSON recovery and repair, configurable retry strategies, and
// detailed failure reporting. Hard validation failures block pipeline
// progression while soft failures log warnings.
package contract
