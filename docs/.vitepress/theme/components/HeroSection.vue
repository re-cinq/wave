<script setup lang="ts">
import type { HeroSectionProps } from '../types'

const props = withDefaults(defineProps<HeroSectionProps>(), {
  secondaryAction: undefined
})
</script>

<template>
  <section class="hero-section">
    <h1>{{ props.title }}</h1>
    <p class="tagline">{{ props.tagline }}</p>
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
    </div>
  </section>
</template>

<style scoped>
/* Styles are in styles/components.css */
</style>
