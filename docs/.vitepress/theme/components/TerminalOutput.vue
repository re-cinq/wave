<script setup>
import { h } from 'vue'

defineProps({
  lines: {
    type: Array,
    required: true
  }
})
</script>

<template>
  <div class="terminal-output">
    <div class="terminal-header">
      <span class="dot red"></span>
      <span class="dot yellow"></span>
      <span class="dot green"></span>
    </div>
    <pre class="terminal-content"><div v-for="(line, i) in lines" :key="i" class="terminal-line">{{ line }}</div></pre>
  </div>
</template>

<style scoped>
.terminal-output {
  background: #1a1a2e;
  border-radius: 8px;
  overflow: hidden;
  margin: 1rem 0;
  font-family: 'SF Mono', 'Fira Code', monospace;
}
.terminal-header {
  background: #2d2d44;
  padding: 8px 12px;
  display: flex;
  gap: 6px;
}
.dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}
.dot.red { background: #ff5f56; }
.dot.yellow { background: #ffbd2e; }
.dot.green { background: #27c93f; }
.terminal-content {
  padding: 12px 16px;
  margin: 0;
  color: #a9b1d6;
  font-size: 13px;
  line-height: 1.6;
}
.terminal-line {
  white-space: pre-wrap;
}
</style>
