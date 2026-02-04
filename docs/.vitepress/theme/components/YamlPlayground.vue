<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'

interface ValidationError {
  line: number
  column: number
  message: string
  severity: 'error' | 'warning'
}

interface ValidationResult {
  valid: boolean
  errors: ValidationError[]
  parsed?: any
}

const props = defineProps<{
  initialValue?: string
  placeholder?: string
  readonly?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:value', value: string): void
  (e: 'validation', result: ValidationResult): void
}>()

const defaultYaml = `kind: WavePipeline
metadata:
  name: my-pipeline
  description: "Describe your pipeline"

input:
  source: cli

steps:
  - id: analyze
    persona: navigator
    exec:
      type: prompt
      source: "Analyze: {{ input }}"
    output_artifacts:
      - name: analysis
        path: output/analysis.json
        type: json
`

const yamlContent = ref(props.initialValue || defaultYaml)
const validationResult = ref<ValidationResult>({ valid: true, errors: [] })
const lineNumbers = ref<number[]>([])

// Calculate line numbers
const updateLineNumbers = () => {
  const lines = yamlContent.value.split('\n')
  lineNumbers.value = Array.from({ length: lines.length }, (_, i) => i + 1)
}

// Basic YAML validation
const validateYaml = (content: string): ValidationResult => {
  const errors: ValidationError[] = []
  const lines = content.split('\n')

  // Track indentation levels
  let indentStack: number[] = [0]
  let inMultilineString = false
  let multilineIndent = 0

  lines.forEach((line, index) => {
    const lineNum = index + 1
    const trimmed = line.trim()

    // Skip empty lines and comments
    if (!trimmed || trimmed.startsWith('#')) {
      return
    }

    // Check for multiline string markers
    if (trimmed.includes('|') || trimmed.includes('>')) {
      inMultilineString = true
      multilineIndent = line.search(/\S/)
      return
    }

    // If in multiline string, check if we've exited
    if (inMultilineString) {
      const currentIndent = line.search(/\S/)
      if (currentIndent !== -1 && currentIndent <= multilineIndent) {
        inMultilineString = false
      } else {
        return // Skip validation for multiline content
      }
    }

    // Check for tabs (YAML prefers spaces)
    if (line.includes('\t')) {
      errors.push({
        line: lineNum,
        column: line.indexOf('\t') + 1,
        message: 'YAML uses spaces for indentation, not tabs',
        severity: 'error'
      })
    }

    // Check for inconsistent indentation
    const indent = line.search(/\S/)
    if (indent > 0 && indent % 2 !== 0) {
      errors.push({
        line: lineNum,
        column: 1,
        message: 'Indentation should be a multiple of 2 spaces',
        severity: 'warning'
      })
    }

    // Check for common syntax errors
    // Colon without space
    if (trimmed.match(/^\w+:[^\s]/) && !trimmed.includes('://')) {
      errors.push({
        line: lineNum,
        column: trimmed.indexOf(':') + 2,
        message: 'Missing space after colon',
        severity: 'error'
      })
    }

    // Unclosed quotes
    const singleQuotes = (trimmed.match(/'/g) || []).length
    const doubleQuotes = (trimmed.match(/"/g) || []).length
    if (singleQuotes % 2 !== 0) {
      errors.push({
        line: lineNum,
        column: 1,
        message: 'Unclosed single quote',
        severity: 'error'
      })
    }
    if (doubleQuotes % 2 !== 0) {
      errors.push({
        line: lineNum,
        column: 1,
        message: 'Unclosed double quote',
        severity: 'error'
      })
    }

    // List item without space after dash
    if (trimmed.match(/^-\w/)) {
      errors.push({
        line: lineNum,
        column: trimmed.indexOf('-') + 2,
        message: 'Missing space after list item dash',
        severity: 'error'
      })
    }
  })

  // Wave-specific validation
  if (!content.includes('kind:')) {
    errors.push({
      line: 1,
      column: 1,
      message: 'Missing required field: kind',
      severity: 'error'
    })
  }

  if (!content.includes('metadata:')) {
    errors.push({
      line: 1,
      column: 1,
      message: 'Missing required field: metadata',
      severity: 'warning'
    })
  }

  if (!content.includes('steps:')) {
    errors.push({
      line: 1,
      column: 1,
      message: 'Missing required field: steps (pipelines need at least one step)',
      severity: 'error'
    })
  }

  // Sort errors by line number
  errors.sort((a, b) => a.line - b.line)

  return {
    valid: errors.filter(e => e.severity === 'error').length === 0,
    errors
  }
}

// Debounced validation
let validationTimeout: ReturnType<typeof setTimeout> | null = null

const handleInput = (event: Event) => {
  const target = event.target as HTMLTextAreaElement
  yamlContent.value = target.value
  updateLineNumbers()

  if (validationTimeout) {
    clearTimeout(validationTimeout)
  }

  validationTimeout = setTimeout(() => {
    validationResult.value = validateYaml(yamlContent.value)
    emit('validation', validationResult.value)
    emit('update:value', yamlContent.value)
  }, 300)
}

// Sync scroll between line numbers and textarea
const textareaRef = ref<HTMLTextAreaElement | null>(null)
const lineNumbersRef = ref<HTMLElement | null>(null)

const syncScroll = () => {
  if (textareaRef.value && lineNumbersRef.value) {
    lineNumbersRef.value.scrollTop = textareaRef.value.scrollTop
  }
}

// Get error lines for highlighting
const errorLines = computed(() => {
  return new Set(validationResult.value.errors.map(e => e.line))
})

// Copy to clipboard
const copied = ref(false)
const copyToClipboard = async () => {
  try {
    await navigator.clipboard.writeText(yamlContent.value)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  } catch (err) {
    console.error('Failed to copy:', err)
  }
}

// Reset to initial value
const reset = () => {
  yamlContent.value = props.initialValue || defaultYaml
  updateLineNumbers()
  validationResult.value = validateYaml(yamlContent.value)
}

onMounted(() => {
  updateLineNumbers()
  validationResult.value = validateYaml(yamlContent.value)
})

watch(() => props.initialValue, (newValue) => {
  if (newValue) {
    yamlContent.value = newValue
    updateLineNumbers()
    validationResult.value = validateYaml(yamlContent.value)
  }
})
</script>

<template>
  <div class="yaml-playground">
    <div class="editor-pane">
      <div class="pane-header">
        <span>YAML Editor</span>
        <div class="header-actions">
          <button class="action-btn" @click="reset" title="Reset">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"></path>
              <path d="M3 3v5h5"></path>
            </svg>
          </button>
          <button class="action-btn" :class="{ copied }" @click="copyToClipboard" title="Copy">
            <svg v-if="!copied" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
              <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
            </svg>
            <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="20 6 9 17 4 12"></polyline>
            </svg>
          </button>
        </div>
      </div>
      <div class="editor-content">
        <div class="line-numbers" ref="lineNumbersRef">
          <div
            v-for="num in lineNumbers"
            :key="num"
            class="line-number"
            :class="{ 'has-error': errorLines.has(num) }"
          >
            {{ num }}
          </div>
        </div>
        <textarea
          ref="textareaRef"
          :value="yamlContent"
          @input="handleInput"
          @scroll="syncScroll"
          :placeholder="placeholder || 'Enter YAML here...'"
          :readonly="readonly"
          spellcheck="false"
        ></textarea>
      </div>
    </div>

    <div class="output-pane">
      <div class="pane-header">
        <span>Validation</span>
        <span class="status-badge" :class="validationResult.valid ? 'valid' : 'invalid'">
          {{ validationResult.valid ? 'Valid' : 'Issues Found' }}
        </span>
      </div>
      <div class="validation-result" :class="validationResult.valid ? 'valid' : 'error'">
        <div v-if="validationResult.valid && validationResult.errors.length === 0" class="success-message">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path>
            <polyline points="22 4 12 14.01 9 11.01"></polyline>
          </svg>
          <span>YAML syntax is valid</span>
        </div>
        <div v-else class="error-list">
          <div
            v-for="(error, index) in validationResult.errors"
            :key="index"
            class="error-item"
            :class="error.severity"
          >
            <span class="error-location">Line {{ error.line }}</span>
            <span class="error-message">{{ error.message }}</span>
            <span class="error-severity">{{ error.severity }}</span>
          </div>
        </div>

        <div class="yaml-tips" v-if="!validationResult.valid">
          <h4>Quick Tips</h4>
          <ul>
            <li>Use 2 spaces for indentation (not tabs)</li>
            <li>Add a space after colons in key-value pairs</li>
            <li>Add a space after dashes in list items</li>
            <li>Quote strings containing special characters</li>
          </ul>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.yaml-playground {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  margin: 24px 0;
}

@media (max-width: 768px) {
  .yaml-playground {
    grid-template-columns: 1fr;
  }
}

.editor-pane,
.output-pane {
  border: 1px solid var(--vp-c-divider);
  border-radius: 8px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.pane-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: var(--vp-c-bg-soft);
  border-bottom: 1px solid var(--vp-c-divider);
  font-weight: 600;
  font-size: 14px;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  background: transparent;
  border: 1px solid var(--vp-c-divider);
  border-radius: 4px;
  color: var(--vp-c-text-2);
  cursor: pointer;
  transition: all 0.15s ease;
}

.action-btn:hover {
  color: var(--vp-c-text-1);
  border-color: var(--vp-c-brand-1);
}

.action-btn.copied {
  color: var(--wave-trust-green, #27c93f);
  border-color: var(--wave-trust-green, #27c93f);
}

.status-badge {
  font-size: 12px;
  font-weight: 500;
  padding: 4px 10px;
  border-radius: 12px;
}

.status-badge.valid {
  background: rgba(39, 201, 63, 0.1);
  color: var(--wave-trust-green, #27c93f);
}

.status-badge.invalid {
  background: rgba(217, 74, 74, 0.1);
  color: var(--wave-danger, #d94a4a);
}

.editor-content {
  display: flex;
  flex: 1;
  min-height: 350px;
  overflow: hidden;
}

.line-numbers {
  padding: 16px 0;
  background: var(--vp-c-bg-soft);
  border-right: 1px solid var(--vp-c-divider);
  font-family: var(--wave-font-mono, 'SF Mono', 'Fira Code', monospace);
  font-size: 14px;
  line-height: 1.6;
  color: var(--vp-c-text-3);
  user-select: none;
  overflow: hidden;
}

.line-number {
  padding: 0 12px;
  text-align: right;
  min-width: 40px;
}

.line-number.has-error {
  background: rgba(217, 74, 74, 0.15);
  color: var(--wave-danger, #d94a4a);
}

textarea {
  flex: 1;
  width: 100%;
  padding: 16px;
  font-family: var(--wave-font-mono, 'SF Mono', 'Fira Code', monospace);
  font-size: 14px;
  line-height: 1.6;
  border: none;
  background: var(--vp-c-bg);
  color: var(--vp-c-text-1);
  resize: none;
  outline: none;
}

textarea:focus {
  background: var(--vp-c-bg);
}

textarea::placeholder {
  color: var(--vp-c-text-3);
}

.validation-result {
  flex: 1;
  padding: 16px;
  font-family: var(--wave-font-mono, 'SF Mono', 'Fira Code', monospace);
  font-size: 13px;
  overflow-y: auto;
}

.success-message {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--wave-trust-green, #27c93f);
  padding: 12px;
  background: rgba(39, 201, 63, 0.08);
  border-radius: 6px;
}

.error-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.error-item {
  display: grid;
  grid-template-columns: auto 1fr auto;
  gap: 12px;
  padding: 10px 12px;
  background: var(--vp-c-bg-soft);
  border-radius: 6px;
  align-items: center;
}

.error-item.error {
  border-left: 3px solid var(--wave-danger, #d94a4a);
}

.error-item.warning {
  border-left: 3px solid var(--wave-warning, #ffbd2e);
}

.error-location {
  font-weight: 600;
  color: var(--vp-c-text-2);
  white-space: nowrap;
}

.error-message {
  color: var(--vp-c-text-1);
}

.error-severity {
  font-size: 11px;
  font-weight: 500;
  text-transform: uppercase;
  padding: 2px 8px;
  border-radius: 4px;
}

.error-item.error .error-severity {
  background: rgba(217, 74, 74, 0.15);
  color: var(--wave-danger, #d94a4a);
}

.error-item.warning .error-severity {
  background: rgba(255, 189, 46, 0.15);
  color: var(--wave-warning, #e6a700);
}

.yaml-tips {
  margin-top: 20px;
  padding: 16px;
  background: var(--vp-c-bg-soft);
  border-radius: 6px;
}

.yaml-tips h4 {
  margin: 0 0 10px 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--vp-c-text-2);
}

.yaml-tips ul {
  margin: 0;
  padding-left: 20px;
}

.yaml-tips li {
  margin: 4px 0;
  font-size: 12px;
  color: var(--vp-c-text-2);
  font-family: var(--vp-font-family-base);
}
</style>
