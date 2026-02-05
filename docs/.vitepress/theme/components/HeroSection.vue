<script setup lang="ts">
import { computed, ref } from 'vue'
import type { HeroSectionProps, TerminalLineVariant, TerminalIcon } from '../types'

const props = withDefaults(defineProps<HeroSectionProps>(), {
  secondaryAction: undefined,
  terminal: undefined,
  valuePills: undefined,
  github: undefined,
  showBackground: undefined
})

// Determine if we should use two-column layout (when terminal is provided)
const isTwoColumn = computed(() => !!props.terminal)

// Determine if background should be shown
const hasBackground = computed(() => {
  if (props.showBackground !== undefined) {
    return props.showBackground
  }
  return isTwoColumn.value
})

// Terminal prompt with fallback
const terminalPrompt = computed(() => props.terminal?.prompt ?? '$')

// Build full terminal command display
const terminalCommand = computed(() => {
  if (!props.terminal) return ''
  return `${terminalPrompt.value} ${props.terminal.command}`
})

// Get shields.io badge URL
const githubBadgeUrl = computed(() => {
  if (!props.github) return ''
  const style = props.github.style ?? 'social'
  return `https://img.shields.io/github/stars/${props.github.repo}?style=${style}`
})

// Get GitHub repo URL
const githubRepoUrl = computed(() => {
  if (!props.github) return ''
  return `https://github.com/${props.github.repo}`
})

// Clipboard state for copy button
const copied = ref(false)

// Copy just the wave command to clipboard
function copyTerminalContent() {
  if (!props.terminal) return

  navigator.clipboard.writeText(props.terminal.command).then(() => {
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  })
}

// Get CSS class for terminal line variant
function getLineVariantClass(variant?: TerminalLineVariant): string {
  if (!variant || variant === 'default') return ''
  return `line-${variant}`
}

// Get icon for terminal line
function getLineIcon(icon?: TerminalIcon): string {
  switch (icon) {
    case 'check': return '\u2713'
    case 'cross': return '\u2717'
    case 'spinner': return '\u25CB'
    case 'arrow': return '\u2192'
    case 'dot': return '\u2022'
    default: return ''
  }
}
</script>

<template>
  <section
    class="hero-section"
    :class="{
      'hero-two-column': isTwoColumn,
      'hero-with-background': hasBackground
    }"
  >
    <!-- Background pattern -->
    <div v-if="hasBackground" class="hero-background-pattern" aria-hidden="true"></div>

    <div class="hero-container">
      <!-- Left column: Content -->
      <div class="hero-content">
        <h1>{{ props.title }}</h1>
        <p class="tagline">{{ props.tagline }}</p>

        <!-- Value proposition pills -->
        <div v-if="props.valuePills && props.valuePills.length > 0" class="hero-pills">
          <template v-for="(pill, index) in props.valuePills" :key="index">
            <a
              v-if="pill.link"
              :href="pill.link"
              class="hero-pill"
              :title="pill.tooltip"
            >
              <span v-if="pill.icon" class="pill-icon">{{ pill.icon }}</span>
              <span class="pill-label">{{ pill.label }}</span>
            </a>
            <span
              v-else
              class="hero-pill"
              :title="pill.tooltip"
            >
              <span v-if="pill.icon" class="pill-icon">{{ pill.icon }}</span>
              <span class="pill-label">{{ pill.label }}</span>
            </span>
          </template>
        </div>

        <!-- CTA buttons and GitHub badge -->
        <div class="hero-actions">
          <a :href="props.primaryAction.link" class="btn btn-primary">
            {{ props.primaryAction.text }}
          </a>
          <a
            v-if="props.secondaryAction"
            :href="props.secondaryAction.link"
            class="btn btn-secondary"
          >
            {{ props.secondaryAction.text }}
          </a>

          <!-- GitHub stars badge -->
          <a
            v-if="props.github"
            :href="githubRepoUrl"
            class="github-badge"
            target="_blank"
            rel="noopener noreferrer"
            aria-label="View on GitHub"
          >
            <img
              :src="githubBadgeUrl"
              alt="GitHub stars"
              loading="lazy"
            />
          </a>
        </div>
      </div>

      <!-- Right column: Terminal preview -->
      <div v-if="props.terminal" class="hero-terminal">
        <div class="terminal-window">
          <div class="terminal-header">
            <div class="terminal-dots">
              <span class="dot red"></span>
              <span class="dot yellow"></span>
              <span class="dot green"></span>
            </div>
            <span v-if="props.terminal.title" class="terminal-title">{{ props.terminal.title }}</span>
            <button
              class="terminal-copy-btn"
              :class="{ copied }"
              @click="copyTerminalContent"
              :aria-label="copied ? 'Copied!' : 'Copy to clipboard'"
            >
              <svg v-if="!copied" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
              </svg>
              <svg v-else xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="20 6 9 17 4 12"></polyline>
              </svg>
            </button>
          </div>
          <div class="terminal-content" role="region" aria-label="Terminal output preview">
            <div class="terminal-line command">{{ terminalCommand }}</div>
            <div class="terminal-line empty"></div>
            <div
              v-for="(line, index) in props.terminal.outputLines"
              :key="index"
              class="terminal-line"
              :class="getLineVariantClass(line.variant)"
            >
              <span v-if="line.icon" class="line-icon">{{ getLineIcon(line.icon) }}</span>
              {{ line.text }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
/* Base hero section - maintains backwards compatibility */
.hero-section {
  position: relative;
  text-align: center;
  padding: 80px 24px;
  max-width: 900px;
  margin: 0 auto;
  overflow: hidden;
}

/* Two-column layout mode */
.hero-section.hero-two-column {
  max-width: 1200px;
  text-align: left;
}

.hero-container {
  position: relative;
  z-index: 1;
}

.hero-two-column .hero-container {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 40px;
  align-items: center;
}

.hero-content h1 {
  font-size: 3.5rem;
  font-weight: 700;
  line-height: 1.1;
  margin-bottom: 16px;
  background: linear-gradient(135deg, var(--wave-primary) 0%, var(--wave-accent) 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.hero-content .tagline {
  font-size: 1.5rem;
  color: var(--vp-c-text-2);
  margin-bottom: 24px;
  max-width: 600px;
  line-height: 1.5;
}

.hero-section:not(.hero-two-column) .hero-content .tagline {
  margin-left: auto;
  margin-right: auto;
}

/* Value proposition pills */
.hero-pills {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 28px;
}

.hero-section:not(.hero-two-column) .hero-pills {
  justify-content: center;
}

.hero-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  font-size: 14px;
  font-weight: 500;
  color: var(--vp-c-text-2);
  background: var(--vp-c-bg-soft);
  border: 1px solid var(--vp-c-divider);
  border-radius: 20px;
  text-decoration: none;
  transition: all 0.2s ease;
}

.hero-pill:hover {
  transform: scale(1.05);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  border-color: var(--wave-primary);
  color: var(--wave-primary);
}

a.hero-pill {
  cursor: pointer;
}

.pill-icon {
  font-size: 16px;
}

/* Hero actions */
.hero-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
}

.hero-section:not(.hero-two-column) .hero-actions {
  justify-content: center;
}

.hero-two-column .hero-actions {
  justify-content: flex-start;
}

.hero-actions .btn {
  padding: 12px 24px;
  font-size: 16px;
  font-weight: 600;
  border-radius: 8px;
  text-decoration: none;
  transition: all 0.2s ease;
}

.hero-actions .btn-primary {
  background: var(--wave-primary);
  color: white;
}

.hero-actions .btn-primary:hover {
  background: var(--wave-primary-dark);
  transform: translateY(-2px);
}

.hero-actions .btn-secondary {
  background: var(--vp-c-bg-soft);
  color: var(--vp-c-text-1);
  border: 1px solid var(--vp-c-divider);
}

.hero-actions .btn-secondary:hover {
  border-color: var(--wave-primary);
  color: var(--wave-primary);
}

.github-badge {
  display: inline-flex;
  align-items: center;
  transition: opacity 0.2s ease;
}

.github-badge:hover {
  opacity: 0.8;
}

.github-badge img {
  height: 24px;
}

/* Terminal preview */
.hero-terminal {
  display: flex;
  justify-content: center;
}

.terminal-window {
  width: 100%;
  max-width: 100%;
  background: #1a1a2e;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.2);
}

.terminal-header {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  background: #2d2d44;
  gap: 12px;
}

.terminal-dots {
  display: flex;
  gap: 8px;
}

.terminal-dots .dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}

.terminal-dots .dot.red { background: #ff5f56; }
.terminal-dots .dot.yellow { background: #ffbd2e; }
.terminal-dots .dot.green { background: #27c93f; }

.terminal-title {
  flex: 1;
  text-align: center;
  font-size: 13px;
  color: #a9b1d6;
  font-family: var(--wave-font-mono, 'SF Mono', 'Fira Code', monospace);
}

.terminal-copy-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 4px 8px;
  background: transparent;
  border: none;
  color: #6b7280;
  cursor: pointer;
  border-radius: 4px;
  transition: all 0.15s ease;
}

.terminal-copy-btn:hover {
  background: rgba(255, 255, 255, 0.1);
  color: #a9b1d6;
}

.terminal-copy-btn.copied {
  color: #27c93f;
}

.terminal-content {
  padding: 16px 20px;
  font-family: var(--wave-font-mono, 'SF Mono', 'Fira Code', monospace);
  font-size: 13px;
  line-height: 1.7;
  color: #a9b1d6;
  min-height: 200px;
  max-height: 320px;
  overflow-x: auto;
  overflow-y: auto;
}

.terminal-line {
  white-space: pre-wrap;
  word-break: break-word;
}

.terminal-line.empty {
  height: 1.7em;
}

.terminal-line.command {
  color: #7dd3fc;
}

.terminal-line .line-icon {
  margin-right: 8px;
}

/* Terminal line variants */
.terminal-line.line-success { color: #27c93f; }
.terminal-line.line-error { color: #ff5f56; }
.terminal-line.line-warning { color: #ffbd2e; }
.terminal-line.line-info { color: #7dd3fc; }
.terminal-line.line-muted { color: #6b7280; }
.terminal-line.line-highlight { color: #c4b5fd; font-weight: 500; }

/* Background pattern */
.hero-background-pattern {
  position: absolute;
  inset: 0;
  z-index: 0;
  background-image: radial-gradient(circle, var(--vp-c-divider) 1px, transparent 1px);
  background-size: 40px 40px;
  opacity: 0.4;
  pointer-events: none;
}

.dark .hero-background-pattern {
  opacity: 0.15;
}

/* Responsive: Tablet */
@media (max-width: 1023px) {
  .hero-two-column .hero-container {
    grid-template-columns: 1fr;
    gap: 40px;
  }

  .hero-section.hero-two-column {
    text-align: center;
  }

  .hero-two-column .hero-pills {
    justify-content: center;
  }

  .hero-two-column .hero-actions {
    justify-content: center;
  }

  .hero-two-column .hero-content .tagline {
    margin-left: auto;
    margin-right: auto;
  }

  .terminal-content {
    max-height: 240px;
  }
}

/* Responsive: Mobile */
@media (max-width: 640px) {
  .hero-section {
    padding: 48px 20px;
  }

  .hero-content h1 {
    font-size: 2.5rem;
  }

  .hero-content .tagline {
    font-size: 1.2rem;
  }

  .hero-pills {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 8px;
  }

  .hero-pill {
    justify-content: center;
    padding: 8px 12px;
    font-size: 13px;
  }

  .hero-actions {
    flex-direction: column;
    width: 100%;
  }

  .hero-actions .btn {
    width: 100%;
    text-align: center;
  }

  .github-badge {
    justify-content: center;
    width: 100%;
  }

  .terminal-window {
    max-width: 100%;
  }

  .terminal-content {
    min-height: 160px;
    max-height: 180px;
    font-size: 12px;
    padding: 12px 16px;
  }
}

/* Accessibility: Reduced motion */
@media (prefers-reduced-motion: reduce) {
  .hero-pill,
  .hero-actions .btn,
  .terminal-copy-btn,
  .github-badge {
    transition: none;
  }

  .hero-pill:hover,
  .hero-actions .btn-primary:hover {
    transform: none;
  }
}
</style>
