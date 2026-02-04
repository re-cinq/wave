<script setup lang="ts">
import { computed } from 'vue'
import { useData, useRoute } from 'vitepress'

interface BreadcrumbItem {
  text: string
  link?: string
}

const route = useRoute()
const { site } = useData()

const breadcrumbs = computed<BreadcrumbItem[]>(() => {
  const path = route.path
  const segments = path.split('/').filter(Boolean)

  const items: BreadcrumbItem[] = [
    { text: 'Home', link: '/' }
  ]

  let currentPath = ''

  const sectionLabels: Record<string, string> = {
    'concepts': 'Concepts',
    'guides': 'Guides',
    'reference': 'Reference',
    'use-cases': 'Use Cases',
    'trust-center': 'Trust Center',
    'integrations': 'Integrations',
    'examples': 'Examples'
  }

  segments.forEach((segment, index) => {
    currentPath += `/${segment}`

    // Skip .html extension and index pages
    if (segment.endsWith('.html') || segment === 'index.md') {
      return
    }

    const isLast = index === segments.length - 1
    const label = sectionLabels[segment] || formatSegment(segment)

    items.push({
      text: label,
      link: isLast ? undefined : currentPath + '/'
    })
  })

  return items
})

function formatSegment(segment: string): string {
  return segment
    .replace(/\.html$/, '')
    .replace(/\.md$/, '')
    .split('-')
    .map(word => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ')
}
</script>

<template>
  <nav class="breadcrumb" aria-label="Breadcrumb" v-if="breadcrumbs.length > 1">
    <ol>
      <li v-for="(item, index) in breadcrumbs" :key="index">
        <a v-if="item.link" :href="item.link">{{ item.text }}</a>
        <span v-else class="current" aria-current="page">{{ item.text }}</span>
        <span v-if="index < breadcrumbs.length - 1" class="separator" aria-hidden="true">/</span>
      </li>
    </ol>
  </nav>
</template>

<style scoped>
.breadcrumb {
  margin-bottom: 24px;
  font-size: 14px;
}

.breadcrumb ol {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  list-style: none;
  padding: 0;
  margin: 0;
}

.breadcrumb li {
  display: flex;
  align-items: center;
  gap: 8px;
}

.breadcrumb a {
  color: var(--vp-c-text-2);
  text-decoration: none;
  transition: color 0.15s ease;
}

.breadcrumb a:hover {
  color: var(--vp-c-brand-1);
}

.breadcrumb .current {
  color: var(--vp-c-text-1);
  font-weight: 500;
}

.breadcrumb .separator {
  color: var(--vp-c-divider);
}
</style>
