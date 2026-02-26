<script setup lang="ts">
import { withBase } from 'vitepress'
interface Card {
  title: string
  description: string
  link: string
  icon?: string
}

defineProps<{
  cards: Card[]
}>()
</script>

<template>
  <div class="card-grid">
    <a
      v-for="card in cards"
      :key="card.link"
      :href="withBase(card.link)"
      class="card"
    >
      <div class="card-icon" v-if="card.icon">{{ card.icon }}</div>
      <h3 class="card-title">{{ card.title }}</h3>
      <p class="card-description">{{ card.description }}</p>
    </a>
  </div>
</template>

<style scoped>
.card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
  margin: 24px 0;
}

.card {
  display: block;
  padding: 20px;
  background: var(--vp-c-bg-soft);
  border: 1px solid var(--vp-c-divider);
  border-radius: 8px;
  text-decoration: none;
  transition: all 0.2s ease;
}

.card:hover {
  border-color: var(--vp-c-brand-1);
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
  transform: translateY(-2px);
}

.card-icon {
  font-size: 24px;
  margin-bottom: 12px;
}

.card-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--vp-c-text-1);
  margin: 0 0 8px 0;
}

.card:hover .card-title {
  color: var(--vp-c-brand-1);
}

.card-description {
  font-size: 14px;
  color: var(--vp-c-text-2);
  margin: 0;
  line-height: 1.5;
}
</style>
