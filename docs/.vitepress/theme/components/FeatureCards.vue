<script setup lang="ts">
import type { FeatureCardsProps } from '../types'

const props = defineProps<FeatureCardsProps>()

// SVG icons for each feature type - inline for simplicity
const icons: Record<string, string> = {
  pipeline: `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="M22 12h-4l-3 9L9 3l-3 9H2"/>
  </svg>`,
  contract: `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="M9 12l2 2 4-4"/>
    <path d="M12 2a10 10 0 1 0 0 20 10 10 0 0 0 0-20z"/>
  </svg>`,
  isolation: `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <rect x="3" y="3" width="7" height="7"/>
    <rect x="14" y="3" width="7" height="7"/>
    <rect x="14" y="14" width="7" height="7"/>
    <rect x="3" y="14" width="7" height="7"/>
  </svg>`,
  ready: `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/>
  </svg>`,
  default: `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <circle cx="12" cy="12" r="10"/>
    <line x1="12" y1="8" x2="12" y2="16"/>
    <line x1="8" y1="12" x2="16" y2="12"/>
  </svg>`
}

function getIcon(iconName: string): string {
  return icons[iconName] || icons.default
}
</script>

<template>
  <div class="feature-cards">
    <a
      v-for="feature in props.features"
      :key="feature.title"
      :href="feature.link"
      class="feature-card"
    >
      <div class="icon" v-html="getIcon(feature.icon)"></div>
      <h3>{{ feature.title }}</h3>
      <p>{{ feature.description }}</p>
    </a>
  </div>
</template>

<style scoped>
/* Base styles are in styles/components.css */

.feature-card {
  text-decoration: none;
  color: inherit;
  display: block;
}

.feature-card:hover {
  text-decoration: none;
}
</style>
