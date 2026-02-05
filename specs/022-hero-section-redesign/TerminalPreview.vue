<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'

export interface TerminalPreviewProps {
  command: string
  outputLines: string[]
  typingSpeed?: number
  autoplay?: boolean
}

const props = withDefaults(defineProps<TerminalPreviewProps>(), {
  typingSpeed: 50,
  autoplay: true
})

const emit = defineEmits<{
  (e: 'animationStart'): void
  (e: 'animationComplete'): void
}>()

// Animation state
const displayedCommand = ref('')
const visibleOutputLines = ref<number>(0)
const isTyping = ref(false)
const isComplete = ref(false)
const cursorVisible = ref(true)

// Cursor blink effect
let cursorInterval: ReturnType<typeof setInterval> | null = null

const startCursorBlink = () => {
  if (cursorInterval) clearInterval(cursorInterval)
  cursorInterval = setInterval(() => {
    cursorVisible.value = !cursorVisible.value
  }, 530)
}

const stopCursorBlink = () => {
  if (cursorInterval) {
    clearInterval(cursorInterval)
    cursorInterval = null
  }
  cursorVisible.value = true
}

// Type the command character by character
const typeCommand = async (): Promise<void> => {
  isTyping.value = true
  displayedCommand.value = ''

  for (let i = 0; i < props.command.length; i++) {
    displayedCommand.value += props.command[i]
    await sleep(props.typingSpeed)
  }

  isTyping.value = false
}

// Show output lines one by one
const showOutputLines = async (): Promise<void> => {
  // Small pause after command before output appears
  await sleep(300)

  for (let i = 0; i < props.outputLines.length; i++) {
    visibleOutputLines.value = i + 1
    await sleep(100)
  }
}

// Utility sleep function
const sleep = (ms: number): Promise<void> => {
  return new Promise(resolve => setTimeout(resolve, ms))
}

// Main animation sequence
const runAnimation = async (): Promise<void> => {
  // Reset state
  displayedCommand.value = ''
  visibleOutputLines.value = 0
  isComplete.value = false

  emit('animationStart')
  startCursorBlink()

  await typeCommand()
  await showOutputLines()

  stopCursorBlink()
  isComplete.value = true
  emit('animationComplete')
}

// Reset and replay
const replay = (): void => {
  runAnimation()
}

// Computed classes for cursor
const cursorClass = computed(() => ({
  cursor: true,
  'cursor--visible': cursorVisible.value,
  'cursor--typing': isTyping.value
}))

// Get visible output lines
const displayedOutputLines = computed(() => {
  return props.outputLines.slice(0, visibleOutputLines.value)
})

// Lifecycle
onMounted(() => {
  if (props.autoplay) {
    // Small delay before starting animation
    setTimeout(() => {
      runAnimation()
    }, 500)
  }
})

// Watch for autoplay changes
watch(() => props.autoplay, (newVal) => {
  if (newVal && !isTyping.value && !isComplete.value) {
    runAnimation()
  }
})

// Expose replay method for parent components
defineExpose({ replay })
</script>

<template>
  <div class="terminal-preview">
    <div class="terminal-window">
      <!-- macOS-style title bar -->
      <div class="terminal-header">
        <div class="terminal-buttons">
          <span class="terminal-button terminal-button--close"></span>
          <span class="terminal-button terminal-button--minimize"></span>
          <span class="terminal-button terminal-button--maximize"></span>
        </div>
        <div class="terminal-title">wave</div>
        <div class="terminal-spacer"></div>
      </div>

      <!-- Terminal content -->
      <div class="terminal-body">
        <!-- Command line -->
        <div class="terminal-line terminal-line--command">
          <span class="terminal-prompt">$</span>
          <span class="terminal-command">{{ displayedCommand }}</span>
          <span :class="cursorClass"></span>
        </div>

        <!-- Output lines -->
        <TransitionGroup name="output-line" tag="div" class="terminal-output">
          <div
            v-for="(line, index) in displayedOutputLines"
            :key="index"
            class="terminal-line terminal-line--output"
          >
            <span v-html="formatLine(line)"></span>
          </div>
        </TransitionGroup>
      </div>
    </div>

    <!-- Replay button (shown when animation is complete) -->
    <button
      v-if="isComplete"
      class="terminal-replay"
      @click="replay"
      aria-label="Replay animation"
    >
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/>
        <path d="M3 3v5h5"/>
      </svg>
    </button>
  </div>
</template>

<script lang="ts">
// Helper function to format output lines with color codes
function formatLine(line: string): string {
  // Support basic ANSI-style color markers
  // Format: [color:text] e.g., [green:SUCCESS] or [blue:step-1]
  return line
    .replace(/\[green:([^\]]+)\]/g, '<span class="text-green">$1</span>')
    .replace(/\[blue:([^\]]+)\]/g, '<span class="text-blue">$1</span>')
    .replace(/\[yellow:([^\]]+)\]/g, '<span class="text-yellow">$1</span>')
    .replace(/\[red:([^\]]+)\]/g, '<span class="text-red">$1</span>')
    .replace(/\[cyan:([^\]]+)\]/g, '<span class="text-cyan">$1</span>')
    .replace(/\[dim:([^\]]+)\]/g, '<span class="text-dim">$1</span>')
    .replace(/\[bold:([^\]]+)\]/g, '<span class="text-bold">$1</span>')
}
</script>

<style scoped>
.terminal-preview {
  position: relative;
  width: 100%;
  max-width: 700px;
  margin: 0 auto;
}

.terminal-window {
  background: #1a1b26;
  border-radius: 12px;
  overflow: hidden;
  box-shadow:
    0 0 0 1px rgba(255, 255, 255, 0.1),
    0 0 30px rgba(99, 102, 241, 0.15),
    0 0 60px rgba(99, 102, 241, 0.1);
  font-family: 'SF Mono', 'Fira Code', 'JetBrains Mono', 'Cascadia Code', Consolas, monospace;
}

/* Subtle glow effect on the border */
.terminal-window::before {
  content: '';
  position: absolute;
  inset: -1px;
  border-radius: 13px;
  padding: 1px;
  background: linear-gradient(
    135deg,
    rgba(99, 102, 241, 0.4),
    rgba(99, 102, 241, 0.1) 50%,
    rgba(99, 102, 241, 0.4)
  );
  -webkit-mask:
    linear-gradient(#fff 0 0) content-box,
    linear-gradient(#fff 0 0);
  -webkit-mask-composite: xor;
  mask-composite: exclude;
  pointer-events: none;
  z-index: 1;
}

/* Title bar */
.terminal-header {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  background: #24253a;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.terminal-buttons {
  display: flex;
  gap: 8px;
}

.terminal-button {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  transition: opacity 0.2s ease;
}

.terminal-button--close {
  background: #ff5f56;
}

.terminal-button--minimize {
  background: #ffbd2e;
}

.terminal-button--maximize {
  background: #27c93f;
}

.terminal-title {
  flex: 1;
  text-align: center;
  font-size: 13px;
  color: rgba(255, 255, 255, 0.5);
  font-weight: 500;
}

.terminal-spacer {
  width: 52px; /* Balance the buttons on the left */
}

/* Terminal body */
.terminal-body {
  padding: 20px 24px;
  min-height: 180px;
}

.terminal-line {
  line-height: 1.7;
  white-space: pre-wrap;
  word-break: break-word;
}

.terminal-line--command {
  display: flex;
  align-items: center;
  color: #c0caf5;
  font-size: 14px;
}

.terminal-prompt {
  color: #7aa2f7;
  margin-right: 10px;
  font-weight: 600;
}

.terminal-command {
  color: #c0caf5;
}

/* Cursor */
.cursor {
  display: inline-block;
  width: 8px;
  height: 18px;
  background: #7aa2f7;
  margin-left: 2px;
  vertical-align: text-bottom;
  opacity: 0;
  transition: opacity 0.1s ease;
}

.cursor--visible {
  opacity: 1;
}

.cursor--typing {
  animation: none;
}

/* Output lines */
.terminal-output {
  margin-top: 12px;
}

.terminal-line--output {
  color: #9aa5ce;
  font-size: 13px;
  padding: 2px 0;
}

/* Output line transition */
.output-line-enter-active {
  transition: all 0.3s ease;
}

.output-line-enter-from {
  opacity: 0;
  transform: translateY(-8px);
}

/* Color classes for formatted output */
:deep(.text-green) {
  color: #9ece6a;
}

:deep(.text-blue) {
  color: #7aa2f7;
}

:deep(.text-yellow) {
  color: #e0af68;
}

:deep(.text-red) {
  color: #f7768e;
}

:deep(.text-cyan) {
  color: #7dcfff;
}

:deep(.text-dim) {
  color: #565f89;
}

:deep(.text-bold) {
  font-weight: 600;
  color: #c0caf5;
}

/* Replay button */
.terminal-replay {
  position: absolute;
  bottom: 16px;
  right: 16px;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 8px;
  background: rgba(122, 162, 247, 0.15);
  color: #7aa2f7;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0.7;
  transition: all 0.2s ease;
  z-index: 2;
}

.terminal-replay:hover {
  opacity: 1;
  background: rgba(122, 162, 247, 0.25);
  transform: scale(1.05);
}

.terminal-replay:active {
  transform: scale(0.95);
}

/* Dark theme adjustments - works in both light and dark mode */
/* The terminal intentionally stays dark even in light mode for authenticity */

/* Responsive */
@media (max-width: 640px) {
  .terminal-body {
    padding: 16px 18px;
  }

  .terminal-line--command {
    font-size: 13px;
  }

  .terminal-line--output {
    font-size: 12px;
  }
}
</style>
