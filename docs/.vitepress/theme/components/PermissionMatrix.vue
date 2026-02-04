<script setup lang="ts">
import { ref, computed } from 'vue'
import type { PersonaPermission, PermissionLevel } from '../types'

const props = withDefaults(defineProps<{
  personas: PersonaPermission[]
  showLegend?: boolean
}>(), {
  showLegend: true
})

// Filter and sort state
const searchQuery = ref('')
const sortColumn = ref<'persona' | 'read' | 'write' | 'execute' | 'network'>('persona')
const sortDirection = ref<'asc' | 'desc'>('asc')

// Permission level display mapping
const permissionLabels: Record<PermissionLevel, string> = {
  allow: 'Allow',
  deny: 'Deny',
  conditional: 'Conditional'
}

const permissionIcons: Record<PermissionLevel, string> = {
  allow: '\u2713',  // checkmark
  deny: '\u2717',   // x mark
  conditional: '\u25CF'  // filled circle
}

// Filter personas by search query
const filteredPersonas = computed(() => {
  let result = [...props.personas]

  // Apply search filter
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(p =>
      p.persona.toLowerCase().includes(query) ||
      p.description.toLowerCase().includes(query)
    )
  }

  // Apply sorting
  result.sort((a, b) => {
    let comparison = 0

    if (sortColumn.value === 'persona') {
      comparison = a.persona.localeCompare(b.persona)
    } else {
      const aLevel = a.permissions[sortColumn.value]
      const bLevel = b.permissions[sortColumn.value]
      // Sort order: allow > conditional > deny
      const order: Record<PermissionLevel, number> = { allow: 0, conditional: 1, deny: 2 }
      comparison = order[aLevel] - order[bLevel]
    }

    return sortDirection.value === 'asc' ? comparison : -comparison
  })

  return result
})

// Toggle sort on column click
function toggleSort(column: 'persona' | 'read' | 'write' | 'execute' | 'network') {
  if (sortColumn.value === column) {
    sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortColumn.value = column
    sortDirection.value = 'asc'
  }
}

// Get CSS class for permission level
function getPermissionClass(level: PermissionLevel): string {
  return `permission-${level}`
}

// Get sort indicator for column headers
function getSortIndicator(column: string): string {
  if (sortColumn.value !== column) return ''
  return sortDirection.value === 'asc' ? ' \u25B2' : ' \u25BC'
}
</script>

<template>
  <div class="permission-matrix-container">
    <!-- Search/Filter Bar -->
    <div class="matrix-controls">
      <div class="search-wrapper">
        <input
          v-model="searchQuery"
          type="text"
          placeholder="Filter personas..."
          class="search-input"
        />
      </div>
    </div>

    <!-- Legend -->
    <div v-if="showLegend" class="matrix-legend">
      <div class="legend-item">
        <span class="legend-icon permission-allow">{{ permissionIcons.allow }}</span>
        <span class="legend-label">Allow - Full access granted</span>
      </div>
      <div class="legend-item">
        <span class="legend-icon permission-deny">{{ permissionIcons.deny }}</span>
        <span class="legend-label">Deny - Access blocked</span>
      </div>
      <div class="legend-item">
        <span class="legend-icon permission-conditional">{{ permissionIcons.conditional }}</span>
        <span class="legend-label">Conditional - Scoped/limited access</span>
      </div>
    </div>

    <!-- Permission Matrix Table -->
    <div class="permission-matrix">
      <table>
        <thead>
          <tr>
            <th
              class="sortable"
              @click="toggleSort('persona')"
            >
              Persona{{ getSortIndicator('persona') }}
            </th>
            <th class="description-col">Description</th>
            <th
              class="permission-col sortable"
              @click="toggleSort('read')"
            >
              Read{{ getSortIndicator('read') }}
            </th>
            <th
              class="permission-col sortable"
              @click="toggleSort('write')"
            >
              Write{{ getSortIndicator('write') }}
            </th>
            <th
              class="permission-col sortable"
              @click="toggleSort('execute')"
            >
              Execute{{ getSortIndicator('execute') }}
            </th>
            <th
              class="permission-col sortable"
              @click="toggleSort('network')"
            >
              Network{{ getSortIndicator('network') }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="persona in filteredPersonas" :key="persona.persona">
            <td class="persona-name">
              <code>{{ persona.persona }}</code>
            </td>
            <td class="persona-description">{{ persona.description }}</td>
            <td :class="['permission-cell', getPermissionClass(persona.permissions.read)]">
              <span class="permission-icon">{{ permissionIcons[persona.permissions.read] }}</span>
              <span class="permission-text">{{ permissionLabels[persona.permissions.read] }}</span>
            </td>
            <td :class="['permission-cell', getPermissionClass(persona.permissions.write)]">
              <span class="permission-icon">{{ permissionIcons[persona.permissions.write] }}</span>
              <span class="permission-text">{{ permissionLabels[persona.permissions.write] }}</span>
            </td>
            <td :class="['permission-cell', getPermissionClass(persona.permissions.execute)]">
              <span class="permission-icon">{{ permissionIcons[persona.permissions.execute] }}</span>
              <span class="permission-text">{{ permissionLabels[persona.permissions.execute] }}</span>
            </td>
            <td :class="['permission-cell', getPermissionClass(persona.permissions.network)]">
              <span class="permission-icon">{{ permissionIcons[persona.permissions.network] }}</span>
              <span class="permission-text">{{ permissionLabels[persona.permissions.network] }}</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Empty state -->
    <div v-if="filteredPersonas.length === 0" class="empty-state">
      No personas match your filter criteria.
    </div>
  </div>
</template>

<style scoped>
.permission-matrix-container {
  margin: 24px 0;
}

.matrix-controls {
  display: flex;
  gap: 16px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.search-wrapper {
  flex: 1;
  min-width: 200px;
}

.search-input {
  width: 100%;
  padding: 10px 14px;
  font-size: 14px;
  border: 1px solid var(--vp-c-divider);
  border-radius: 8px;
  background: var(--vp-c-bg);
  color: var(--vp-c-text-1);
  transition: border-color 0.2s ease;
}

.search-input:focus {
  outline: none;
  border-color: var(--wave-primary);
}

.search-input::placeholder {
  color: var(--vp-c-text-3);
}

.matrix-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
  padding: 16px;
  background: var(--vp-c-bg-soft);
  border: 1px solid var(--vp-c-divider);
  border-radius: 8px;
  margin-bottom: 16px;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}

.legend-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: 4px;
  font-weight: 700;
}

.legend-label {
  color: var(--vp-c-text-2);
}

.permission-matrix {
  overflow-x: auto;
}

.permission-matrix table {
  width: 100%;
  border-collapse: collapse;
  font-size: 14px;
  min-width: 700px;
}

.permission-matrix th,
.permission-matrix td {
  padding: 12px 16px;
  text-align: left;
  border: 1px solid var(--vp-c-divider);
}

.permission-matrix th {
  background: var(--vp-c-bg-soft);
  font-weight: 600;
  white-space: nowrap;
}

.permission-matrix th.sortable {
  cursor: pointer;
  user-select: none;
  transition: background-color 0.15s ease;
}

.permission-matrix th.sortable:hover {
  background: var(--vp-c-bg-mute);
}

.permission-col {
  text-align: center;
  width: 100px;
}

.description-col {
  min-width: 200px;
}

.persona-name code {
  background: var(--vp-c-bg-mute);
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 13px;
  font-weight: 600;
}

.persona-description {
  color: var(--vp-c-text-2);
  font-size: 13px;
}

.permission-cell {
  text-align: center;
  font-weight: 600;
}

.permission-icon {
  display: block;
  font-size: 16px;
  margin-bottom: 2px;
}

.permission-text {
  display: block;
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

/* Permission level colors */
.permission-allow {
  color: var(--wave-trust-green, #22c55e);
  background-color: rgba(34, 197, 94, 0.1);
}

.permission-deny {
  color: var(--wave-danger, #ef4444);
  background-color: rgba(239, 68, 68, 0.1);
}

.permission-conditional {
  color: var(--wave-warning, #f59e0b);
  background-color: rgba(245, 158, 11, 0.1);
}

.permission-matrix tbody tr:hover {
  background: var(--vp-c-bg-soft);
}

.empty-state {
  padding: 48px 24px;
  text-align: center;
  color: var(--vp-c-text-2);
  background: var(--vp-c-bg-soft);
  border: 1px solid var(--vp-c-divider);
  border-radius: 8px;
}

/* Responsive adjustments */
@media (max-width: 768px) {
  .matrix-legend {
    flex-direction: column;
    gap: 12px;
  }

  .permission-text {
    display: none;
  }

  .permission-col {
    width: 60px;
  }
}
</style>
