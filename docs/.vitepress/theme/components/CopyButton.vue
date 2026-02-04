<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{
  code: string
}>()

const copied = ref(false)
const copyTimeout = ref<ReturnType<typeof setTimeout> | null>(null)

async function copyCode() {
  try {
    await navigator.clipboard.writeText(props.code)
    copied.value = true

    if (copyTimeout.value) {
      clearTimeout(copyTimeout.value)
    }

    copyTimeout.value = setTimeout(() => {
      copied.value = false
    }, 2000)
  } catch (err) {
    // Fallback for browsers without clipboard API
    const textarea = document.createElement('textarea')
    textarea.value = props.code
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()

    try {
      document.execCommand('copy')
      copied.value = true

      if (copyTimeout.value) {
        clearTimeout(copyTimeout.value)
      }

      copyTimeout.value = setTimeout(() => {
        copied.value = false
      }, 2000)
    } catch (fallbackErr) {
      console.error('Failed to copy:', fallbackErr)
    } finally {
      document.body.removeChild(textarea)
    }
  }
}
</script>

<template>
  <button
    class="copy-button"
    :class="{ copied }"
    @click="copyCode"
    :aria-label="copied ? 'Copied!' : 'Copy code'"
    :title="copied ? 'Copied!' : 'Copy to clipboard'"
  >
    <svg v-if="!copied" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
    </svg>
    <svg v-else xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <polyline points="20 6 9 17 4 12"></polyline>
    </svg>
    <span>{{ copied ? 'Copied!' : 'Copy' }}</span>
  </button>
</template>

<style scoped>
.copy-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 6px 10px;
  font-size: 12px;
  font-weight: 500;
  color: var(--vp-c-text-2);
  background: var(--vp-c-bg-soft);
  border: 1px solid var(--vp-c-divider);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.copy-button:hover {
  color: var(--vp-c-text-1);
  background: var(--vp-c-bg-mute);
  border-color: var(--vp-c-brand-1);
}

.copy-button.copied {
  color: #10b981;
  border-color: #10b981;
}

.copy-button svg {
  flex-shrink: 0;
}
</style>
