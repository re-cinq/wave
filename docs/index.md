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
    command: 'wave run ops-pr-review',
    outputLines: [
      { text: ' ╦ ╦╔═╗╦  ╦╔═╗    Pipeline: ops-pr-review', variant: 'logo' },
      { text: ' ║║║╠═╣╚╗╔╝║╣     Config:   wave.yaml', variant: 'logo' },
      { text: ' ╚╩╝╩ ╩ ╚╝ ╚═╝    Elapsed:  1m 21s', variant: 'logo' },
      { text: ' Pipeline: ops-pr-review', variant: 'meta' },
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
    command: 'wave run impl-speckit',
    outputLines: [
      { text: ' ╦ ╦╔═╗╦  ╦╔═╗    Pipeline: impl-speckit', variant: 'logo' },
      { text: ' ║║║╠═╣╚╗╔╝║╣     Config:   wave.yaml', variant: 'logo' },
      { text: ' ╚╩╝╩ ╩ ╚╝ ╚═╝    Elapsed:  26m 27s', variant: 'logo' },
      { text: ' Pipeline: impl-speckit', variant: 'meta' },
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
    command: 'wave run audit-security',
    outputLines: [
      { text: ' ╦ ╦╔═╗╦  ╦╔═╗    Pipeline: audit-security', variant: 'logo' },
      { text: ' ║║║╠═╣╚╗╔╝║╣     Config:   wave.yaml', variant: 'logo' },
      { text: ' ╚╩╝╩ ╩ ╚╝ ╚═╝    Elapsed:  5m 42s', variant: 'logo' },
      { text: ' Pipeline: audit-security', variant: 'meta' },
      { text: ' Config:   wave.yaml', variant: 'meta' },
      { text: ' Elapsed:  5m 42s', variant: 'meta' },
      { text: ' ' },
      { text: ' [████████████████] 100% 3/3 (3 ok)', variant: 'info' },
      { text: ' ' },
      { text: ' ✓ scan (navigator) (98.4s)', variant: 'success' },
      { text: '    ├─ contract: audit-security ✓' },
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
  title: 'Wave · Self-Adaptive V&V Orchestration',
  tagline: 'Wave classifies tasks, selects pipelines, validates outputs, and learns from results. Contract-validated. Gate-controlled. Self-correcting.',
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
    { label: 'Self-Adaptive', link: '/guide/vv-paradigm', tooltip: 'Task classification → pipeline selection → execution → feedback loop' },
    { label: 'Contract-Validated', link: '/guide/contracts', tooltip: '11 contract types validate every step boundary' },
    { label: 'Gate-Controlled', link: '/guide/human-gates', tooltip: '4 gate types: approval, timer, PR merge, CI pass' },
    { label: 'Self-Correcting', link: '/guides/vv-patterns', tooltip: 'Rework loops with convergence tracking abort stalled retries' },
    { label: 'Any Agent', link: '/reference/adapters', tooltip: 'Claude, Gemini, OpenCode, Codex — or mix them' },
    { label: 'Git-Native', link: '/concepts/workspaces', tooltip: 'Real worktrees, real branches, real isolation' }
  ],
  showBackground: true
}))

const features = [
  {
    icon: 'evolution',
    title: 'Self-Adaptive Orchestration',
    description: 'Describe a task and Wave classifies it, selects the right pipeline, executes it, and records the outcome. The feedback loop improves routing over time.',
    link: '/guide/vv-paradigm'
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
    description: '82 built-in pipelines for code review, security scanning, documentation, issue implementation, and orchestration — ready to plug into your agent factory.',
    link: '/use-cases/'
  },
  {
    icon: 'gate',
    title: 'Human & Automated Gates',
    description: 'Four gate types — approval, timer, pr_merge, ci_pass — let you pause pipelines for human review, wait on timers, or poll for PR merges and CI checks before proceeding.',
    link: '/guide/human-gates'
  },
  {
    icon: 'thread',
    title: 'Rework Loops & Convergence',
    description: 'When a contract fails, Wave feeds the error back and retries. Convergence tracking detects score plateaus and aborts stalled loops to save tokens.',
    link: '/guides/vv-patterns'
  },
  {
    icon: 'composition',
    title: 'Pipeline Composition',
    description: 'Five composition primitives — sub-pipelines, iterate, branch, loop, and aggregate — let pipelines compose other pipelines for complex multi-stage workflows.',
    link: '/guide/composition'
  },
  {
    icon: 'routing',
    title: 'Automatic Model Routing',
    description: 'A 3-tier routing system — cheapest, balanced, strongest — classifies step complexity by persona and composition usage, routing each step to the right model tier automatically.',
    link: '/guide/model-routing'
  },
  {
    icon: 'meta',
    title: 'Meta-Pipelines',
    description: 'A philosopher persona dynamically generates and executes child pipelines at runtime, with configurable depth (3), step (20), and token (500K) limits.',
    link: '/concepts/pipelines'
  },
  {
    icon: 'dashboard',
    title: 'Web Dashboard',
    description: 'Monitor pipeline runs, visualize step DAGs, browse artifacts, and control execution — with real-time SSE updates and token-based remote auth.',
    link: '/guides/web-dashboard'
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

<div class="vv-section">
  <h2 class="vv-heading">How Wave Verifies Agent Work</h2>
  <p class="vv-lead">A three-layer verification & validation model ensures every pipeline output meets quality, structural, and behavioral requirements.</p>
  <div class="vv-grid">
    <div class="vv-card">
      <div class="vv-layer">Layer 1</div>
      <div class="vv-name">Personas</div>
      <p>Role-scoped agents with controlled tool access, temperature, and git forensics capabilities for each pipeline step.</p>
    </div>
    <div class="vv-card">
      <div class="vv-layer">Layer 2</div>
      <div class="vv-name">Contracts</div>
      <p>11 contract types with rework loops and convergence tracking. Self-correcting steps retry with feedback until quality thresholds are met.</p>
    </div>
    <div class="vv-card">
      <div class="vv-layer">Layer 3</div>
      <div class="vv-name">Gates</div>
      <p>Four checkpoint types — human approval, timed waits, PR merge polling, and CI pass polling — with toast notifications for attention states.</p>
    </div>
  </div>
</div>

<div class="compat-section">
  <h2 class="compat-heading">Any agent. Any forge. One orchestrator.</h2>
  <p class="compat-lead">Wave speaks your agent's language and works with your platform from day one. No lock-in, no migration.</p>

  <div class="compat-label-row">
    <span class="compat-eyebrow">Adapters</span>
  </div>
  <div class="compat-grid">
    <div class="compat-card">
      <div class="compat-name">Claude Code</div>
      <div class="compat-desc">Anthropic's CLI agent. Full tool use, streaming output, headless mode.</div>
    </div>
    <div class="compat-card">
      <div class="compat-name">OpenCode</div>
      <div class="compat-desc">Open-source AI coding agent. Multi-provider model routing, JSON streaming.</div>
    </div>
    <div class="compat-card">
      <div class="compat-name">Gemini Code</div>
      <div class="compat-desc">Google's Gemini CLI. Auto-approve mode, NDJSON streaming, tool use.</div>
    </div>
    <div class="compat-card">
      <div class="compat-name">Codex</div>
      <div class="compat-desc">OpenAI's Codex CLI. Structured output, AGENTS.md-based prompts.</div>
    </div>
  </div>

  <div class="compat-label-row">
    <span class="compat-eyebrow">Forges</span>
  </div>
  <div class="compat-grid">
    <div class="compat-card">
      <div class="compat-name">GitHub</div>
      <div class="compat-desc">Issue triage, PR review, code search, label management, release automation.</div>
    </div>
    <div class="compat-card">
      <div class="compat-name">GitLab</div>
      <div class="compat-desc">Merge request workflows, issue enhancement, project-level automation.</div>
    </div>
    <div class="compat-card">
      <div class="compat-name">Bitbucket</div>
      <div class="compat-desc">Issue analysis, comments, and enhancement via Bitbucket Cloud REST API.</div>
    </div>
    <div class="compat-card">
      <div class="compat-name">Gitea / Forgejo</div>
      <div class="compat-desc">Self-hosted forge support. Issue creation, commenting, and scoping via tea CLI.</div>
    </div>
    <div class="compat-card">
      <div class="compat-name">Codeberg</div>
      <div class="compat-desc">Community-hosted Gitea instance. Full issue and PR workflow support.</div>
    </div>
    <div class="compat-card">
      <div class="compat-name">Local</div>
      <div class="compat-desc">No forge? No problem. Wave works with local git repos and ad-hoc inputs.</div>
    </div>
  </div>

  <p class="compat-cta">Switch adapters at runtime with <code class="compat-code">--adapter</code> and <code class="compat-code">--model</code>. Mix them in a single pipeline.</p>
</div>

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
  max-width: 720px;
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

/* V&V section */
.vv-section {
  max-width: 1152px;
  margin: 0 auto 72px;
  padding: 0 24px;
}

.vv-heading {
  font-size: 2rem;
  font-weight: 700;
  text-align: center;
  margin-bottom: 12px;
  letter-spacing: -0.02em;
}

.vv-lead {
  text-align: center;
  color: var(--vp-c-text-2);
  margin: 0 auto 40px;
  max-width: 720px;
  font-size: 1.05rem;
  line-height: 1.6;
}

.vv-grid {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  border: 1px solid var(--vp-c-divider);
  border-radius: 12px;
  overflow: hidden;
}

.vv-card {
  padding: 32px;
  background: var(--vp-c-bg-soft);
  border-right: 1px solid var(--vp-c-divider);
}

.vv-card:last-child {
  border-right: none;
}

.vv-layer {
  font-size: 0.75rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  margin-bottom: 8px;
  color: var(--vp-c-text-3);
}

.vv-name {
  font-size: 1.1rem;
  font-weight: 700;
  color: var(--vp-c-brand-1);
  margin-bottom: 12px;
}

.vv-card p {
  color: var(--vp-c-text-2);
  font-size: 0.9rem;
  line-height: 1.65;
  margin: 0;
}

/* Compatibility section */
.compat-section {
  max-width: 1152px;
  margin: 0 auto 72px;
  padding: 0 24px;
}

.compat-heading {
  font-size: 2rem;
  font-weight: 700;
  text-align: center;
  margin-bottom: 12px;
  letter-spacing: -0.02em;
}

.compat-lead {
  text-align: center;
  color: var(--vp-c-text-2);
  margin: 0 auto 48px;
  max-width: 720px;
  font-size: 1.05rem;
  line-height: 1.6;
}

.compat-label-row {
  margin-bottom: 12px;
}

.compat-eyebrow {
  font-size: 0.75rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--vp-c-text-3);
}

.compat-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  border: 1px solid var(--vp-c-divider);
  border-radius: 12px;
  overflow: hidden;
  margin-bottom: 32px;
}

.compat-card {
  padding: 24px;
  background: var(--vp-c-bg-soft);
  border-right: 1px solid var(--vp-c-divider);
  border-bottom: 1px solid var(--vp-c-divider);
}

.compat-card:nth-child(4n) {
  border-right: none;
}

.compat-card:nth-last-child(-n+4) {
  border-bottom: none;
}

.compat-name {
  font-size: 0.95rem;
  font-weight: 700;
  color: var(--vp-c-text-1);
  margin-bottom: 8px;
}

.compat-desc {
  font-size: 0.825rem;
  color: var(--vp-c-text-2);
  line-height: 1.55;
  margin: 0;
}

.compat-cta {
  text-align: center;
  color: var(--vp-c-text-2);
  font-size: 0.9rem;
  margin: 0;
}

.compat-code {
  font-family: var(--vp-font-family-mono);
  background: var(--vp-c-bg-soft);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 0.825rem;
  color: var(--vp-c-brand-1);
}

@media (max-width: 768px) {
  .compat-grid {
    grid-template-columns: 1fr 1fr;
  }

  .compat-card:nth-child(2n) {
    border-right: none;
  }

  .compat-card:nth-last-child(-n+2) {
    border-bottom: none;
  }
}

@media (max-width: 480px) {
  .compat-grid {
    grid-template-columns: 1fr;
  }

  .compat-card {
    border-right: none;
  }

  .compat-card:last-child {
    border-bottom: none;
  }
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

  .vv-grid {
    grid-template-columns: 1fr;
  }

  .vv-card {
    border-right: none;
    border-bottom: 1px solid var(--vp-c-divider);
  }

  .vv-card:last-child {
    border-bottom: none;
  }

  .blog-callout {
    padding: 32px 24px;
  }
}
</style>
