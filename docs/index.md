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
      { text: ' ╦ ╦╔═╗╦  ╦╔═╗    Pipeline: gh-pr-review' },
      { text: ' ║║║╠═╣╚╗╔╝║╣     Config:   wave.yaml' },
      { text: ' ╚╩╝╩ ╩ ╚╝ ╚═╝    Elapsed:  1m 21s' },
      { text: ' ' },
      { text: ' [█████████████████████████] 100% Step 4/4 (4 ok)', variant: 'info' },
      { text: ' ' },
      { text: ' ✓ diff-analysis (navigator) (24.0s)', variant: 'success' },
      { text: '    ├─ contract: diff-analysis.schema.json ✓ valid' },
      { text: '    └─ handover → security-review' },
      { text: ' ✓ security-review (auditor) (19.0s)', variant: 'success' },
      { text: '    ├─ contract: findings.schema.json ✓ valid' },
      { text: '    └─ handover → quality-review' },
      { text: ' ✓ quality-review (reviewer) (22.0s)', variant: 'success' },
      { text: '    └─ handover → summary' },
      { text: ' ✓ summary (summarizer) (16.0s)', variant: 'success' },
    ]
  },
  {
    command: 'wave run speckit-flow',
    outputLines: [
      { text: ' ╦ ╦╔═╗╦  ╦╔═╗    Pipeline: speckit-flow' },
      { text: ' ║║║╠═╣╚╗╔╝║╣     Config:   wave.yaml' },
      { text: ' ╚╩╝╩ ╩ ╚╝ ╚═╝    Elapsed:  26m 27s' },
      { text: ' ' },
      { text: ' [███████████████░░░░░░░░░░] 62% Step 5/8 (5 ok)', variant: 'info' },
      { text: ' ' },
      { text: ' ✓ specify (implementer) (425.5s)', variant: 'success' },
      { text: '    ├─ artifact: .wave/output/specify-status.json (written)' },
      { text: '    └─ contract: specify-status.schema.json ✓ valid' },
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
      { text: ' ╦ ╦╔═╗╦  ╦╔═╗    Pipeline: doc-fix' },
      { text: ' ║║║╠═╣╚╗╔╝║╣     Config:   wave.yaml' },
      { text: ' ╚╩╝╩ ╩ ╚╝ ╚═╝    Elapsed:  4m 38s' },
      { text: ' ' },
      { text: ' [█████████████████████████] 100% Step 3/3 (3 ok)', variant: 'info' },
      { text: ' ' },
      { text: ' ✓ scan-changes (navigator) (45.2s)', variant: 'success' },
      { text: '    ├─ contract: doc-fix-scan.schema.json ✓ valid' },
      { text: '    └─ handover → analyze' },
      { text: ' ✓ analyze (philosopher) (120.8s)', variant: 'success' },
      { text: '    └─ handover → fix-docs' },
      { text: ' ✓ fix-docs (craftsman) (112.3s)', variant: 'success' },
    ]
  },
  {
    command: 'wave run dead-code',
    outputLines: [
      { text: ' ╦ ╦╔═╗╦  ╦╔═╗    Pipeline: dead-code' },
      { text: ' ║║║╠═╣╚╗╔╝║╣     Config:   wave.yaml' },
      { text: ' ╚╩╝╩ ╩ ╚╝ ╚═╝    Elapsed:  2m 15s' },
      { text: ' ' },
      { text: ' [█████████████████████████] 100% Step 3/3 (3 ok)', variant: 'info' },
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
      { text: ' ╦ ╦╔═╗╦  ╦╔═╗    Pipeline: security-scan' },
      { text: ' ║║║╠═╣╚╗╔╝║╣     Config:   wave.yaml' },
      { text: ' ╚╩╝╩ ╩ ╚╝ ╚═╝    Elapsed:  5m 42s' },
      { text: ' ' },
      { text: ' [█████████████████████████] 100% Step 3/3 (3 ok)', variant: 'info' },
      { text: ' ' },
      { text: ' ✓ scan (navigator) (98.4s)', variant: 'success' },
      { text: '    ├─ contract: security-scan.schema.json ✓ valid' },
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
    ...terminals[pipelineIndex.value]
  },
  valuePills: [
    { label: 'Declarative', link: '/concepts/pipelines', tooltip: 'YAML-based configuration' },
    { label: 'Contracts', link: '/concepts/contracts', tooltip: 'Output validation' },
    { label: 'Git Worktrees', link: '/concepts/workspaces', tooltip: 'Native git worktree isolation' },
    { label: 'Auditable', link: '/trust-center/', tooltip: 'Full execution traces' }
  ],
  showBackground: true
}))

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
    description: 'Built-in pipelines for code review, documentation, and test generation.',
    link: '/use-cases/'
  }
]
</script>

<HeroSection v-bind="heroProps" :style="{ '--terminal-opacity': fading ? '0' : '1' }" />

<FeatureCards :features="features" />

<style>
.terminal-content {
  opacity: var(--terminal-opacity, 1);
  transition: opacity 0.4s ease;
}
</style>
