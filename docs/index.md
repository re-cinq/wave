---
layout: home
---

<script setup>
const heroProps = {
  title: 'AI Pipelines as Code',
  tagline: 'Define multi-step AI workflows in YAML. Run them with validation, isolation, and reproducible results.',
  primaryAction: {
    text: 'Get Started',
    link: '/quickstart'
  },
  secondaryAction: {
    text: 'View Examples',
    link: '/use-cases/'
  }
}

const features = [
  {
    icon: 'pipeline',
    title: 'Pipelines as Code',
    description: 'Define multi-step AI workflows in YAML. Version control them, share them, run them anywhere.',
    link: '/concepts/pipelines'
  },
  {
    icon: 'contract',
    title: 'Contract Validation',
    description: 'Every step validates its output against schemas. Get structured, predictable results every time.',
    link: '/concepts/contracts'
  },
  {
    icon: 'isolation',
    title: 'Step Isolation',
    description: 'Each step runs with fresh memory in an ephemeral workspace. No context bleed between steps.',
    link: '/concepts/workspaces'
  },
  {
    icon: 'ready',
    title: 'Ready-to-Run Pipelines',
    description: 'Built-in pipelines for code review, security audits, documentation, and test generation.',
    link: '/use-cases/'
  }
]

const trustBadges = [
  {
    name: 'Ephemeral Isolation',
    status: 'certified',
    description: 'Fresh memory each step',
    link: '/concepts/workspaces'
  },
  {
    name: 'Schema Validation',
    status: 'certified',
    description: 'Output contracts enforced',
    link: '/concepts/contracts'
  },
  {
    name: 'Audit Logging',
    status: 'certified',
    description: 'Full execution traces',
    link: '/trust-center/'
  }
]
</script>

<HeroSection v-bind="heroProps" />

<FeatureCards :features="features" />

<div class="trust-section">
  <h2 class="trust-heading">Built for Security</h2>
  <TrustSignals :badges="trustBadges" />
  <p class="trust-cta">
    <a href="/trust-center/">Learn more about Wave's security model</a>
  </p>
</div>

<div class="quick-example">

## See Wave in Action

```yaml
kind: WavePipeline
metadata:
  name: code-review
  description: "Automated code review pipeline"

steps:
  - id: analyze
    persona: navigator
    exec:
      source: "Analyze the code changes: {<!-- -->{ input }}"
    output_artifacts:
      - name: analysis
        path: output/analysis.json
        type: json

  - id: review
    persona: auditor
    dependencies: [analyze]
    exec:
      source: "Review for security and quality issues"
    output_artifacts:
      - name: review
        path: output/review.md
        type: markdown
```

```bash
wave run code-review "authentication module"
```

Each step runs in complete isolation with fresh memory. Artifacts flow between steps automatically. Contracts validate outputs before the next step begins.

</div>

<style>
.trust-section {
  text-align: center;
  padding: 48px 24px;
  background: var(--vp-c-bg-soft);
  margin: 0 -24px;
}

.trust-heading {
  font-size: 1.75rem;
  font-weight: 600;
  margin-bottom: 24px;
  color: var(--vp-c-text-1);
}

.trust-cta {
  margin-top: 24px;
}

.trust-cta a {
  color: var(--wave-primary);
  font-weight: 500;
  text-decoration: none;
}

.trust-cta a:hover {
  text-decoration: underline;
}

.quick-example {
  max-width: 800px;
  margin: 0 auto;
  padding: 48px 24px;
}

.quick-example h2 {
  text-align: center;
  margin-bottom: 32px;
}
</style>
