<script setup lang="ts">
import { ref, computed } from 'vue'
import type { TrustSignalsProps, TrustBadge, ComplianceStatus } from '../types'

const props = withDefaults(defineProps<TrustSignalsProps>(), {
  badges: () => []
})

// Default badges for Trust Center when no props provided
const defaultBadges: TrustBadge[] = [
  {
    name: 'Ephemeral Isolation',
    status: 'certified',
    description: 'Fresh memory each step',
    link: '/concepts/workspaces'
  },
  {
    name: 'Deny-First Permissions',
    status: 'certified',
    description: 'Persona-scoped tool access',
    link: '/concepts/personas'
  },
  {
    name: 'Schema Validation',
    status: 'certified',
    description: 'Output contracts enforced',
    link: '/concepts/contracts'
  },
  {
    name: 'Audit Logging',
    status: 'certified',
    description: 'Full execution traces',
    link: '/guides/audit-logging'
  }
]

const activeBadges = computed(() => {
  return props.badges.length > 0 ? props.badges : defaultBadges
})

function getStatusClass(badge: TrustBadge): string {
  return badge.status
}

function getStatusLabel(status: ComplianceStatus): string {
  switch (status) {
    case 'certified':
      return 'Compliant'
    case 'in-progress':
      return 'In Progress'
    case 'planned':
      return 'Planned'
    default:
      return status
  }
}
</script>

<template>
  <div class="trust-signals-container">
    <div class="trust-signals-header">
      <h3>Security Guarantees</h3>
      <p class="trust-signals-subtitle">Built-in protections at every layer</p>
    </div>
    <div class="trust-signals">
      <a
        v-for="badge in activeBadges"
        :key="badge.name"
        :href="badge.link"
        class="trust-badge"
        :class="getStatusClass(badge)"
      >
        <span class="status-dot"></span>
        <div class="badge-content">
          <span class="badge-name">{{ badge.name }}</span>
          <span class="badge-status-label">{{ getStatusLabel(badge.status) }}</span>
        </div>
        <span v-if="badge.description" class="badge-description">{{ badge.description }}</span>
      </a>
    </div>
  </div>
</template>

<style scoped>
.trust-signals-container {
  margin: 2rem 0;
  padding: 1.5rem;
  background: var(--vp-c-bg-soft);
  border-radius: 12px;
  border: 1px solid var(--vp-c-divider);
}

.trust-signals-header {
  margin-bottom: 1.5rem;
  text-align: center;
}

.trust-signals-header h3 {
  margin: 0 0 0.25rem 0;
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--vp-c-text-1);
}

.trust-signals-subtitle {
  margin: 0;
  font-size: 0.875rem;
  color: var(--vp-c-text-2);
}

.trust-signals {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
}

.trust-badge {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 1rem;
  background: var(--vp-c-bg);
  border: 1px solid var(--vp-c-divider);
  border-radius: 8px;
  text-decoration: none;
  cursor: pointer;
  transition: all 0.2s ease;
}

.trust-badge:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

.trust-badge .status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}

.trust-badge.certified {
  border-color: var(--wave-trust-green, #10b981);
}

.trust-badge.certified .status-dot {
  background: var(--wave-trust-green, #10b981);
}

.trust-badge.certified .badge-status-label {
  color: var(--wave-trust-green, #10b981);
}

.trust-badge.in-progress {
  border-color: var(--wave-in-progress, #f59e0b);
}

.trust-badge.in-progress .status-dot {
  background: var(--wave-in-progress, #f59e0b);
}

.trust-badge.in-progress .badge-status-label {
  color: var(--wave-in-progress, #d97706);
}

.trust-badge.planned {
  border-color: var(--vp-c-divider);
}

.trust-badge.planned .status-dot {
  background: var(--vp-c-text-3, #6b7280);
}

.trust-badge.planned .badge-status-label {
  color: var(--vp-c-text-3, #6b7280);
}

.badge-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.badge-name {
  font-weight: 600;
  font-size: 0.95rem;
  color: var(--vp-c-text-1);
}

.badge-status-label {
  font-size: 0.75rem;
  font-weight: 500;
  padding: 0.15rem 0.4rem;
  border-radius: 4px;
  background: var(--vp-c-bg-soft);
}

.badge-description {
  font-size: 0.8rem;
  color: var(--vp-c-text-2);
  line-height: 1.4;
}

@media (max-width: 640px) {
  .trust-signals {
    grid-template-columns: 1fr;
  }
}
</style>
