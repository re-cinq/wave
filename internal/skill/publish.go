package skill

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// PublishOpts configures a publish operation.
type PublishOpts struct {
	Force    bool
	DryRun   bool
	Registry string
}

// PublishResult represents the outcome of publishing one skill.
type PublishResult struct {
	Name       string
	Success    bool
	URL        string
	Digest     string
	Warnings   []string
	Error      string
	Skipped    bool
	SkipReason string
}

// Publisher handles skill publishing to registries.
type Publisher struct {
	store        Store
	lockfilePath string
	registryName string
	lookPath     func(string) (string, error)
}

// NewPublisher creates a new Publisher.
func NewPublisher(store Store, lockfilePath, registryName string, lookPath func(string) (string, error)) *Publisher {
	return &Publisher{
		store:        store,
		lockfilePath: lockfilePath,
		registryName: registryName,
		lookPath:     lookPath,
	}
}

// PublishOne publishes a single skill to the registry.
func (p *Publisher) PublishOne(ctx context.Context, name string, opts PublishOpts) PublishResult {
	result := PublishResult{Name: name}

	// 1. Read skill from store
	s, err := p.store.Read(name)
	if err != nil {
		result.Error = fmt.Sprintf("skill not found: %v", err)
		return result
	}

	// 2. Validate
	report := ValidateForPublish(s)
	for _, w := range report.Warnings {
		result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %s", w.Field, w.Message))
	}
	if !report.Valid() {
		var msgs []string
		for _, e := range report.Errors {
			msgs = append(msgs, fmt.Sprintf("%s: %s", e.Field, e.Message))
		}
		result.Error = "validation failed: " + strings.Join(msgs, "; ")
		return result
	}

	// 3. Classify — warn on wave-specific
	classification := ClassifySkill(s)
	if classification.Tag == TagWaveSpecific && !opts.Force {
		result.Skipped = true
		result.SkipReason = "wave-specific skill (use --force to override)"
		return result
	}

	// 4. Compute digest
	digest, err := ComputeDigest(s)
	if err != nil {
		result.Error = fmt.Sprintf("digest computation failed: %v", err)
		return result
	}
	result.Digest = digest

	// 5. Check lockfile for idempotency
	lf, err := LoadLockfile(p.lockfilePath)
	if err != nil {
		result.Error = fmt.Sprintf("failed to load lockfile: %v", err)
		return result
	}

	existing := lf.FindByName(name)
	if existing != nil && existing.Digest == digest {
		result.Skipped = true
		result.SkipReason = "up-to-date"
		result.Success = true
		return result
	}

	// 6. Dry run — stop before actual publish
	if opts.DryRun {
		result.Success = true
		result.URL = "[dry-run]"
		return result
	}

	// 7. Execute tessl publish
	tesslPath, err := p.lookPath("tessl")
	if err != nil {
		result.Error = "tessl CLI not found: install tessl to publish skills"
		return result
	}

	publishCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(publishCtx, tesslPath, "publish", s.SourcePath)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		detail := fmt.Sprintf("tessl publish failed: %v", err)
		if stderr.Len() > 0 {
			detail += ": " + strings.TrimSpace(stderr.String())
		}
		result.Error = detail
		return result
	}

	// 8. Parse URL from stdout
	result.URL = strings.TrimSpace(stdout.String())

	// 9. Update lockfile
	lf.Upsert(PublishRecord{
		Name:        name,
		Digest:      digest,
		Registry:    p.registryName,
		URL:         result.URL,
		PublishedAt: time.Now(),
	})
	if err := lf.Save(p.lockfilePath); err != nil {
		result.Error = fmt.Sprintf("published but failed to update lockfile: %v", err)
		return result
	}

	result.Success = true
	return result
}

// PublishAll publishes all standalone-eligible skills.
func (p *Publisher) PublishAll(ctx context.Context, opts PublishOpts) ([]PublishResult, error) {
	classifications, err := ClassifyAll(p.store)
	if err != nil {
		return nil, fmt.Errorf("failed to classify skills: %w", err)
	}

	var results []PublishResult
	for _, c := range classifications {
		if c.Tag == TagWaveSpecific && !opts.Force {
			results = append(results, PublishResult{
				Name:       c.Name,
				Skipped:    true,
				SkipReason: "wave-specific",
			})
			continue
		}

		result := p.PublishOne(ctx, c.Name, opts)
		results = append(results, result)
	}
	return results, nil
}
