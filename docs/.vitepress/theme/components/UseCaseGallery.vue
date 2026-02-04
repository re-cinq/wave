<script setup lang="ts">
import { ref, computed } from 'vue'
import type { UseCaseGalleryProps, UseCaseCategory, ComplexityLevel } from '../types'

const props = withDefaults(defineProps<UseCaseGalleryProps>(), {
  showFilters: true
})

const selectedCategory = ref<UseCaseCategory | 'all'>('all')
const selectedComplexity = ref<ComplexityLevel | 'all'>('all')

const categories: { value: UseCaseCategory | 'all'; label: string }[] = [
  { value: 'all', label: 'All Categories' },
  { value: 'code-quality', label: 'Code Quality' },
  { value: 'security', label: 'Security' },
  { value: 'documentation', label: 'Documentation' },
  { value: 'testing', label: 'Testing' },
  { value: 'devops', label: 'DevOps' },
  { value: 'onboarding', label: 'Onboarding' }
]

const complexityLevels: { value: ComplexityLevel | 'all'; label: string }[] = [
  { value: 'all', label: 'All Levels' },
  { value: 'beginner', label: 'Beginner' },
  { value: 'intermediate', label: 'Intermediate' },
  { value: 'advanced', label: 'Advanced' }
]

const complexityColors: Record<ComplexityLevel, string> = {
  beginner: '#22c55e',
  intermediate: '#f59e0b',
  advanced: '#ef4444'
}

const categoryIcons: Record<UseCaseCategory, string> = {
  'code-quality': `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>`,
  'security': `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>`,
  'documentation': `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/></svg>`,
  'testing': `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"/></svg>`,
  'devops': `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>`,
  'onboarding': `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>`
}

const filteredUseCases = computed(() => {
  return props.useCases.filter(useCase => {
    const categoryMatch = selectedCategory.value === 'all' || useCase.category === selectedCategory.value
    const complexityMatch = selectedComplexity.value === 'all' || useCase.complexity === selectedComplexity.value
    return categoryMatch && complexityMatch
  })
})

function setCategory(category: UseCaseCategory | 'all') {
  selectedCategory.value = category
}

function setComplexity(complexity: ComplexityLevel | 'all') {
  selectedComplexity.value = complexity
}

function getComplexityColor(complexity: ComplexityLevel): string {
  return complexityColors[complexity]
}

function getCategoryIcon(category: UseCaseCategory): string {
  return categoryIcons[category]
}

function formatComplexity(complexity: ComplexityLevel): string {
  return complexity.charAt(0).toUpperCase() + complexity.slice(1)
}

function formatCategory(category: UseCaseCategory): string {
  return category.split('-').map(word => word.charAt(0).toUpperCase() + word.slice(1)).join(' ')
}
</script>

<template>
  <div class="use-case-gallery">
    <div v-if="props.showFilters" class="filters-section">
      <div class="filter-group">
        <span class="filter-label">Category:</span>
        <div class="filters">
          <button
            v-for="cat in categories"
            :key="cat.value"
            :class="['filter-chip', { active: selectedCategory === cat.value }]"
            @click="setCategory(cat.value)"
          >
            {{ cat.label }}
          </button>
        </div>
      </div>
      <div class="filter-group">
        <span class="filter-label">Complexity:</span>
        <div class="filters">
          <button
            v-for="level in complexityLevels"
            :key="level.value"
            :class="['filter-chip', { active: selectedComplexity === level.value }]"
            @click="setComplexity(level.value)"
          >
            {{ level.label }}
          </button>
        </div>
      </div>
    </div>

    <div class="results-count" v-if="props.showFilters">
      Showing {{ filteredUseCases.length }} of {{ props.useCases.length }} use cases
    </div>

    <div class="gallery-grid">
      <a
        v-for="useCase in filteredUseCases"
        :key="useCase.id"
        :href="useCase.link"
        class="use-case-card"
      >
        <div class="card-header">
          <span class="category-icon" v-html="getCategoryIcon(useCase.category)"></span>
          <span
            class="complexity-badge"
            :style="{ backgroundColor: getComplexityColor(useCase.complexity) }"
          >
            {{ formatComplexity(useCase.complexity) }}
          </span>
        </div>
        <h3>{{ useCase.title }}</h3>
        <p>{{ useCase.description }}</p>
        <div class="card-footer">
          <div class="tags">
            <span class="tag category-tag">{{ formatCategory(useCase.category) }}</span>
            <span v-for="tag in useCase.tags.slice(0, 2)" :key="tag" class="tag">{{ tag }}</span>
          </div>
          <div class="personas" v-if="useCase.personas.length > 0">
            <span class="personas-label">Personas:</span>
            <span class="persona-list">{{ useCase.personas.join(', ') }}</span>
          </div>
        </div>
      </a>
    </div>

    <div v-if="filteredUseCases.length === 0" class="no-results">
      <p>No use cases match your current filters.</p>
      <button class="reset-filters" @click="selectedCategory = 'all'; selectedComplexity = 'all'">
        Reset Filters
      </button>
    </div>
  </div>
</template>

<style scoped>
.use-case-gallery {
  padding: 16px 0;
}

.filters-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
  margin-bottom: 24px;
  padding: 20px;
  background: var(--vp-c-bg-soft);
  border-radius: 12px;
  border: 1px solid var(--vp-c-divider);
}

.filter-group {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
}

.filter-label {
  font-size: 14px;
  font-weight: 600;
  color: var(--vp-c-text-1);
  min-width: 80px;
}

.filters {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.filter-chip {
  padding: 6px 14px;
  font-size: 13px;
  font-weight: 500;
  background: var(--vp-c-bg);
  border: 1px solid var(--vp-c-divider);
  border-radius: 20px;
  cursor: pointer;
  transition: all 0.15s ease;
  color: var(--vp-c-text-2);
}

.filter-chip:hover {
  border-color: var(--vp-c-brand-1);
  color: var(--vp-c-brand-1);
}

.filter-chip.active {
  background: var(--vp-c-brand-1);
  border-color: var(--vp-c-brand-1);
  color: white;
}

.results-count {
  font-size: 14px;
  color: var(--vp-c-text-2);
  margin-bottom: 16px;
}

.gallery-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 20px;
}

.use-case-card {
  display: flex;
  flex-direction: column;
  padding: 24px;
  background: var(--vp-c-bg-soft);
  border: 1px solid var(--vp-c-divider);
  border-radius: 12px;
  text-decoration: none;
  color: inherit;
  transition: all 0.2s ease;
}

.use-case-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
  border-color: var(--vp-c-brand-1);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.category-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  background: var(--vp-c-bg);
  border-radius: 8px;
  color: var(--vp-c-brand-1);
}

.complexity-badge {
  padding: 4px 10px;
  font-size: 11px;
  font-weight: 600;
  color: white;
  border-radius: 12px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.use-case-card h3 {
  font-size: 1.1rem;
  font-weight: 600;
  margin-bottom: 8px;
  color: var(--vp-c-text-1);
}

.use-case-card p {
  color: var(--vp-c-text-2);
  font-size: 14px;
  line-height: 1.5;
  margin-bottom: 16px;
  flex-grow: 1;
}

.card-footer {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: auto;
}

.tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.tag {
  padding: 4px 10px;
  font-size: 11px;
  font-weight: 500;
  background: var(--vp-c-bg);
  border-radius: 4px;
  color: var(--vp-c-text-2);
}

.category-tag {
  background: var(--vp-c-brand-soft);
  color: var(--vp-c-brand-1);
}

.personas {
  font-size: 12px;
  color: var(--vp-c-text-3);
}

.personas-label {
  font-weight: 500;
}

.persona-list {
  font-family: var(--vp-font-family-mono);
}

.no-results {
  text-align: center;
  padding: 48px 24px;
  background: var(--vp-c-bg-soft);
  border-radius: 12px;
  border: 1px dashed var(--vp-c-divider);
}

.no-results p {
  color: var(--vp-c-text-2);
  margin-bottom: 16px;
}

.reset-filters {
  padding: 10px 20px;
  font-size: 14px;
  font-weight: 500;
  background: var(--vp-c-brand-1);
  color: white;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.2s ease;
}

.reset-filters:hover {
  background: var(--vp-c-brand-2);
}

@media (max-width: 640px) {
  .gallery-grid {
    grid-template-columns: 1fr;
  }

  .filter-group {
    flex-direction: column;
    align-items: flex-start;
  }

  .filter-label {
    min-width: auto;
  }
}
</style>
