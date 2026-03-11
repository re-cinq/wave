---
layout: home
---

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'

const pipelineIndex = ref(0)
const fading = ref(false)
let rotationInterval = null

const terminals = [
  {
    command: 'wave run gh-pr-review',
    outputLines: [
      { text: ' ╦ ╦╔═╗╦  ╦╔═╗    Pipeline: gh-pr-review', variant: 'logo' },
      { text: ' ║║║╠═╣╚╗╔╝║╣     Config:   wave.yaml', variant: 'logo' },
      { text: ' ╚╩╝╩ ╩ ╚╝ ╚═╝    Elapsed:  1m 21s', variant: 'logo' },
      { text: ' Pipeline: gh-pr-review', variant: 'meta' },
      { text: ' Config:   wave.yaml', variant: 'meta' },
      { text: ' Elapsed:  1m 21s', variant: 'meta' },
      { text: ' ' },
      { text: ' [████████████████] 100% 4/4 (4 ok)', variant: 'info' },
      { text: ' ' },
      { text: ' ✓ diff-analysis (navigator) (24.0s)', variant: 'success' },
      { text: '    ├─ contract: diff-analysis ✓' },
      { text: '    └─ handover → security-review' },
      { text: ' ✓ security-review (auditor) (19.0s)', variant: 'success' },
      { text: '    ├─ contract: findings ✓' },
      { text: '    └─ handover → quality-review' },
      { text: ' ✓ quality-review (reviewer) (22.0s)', variant: 'success' },
      { text: '    └─ handover → summary' },
      { text: ' ✓ summary (summarizer) (16.0s)', variant: 'success' },
    ]
  },
  {
    command: 'wave run speckit-flow',
    outputLines: [
      { text: ' ╦ ╦╔═╗╦  ╦╔═╗    Pipeline: speckit-flow', variant: 'logo' },
      { text: ' ║║║╠═╣╚╗╔╝║╣     Config:   wave.yaml', variant: 'logo' },
      { text: ' ╚╩╝╩ ╩ ╚╝ ╚═╝    Elapsed:  26m 27s', variant: 'logo' },
      { text: ' Pipeline: speckit-flow', variant: 'meta' },
      { text: ' Config:   wave.yaml', variant: 'meta' },
      { text: ' Elapsed:  26m 27s', variant: 'meta' },
      { text: ' ' },
      { text: ' [██████████░░░░░░] 62% 5/8 (5 ok)', variant: 'info' },
      { text: ' ' },
      { text: ' ✓ specify (implementer) (425.5s)', variant: 'success' },
      { text: '    ├─ artifact: specify-status.json' },
      { text: '    └─ contract: specify-status ✓' },
      { text: ' ✓ clarify (implementer) (205.7s)', variant: 'success' },
      { text: ' ✓ plan (implementer) (400.5s)', variant: 'success' },
      { text: ' ✓ tasks (implementer) (303.7s)', variant: 'success' },
      { text: ' ✓ checklist (implementer) (251.3s)', variant: 'success' },
      { text: ' ● analyze ...', variant: 'info' },
    ]
  },
  {
    command: 'wave run doc-fix',
    outputLines: [
      { text: ' ╦ ╦╔═╗╦  ╦╔═╗    Pipeline: doc-fix', variant: 'logo' },
      { text: ' ║║║╠═╣╚╗╔╝║╣     Config:   wave.yaml', variant: 'logo' },
      { text: ' ╚╩╝╩ ╩ ╚╝ ╚═╝    Elapsed:  4m 38s', variant: 'logo' },
      { text: ' Pipeline: doc-fix', variant: 'meta' },
      { text: ' Config:   wave.yaml', variant: 'meta' },
      { text: ' Elapsed:  4m 38s', variant: 'meta' },
      { text: ' ' },
      { text: ' [████████████████] 100% 3/3 (3 ok)', variant: 'info' },
      { text: ' ' },
      { text: ' ✓ scan-changes (navigator) (45.2s)', variant: 'success' },
      { text: '    ├─ contract: doc-fix-scan ✓' },
      { text: '    └─ handover → analyze' },
      { text: ' ✓ analyze (philosopher) (120.8s)', variant: 'success' },
      { text: '    └─ handover → fix-docs' },
      { text: ' ✓ fix-docs (craftsman) (112.3s)', variant: 'success' },
    ]
  },
  {
    command: 'wave run dead-code',
    outputLines: [
      { text: ' ╦ ╦╔═╗╦  ╦╔═╗    Pipeline: dead-code', variant: 'logo' },
      { text: ' ║║║╠═╣╚╗╔╝║╣     Config:   wave.yaml', variant: 'logo' },
      { text: ' ╚╩╝╩ ╩ ╚╝ ╚═╝    Elapsed:  2m 15s', variant: 'logo' },
      { text: ' Pipeline: dead-code', variant: 'meta' },
      { text: ' Config:   wave.yaml', variant: 'meta' },
      { text: ' Elapsed:  2m 15s', variant: 'meta' },
      { text: ' ' },
      { text: ' [████████████████] 100% 3/3 (3 ok)', variant: 'info' },
      { text: ' ' },
      { text: ' ✓ scan (navigator) (52.3s)', variant: 'success' },
      { text: '    └─ handover → clean' },
      { text: ' ✓ clean (craftsman) (48.7s)', variant: 'success' },
      { text: '    └─ handover → verify' },
      { text: ' ✓ verify (reviewer) (34.1s)', variant: 'success' },
    ]
  },
  {
    command: 'wave run security-scan',
    outputLines: [
      { text: ' ╦ ╦╔═╗╦  ╦╔═╗    Pipeline: security-scan', variant: 'logo' },
      { text: ' ║║║╠═╣╚╗╔╝║╣     Config:   wave.yaml', variant: 'logo' },
      { text: ' ╚╩╝╩ ╩ ╚╝ ╚═╝    Elapsed:  5m 42s', variant: 'logo' },
      { text: ' Pipeline: security-scan', variant: 'meta' },
      { text: ' Config:   wave.yaml', variant: 'meta' },
      { text: ' Elapsed:  5m 42s', variant: 'meta' },
      { text: ' ' },
      { text: ' [████████████████] 100% 3/3 (3 ok)', variant: 'info' },
      { text: ' ' },
      { text: ' ✓ scan (navigator) (98.4s)', variant: 'success' },
      { text: '    ├─ contract: security-scan ✓' },
      { text: '    └─ handover → deep-dive' },
      { text: ' ✓ deep-dive (auditor) (145.2s)', variant: 'success' },
      { text: '    └─ handover → report' },
      { text: ' ✓ report (summarizer) (98.7s)', variant: 'success' },
    ]
  },
]

function rotatePipeline() {
  fading.value = true
  setTimeout(() => {
    pipelineIndex.value = (pipelineIndex.value + 1) % terminals.length
    requestAnimationFrame(() => {
      fading.value = false
    })
  }, 400)
}

onMounted(() => {
  rotationInterval = setInterval(rotatePipeline, 12000)
})

onUnmounted(() => {
  if (rotationInterval) clearInterval(rotationInterval)
})

const heroProps = computed(() => ({
  title: 'Wave — Orchestration for Agent Factories',
  tagline: 'Define your agent factory in code. Scope every persona\'s permissions. Run repeatable AI workflows with just the right amount of guardrails.',
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
    ...terminals[pipelineIndex.value]
  },
  valuePills: [
    { label: 'Scoped Autonomy', link: '/concepts/personas', tooltip: 'Per-persona permission boundaries' },
    { label: 'Contracts', link: '/concepts/contracts', tooltip: 'Validated handoffs between steps' },
    { label: 'Git-Native', link: '/concepts/workspaces', tooltip: 'Real git worktree isolation' },
    { label: 'Auditable', link: '/trust-center/', tooltip: 'Full execution traces' }
  ],
  showBackground: true
}))

const features = [
  {
    icon: 'evolution',
    title: 'Factory-Grade Guardrails',
    description: 'Too loose and agents go rogue. Too tight and they accomplish nothing. Per-persona scoping gives each agent exactly the access it needs — no more, no less.',
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
    icon: 'worktree',
    title: 'Git-Native Workspaces',
    description: 'Steps execute in real git worktrees right inside your repo. No detached folders, no mount hacks — just native git.',
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
    description: '47 built-in pipelines for code review, security scanning, documentation, and issue implementation — ready to plug into your agent factory.',
    link: '/use-cases/'
  }
]
</script>

<HeroSection v-bind="heroProps" :style="{ '--terminal-opacity': fading ? '0' : '1' }" />

<div class="spectrum-section">
  <h2 class="spectrum-heading">Just the right amount of guardrails.</h2>
  <p class="spectrum-lead">Agent factories need boundaries — not to hobble agents, but to make them trustworthy enough to run unsupervised.</p>
  <div class="spectrum-grid">
    <div class="spectrum-card">
      <div class="spectrum-label muted">Too tight</div>
      <p>Approval loops at every step. Agents ask before breathing. Safe on paper — useless in practice. You're still doing the work.</p>
    </div>
    <div class="spectrum-card highlight">
      <div class="spectrum-label">Wave</div>
      <p>Each persona is fully empowered inside its role, hard-constrained outside it. Scoping is declarative, enforced at runtime, and versioned in git.</p>
    </div>
    <div class="spectrum-card">
      <div class="spectrum-label muted">Too loose</div>
      <p>Unconstrained agents with full codebase access. One misread prompt from leaked secrets, deleted files, or broken code in production.</p>
    </div>
  </div>
</div>

<FeatureCards :features="features" />

<div class="blog-callout">
  <p class="callout-eyebrow">From the re:cinq blog</p>
  <blockquote class="callout-quote">"The factory sets boundaries on what's safe to do, not what's allowed."</blockquote>
  <a class="callout-link" href="https://re-cinq.com/blog/building-agent-factories" target="_blank" rel="noopener">Building Agent Factories →</a>
</div>

<style>
.terminal-content {
  opacity: var(--terminal-opacity, 1);
  transition: opacity 0.4s ease;
}

/* Guardrail spectrum section */
.spectrum-section {
  max-width: 1152px;
  margin: 0 auto 72px;
  padding: 0 24px;
}

.spectrum-heading {
  font-size: 2rem;
  font-weight: 700;
  text-align: center;
  margin-bottom: 12px;
  letter-spacing: -0.02em;
}

.spectrum-lead {
  text-align: center;
  color: var(--vp-c-text-2);
  margin: 0 auto 40px;
  max-width: 560px;
  font-size: 1.05rem;
  line-height: 1.6;
}

.spectrum-grid {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  border: 1px solid var(--vp-c-divider);
  border-radius: 12px;
  overflow: hidden;
}

.spectrum-card {
  padding: 32px;
  background: var(--vp-c-bg-soft);
  border-right: 1px solid var(--vp-c-divider);
}

.spectrum-card:last-child {
  border-right: none;
}

.spectrum-card.highlight {
  background: var(--vp-c-brand-soft);
}

.spectrum-label {
  font-size: 0.75rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  margin-bottom: 16px;
  color: var(--vp-c-brand-1);
}

.spectrum-label.muted {
  color: var(--vp-c-text-3);
}

.spectrum-card p {
  color: var(--vp-c-text-2);
  font-size: 0.9rem;
  line-height: 1.65;
  margin: 0;
}

/* Blog callout */
.blog-callout {
  max-width: 720px;
  margin: 0 auto 80px;
  padding: 40px 48px;
  border: 1px solid var(--vp-c-divider);
  border-radius: 12px;
  text-align: center;
  background: var(--vp-c-bg-soft);
}

.callout-eyebrow {
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--vp-c-text-3);
  margin-bottom: 16px;
}

.callout-quote {
  font-size: 1.25rem;
  font-style: italic;
  color: var(--vp-c-text-1);
  margin: 0 0 24px;
  line-height: 1.5;
  border: none;
  padding: 0;
}

.callout-link {
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--vp-c-brand-1);
  text-decoration: none;
}

.callout-link:hover {
  text-decoration: underline;
}

@media (max-width: 768px) {
  .spectrum-grid {
    grid-template-columns: 1fr;
  }

  .spectrum-card {
    border-right: none;
    border-bottom: 1px solid var(--vp-c-divider);
  }

  .spectrum-card:last-child {
    border-bottom: none;
  }

  .blog-callout {
    padding: 32px 24px;
  }
}
</style>
