---
layout: home
---

<script setup>
const heroProps = {
  title: 'Wave · AI-as-Code for multi-agent pipelines',
  tagline: 'Define, version, and run AI workflows like you manage infrastructure.',
  primaryAction: {
    text: 'Get Started',
    link: '/quickstart'
  },
  secondaryAction: {
    text: 'View Examples',
    link: '/use-cases/'
  },
  terminal: {
    title: 'wave',
    command: 'wave run code-review "review the auth module"',
    outputLines: [
      { text: '[10:00:01] started   diff-analysis   (navigator)              Starting step', variant: 'muted' },
      { text: '[10:00:25] completed diff-analysis   (navigator)   24s   2.5k Analysis complete', variant: 'success' },
      { text: '[10:00:26] started   security-review (auditor)                Starting step', variant: 'muted' },
      { text: '[10:00:26] started   quality-review  (auditor)                Starting step', variant: 'muted' },
      { text: '[10:00:45] completed security-review (auditor)     19s   1.8k Review complete', variant: 'success' },
      { text: '[10:00:48] completed quality-review  (auditor)     22s   2.1k Review complete', variant: 'success' },
      { text: '[10:00:49] started   summary         (summarizer)             Starting step', variant: 'muted' },
      { text: '[10:01:05] completed summary         (summarizer)  16s   1.2k Summary complete', variant: 'success' },
      { text: '' },
      { text: 'Pipeline code-review completed in 64s', variant: 'success' }
    ]
  },
  valuePills: [
    { label: 'Declarative', link: '/concepts/pipelines', tooltip: 'YAML-based configuration' },
    { label: 'Contracts', link: '/concepts/contracts', tooltip: 'Output validation' },
    { label: 'Isolation', link: '/concepts/workspaces', tooltip: 'Fresh memory each step' },
    { label: 'Auditable', link: '/trust-center/', tooltip: 'Full execution traces' }
  ],
  showBackground: true
}

const features = [
  {
    icon: 'evolution',
    title: 'The Next X-as-Code',
    description: 'Infrastructure → Policy → Security → AI. Bring the same rigor to AI that transformed how you manage infrastructure.',
    link: '/concepts/ai-as-code'
  },
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
    icon: 'audit',
    title: 'Audit Logging',
    description: 'Complete execution traces with credential scrubbing. Full visibility into every pipeline run.',
    link: '/trust-center/'
  },
  {
    icon: 'ready',
    title: 'Ready-to-Run Pipelines',
    description: 'Built-in pipelines for code review, documentation, and test generation.',
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

<div v-pre>

```yaml
kind: WavePipeline
metadata:
  name: code-review
  description: "Automated code review pipeline"

steps:
  - id: analyze
    persona: navigator
    exec:
      source: "Analyze the code changes: {{ input }}"
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

</div>

```bash
wave run code-review "authentication module"
```

Each step runs in complete isolation with fresh memory. Artifacts flow between steps automatically. Contracts validate outputs before the next step begins.

</div>

<style>
.trust-section {
  text-align: center;
  padding: 48px 24px;
  margin: 0 -24px;
}

.trust-heading {
  font-size: 1.75rem;
  font-weight: 600;
  margin-bottom: 24px;
  color: var(--vp-c-text-1);
  border-top: none !important;
  padding-top: 0 !important;
  margin-top: 0 !important;
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
