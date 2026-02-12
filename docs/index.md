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
      { text: '[10:00:01] → diff-analysis (navigator)', variant: 'muted' },
      { text: '[10:00:25] ✓ diff-analysis completed (24.0s, 2.5k tokens)', variant: 'success' },
      { text: '[10:00:26] → security-review (auditor)', variant: 'muted' },
      { text: '[10:00:26] → quality-review (auditor)', variant: 'muted' },
      { text: '[10:00:45] ✓ security-review completed (19.0s, 1.8k tokens)', variant: 'success' },
      { text: '[10:00:48] ✓ quality-review completed (22.0s, 2.1k tokens)', variant: 'success' },
      { text: '[10:00:49] → summary (summarizer)', variant: 'muted' },
      { text: '[10:01:05] ✓ summary completed (16.0s, 1.2k tokens)', variant: 'success' },
      { text: '' },
      { text: "  ✓ Pipeline 'code-review' completed successfully (64s)", variant: 'success' }
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

</script>

<HeroSection v-bind="heroProps" />

<FeatureCards :features="features" />
