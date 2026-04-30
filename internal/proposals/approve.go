// Package proposals carries the shared approval logic for evolution
// proposals. Both the webui handler and the CLI command depend on it so the
// activation sequence (DecideProposal + CreatePipelineVersion) lives in one
// place.
package proposals

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/recinq/wave/internal/state"
)

// Sentinel errors so callers can map outcomes to HTTP status codes / CLI
// exit codes.
var (
	ErrAlreadyDecided   = errors.New("proposal is not in 'proposed' state")
	ErrVersionConflict  = errors.New("pipeline version conflict — concurrent approval?")
	ErrAfterYAMLMissing = errors.New("post-approval yaml file missing")
)

// ApproveResult carries the activation outcome.
type ApproveResult struct {
	NewVersion int
	YAMLPath   string
	SHA256     string
}

// Approve performs the two-step approval: DecideProposal(approved) followed
// by CreatePipelineVersion(active=true). The post-diff yaml file is resolved
// via ResolveAfterYAMLPath; sha256 is computed over its contents.
func Approve(store state.EvolutionStore, rec *state.EvolutionProposalRecord, decidedBy string) (*ApproveResult, error) {
	if store == nil || rec == nil {
		return nil, errors.New("Approve: nil store or record")
	}

	yamlPath, err := ResolveAfterYAMLPath(rec.DiffPath)
	if err != nil {
		return nil, ErrAfterYAMLMissing
	}
	contents, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("read post-diff yaml: %w", err)
	}
	sum := sha256.Sum256(contents)
	sha := hex.EncodeToString(sum[:])

	versions, err := store.ListPipelineVersions(rec.PipelineName)
	if err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}
	nextVersion := 1
	if len(versions) > 0 {
		nextVersion = versions[0].Version + 1
	}

	err = store.ApproveProposalAndActivate(rec.ID, decidedBy, state.PipelineVersionRecord{
		PipelineName: rec.PipelineName,
		Version:      nextVersion,
		SHA256:       sha,
		YAMLPath:     yamlPath,
		Active:       true,
	})
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "not in 'proposed'") {
			return nil, ErrAlreadyDecided
		}
		if strings.Contains(msg, "unique") {
			return nil, ErrVersionConflict
		}
		return nil, fmt.Errorf("approve and activate: %w", err)
	}

	return &ApproveResult{
		NewVersion: nextVersion,
		YAMLPath:   yamlPath,
		SHA256:     sha,
	}, nil
}

// ResolveAfterYAMLPath probes the conventional post-diff yaml file paths
// emitted by pipeline-evolve. Returns the first path that exists on disk.
//
// Order:
//  1. <DiffPath>.after.yaml  (preferred convention)
//  2. <DiffPath>.yaml         (legacy)
//  3. <DiffPath>              (only when DiffPath itself ends in .yaml)
func ResolveAfterYAMLPath(diffPath string) (string, error) {
	if diffPath == "" {
		return "", errors.New("empty diff path")
	}
	candidates := []string{
		diffPath + ".after.yaml",
		diffPath + ".yaml",
	}
	if strings.HasSuffix(diffPath, ".yaml") {
		candidates = append(candidates, diffPath)
	}
	for _, p := range candidates {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p, nil
		}
	}
	return "", fmt.Errorf("no post-diff yaml found for %s", diffPath)
}
