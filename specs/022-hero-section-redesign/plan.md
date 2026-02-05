# Implementation Plan: Hero Section Redesign

**Branch**: `022-hero-section-redesign` | **Date**: 2026-02-05
**Context**: VitePress documentation site with Vue 3 components

## Summary

Redesign the hero section on the Wave documentation landing page to improve visual impact, user engagement, and messaging clarity. The current implementation is minimal - this plan enhances it with better typography, animated elements, responsive design, and clearer call-to-action hierarchy.

## Current State

### Existing Files

| File | Purpose | Size |
|------|---------|------|
| `docs/index.md` | Landing page with hero props | 5.1 KB |
| `docs/.vitepress/theme/components/HeroSection.vue` | Hero component (minimal) | 736 B |
| `docs/.vitepress/theme/types.ts` | TypeScript definitions | 2.7 KB |
| `docs/.vitepress/theme/styles/components.css` | Component styles (hero: lines 42-106) | 9.7 KB |
| `docs/.vitepress/theme/styles/custom.css` | Theme variables | 2.7 KB |
| `docs/.vitepress/theme/index.ts` | Theme entry (already registers HeroSection) | 2.2 KB |

### Current HeroSectionProps Interface

```typescript
export interface HeroSectionProps {
  title: string
  tagline: string
  primaryAction: { text: string; link: string }
  secondaryAction?: { text: string; link: string }
}
```

## Files to Modify (in order of dependency)

### Phase A: Types (Independent)

1. **`docs/.vitepress/theme/types.ts`**
   - Extend `HeroSectionProps` interface with new fields
   - Add new types for hero features/badges if needed

### Phase B: Styles (Independent)

2. **`docs/.vitepress/theme/styles/custom.css`**
   - Add CSS custom properties for hero-specific colors/gradients
   - Add animation keyframes

3. **`docs/.vitepress/theme/styles/components.css`**
   - Enhance hero section styles (lines 42-106)
   - Add responsive breakpoints
   - Add animation classes

### Phase C: Component (Depends on Phase A)

4. **`docs/.vitepress/theme/components/HeroSection.vue`**
   - Implement enhanced template structure
   - Add optional slots for extensibility
   - Implement animations with Vue transitions

### Phase D: Integration (Depends on Phases A, C)

5. **`docs/index.md`**
   - Update heroProps with any new fields
   - Test integration with redesigned component

## New Files to Create

| File | Purpose |
|------|---------|
| `docs/.vitepress/theme/styles/hero.css` | (Optional) Dedicated hero styles if substantial |

**Note**: Creating a separate `hero.css` is optional. If changes to `components.css` remain under 100 lines, keep styles consolidated.

## Step-by-Step Implementation Tasks

### Task 1: Extend TypeScript Types

**File**: `docs/.vitepress/theme/types.ts`

**Changes**:
```typescript
// Update HeroSectionProps (around line 109)
export interface HeroSectionProps {
  title: string
  tagline: string
  subtitle?: string               // NEW: Optional subtitle below title
  primaryAction: {
    text: string
    link: string
    icon?: string                 // NEW: Optional icon
  }
  secondaryAction?: {
    text: string
    link: string
    icon?: string                 // NEW: Optional icon
  }
  badges?: HeroBadge[]            // NEW: Trust/feature badges
  codePreview?: string            // NEW: Optional inline code preview
  animated?: boolean              // NEW: Enable/disable animations
}

// NEW: Badge type for hero section
export interface HeroBadge {
  text: string
  icon?: string
  link?: string
}
```

**Effort**: ~15 minutes

---

### Task 2: Add CSS Custom Properties and Animations

**File**: `docs/.vitepress/theme/styles/custom.css`

**Changes** (append after line 31):
```css
/* Hero Animation Properties */
--hero-animation-duration: 0.8s;
--hero-animation-timing: cubic-bezier(0.16, 1, 0.3, 1);
--hero-gradient-1: linear-gradient(135deg, var(--wave-primary) 0%, var(--wave-accent) 100%);
--hero-gradient-2: linear-gradient(135deg, var(--wave-secondary) 0%, var(--wave-primary) 100%);
```

**Add keyframes** (append at end of file):
```css
/* Hero Animations */
@keyframes hero-fade-up {
  from { opacity: 0; transform: translateY(20px); }
  to { opacity: 1; transform: translateY(0); }
}

@keyframes hero-gradient-shift {
  0%, 100% { background-position: 0% 50%; }
  50% { background-position: 100% 50%; }
}
```

**Effort**: ~10 minutes

---

### Task 3: Enhance Hero Component Styles

**File**: `docs/.vitepress/theme/styles/components.css`

**Replace** lines 42-106 with enhanced styles:

Key enhancements:
- Staggered fade-in animations for title, tagline, actions
- Improved gradient text with animation
- Badge strip styling
- Better mobile responsive scaling
- Hover micro-interactions on buttons
- Optional code preview styling

**Effort**: ~30 minutes

---

### Task 4: Redesign HeroSection Vue Component

**File**: `docs/.vitepress/theme/components/HeroSection.vue`

**Complete rewrite** with:
```vue
<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import type { HeroSectionProps } from '../types'

const props = withDefaults(defineProps<HeroSectionProps>(), {
  secondaryAction: undefined,
  badges: () => [],
  animated: true,
  codePreview: undefined,
  subtitle: undefined
})

const isVisible = ref(false)

onMounted(() => {
  // Trigger animations after mount
  if (props.animated) {
    requestAnimationFrame(() => { isVisible.value = true })
  } else {
    isVisible.value = true
  }
})

const animationClass = computed(() =>
  props.animated ? 'hero--animated' : ''
)
</script>

<template>
  <section
    class="hero-section"
    :class="[animationClass, { 'hero--visible': isVisible }]"
  >
    <!-- Badge strip -->
    <div v-if="props.badges?.length" class="hero-badges">
      <a
        v-for="badge in props.badges"
        :key="badge.text"
        :href="badge.link"
        class="hero-badge"
      >
        <span v-if="badge.icon" class="hero-badge-icon">{{ badge.icon }}</span>
        {{ badge.text }}
      </a>
    </div>

    <!-- Title -->
    <h1 class="hero-title">{{ props.title }}</h1>

    <!-- Subtitle -->
    <p v-if="props.subtitle" class="hero-subtitle">{{ props.subtitle }}</p>

    <!-- Tagline -->
    <p class="hero-tagline">{{ props.tagline }}</p>

    <!-- Code preview -->
    <div v-if="props.codePreview" class="hero-code-preview">
      <code>{{ props.codePreview }}</code>
    </div>

    <!-- Actions -->
    <div class="hero-actions">
      <a :href="props.primaryAction.link" class="btn btn-primary">
        <span v-if="props.primaryAction.icon" class="btn-icon">
          {{ props.primaryAction.icon }}
        </span>
        {{ props.primaryAction.text }}
      </a>
      <a
        v-if="props.secondaryAction"
        :href="props.secondaryAction.link"
        class="btn btn-secondary"
      >
        <span v-if="props.secondaryAction.icon" class="btn-icon">
          {{ props.secondaryAction.icon }}
        </span>
        {{ props.secondaryAction.text }}
      </a>
    </div>
  </section>
</template>

<style scoped>
/* Component-specific overrides only - base styles in components.css */
</style>
```

**Effort**: ~45 minutes

---

### Task 5: Update Landing Page Integration

**File**: `docs/index.md`

**Update heroProps** (lines 6-17) to use new features:
```javascript
const heroProps = {
  title: 'Wave',
  subtitle: 'AI-as-Code',
  tagline: 'Define, version, and run AI workflows like you manage infrastructure.',
  primaryAction: {
    text: 'Get Started',
    link: '/quickstart',
    icon: '→'
  },
  secondaryAction: {
    text: 'View Examples',
    link: '/use-cases/'
  },
  badges: [
    { text: 'v0.1.0', link: '/changelog' },
    { text: 'Open Source', link: 'https://github.com/re-cinq/wave' }
  ],
  animated: true
}
```

**Effort**: ~15 minutes

---

### Task 6: Add Responsive Media Queries

**File**: `docs/.vitepress/theme/styles/components.css`

**Enhance** existing responsive section (lines 470-487):
- Add tablet breakpoint (768px)
- Improve mobile hero sizing
- Ensure badges wrap properly
- Adjust animation timing for mobile

**Effort**: ~20 minutes

---

## Parallel Work Streams

The following can be worked on simultaneously by different contributors:

```
┌─────────────────────────────────────────────────────────────┐
│                     Work Stream Diagram                      │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Stream A (Types)          Stream B (Styles)                 │
│  ─────────────────         ─────────────────                 │
│  Task 1: types.ts          Task 2: custom.css                │
│       │                    Task 3: components.css            │
│       │                         │                            │
│       └────────┬────────────────┘                            │
│                │                                             │
│                ▼                                             │
│         Stream C (Component)                                 │
│         ────────────────────                                 │
│         Task 4: HeroSection.vue                              │
│                │                                             │
│                ▼                                             │
│         Stream D (Integration)                               │
│         ──────────────────────                               │
│         Task 5: index.md                                     │
│         Task 6: Responsive polish                            │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Parallel Assignments

| Stream | Tasks | Can Start | Dependencies |
|--------|-------|-----------|--------------|
| A | Task 1 | Immediately | None |
| B | Tasks 2, 3 | Immediately | None |
| C | Task 4 | After A completes | Task 1 |
| D | Tasks 5, 6 | After C completes | Task 4 |

## Testing Checklist

### Unit Tests

- [ ] TypeScript types compile without errors
- [ ] Props defaults work correctly (animated: true, badges: [])
- [ ] Optional props (subtitle, codePreview) render conditionally

### Visual/Manual Tests

- [ ] Hero displays correctly on desktop (1920px, 1440px, 1280px)
- [ ] Hero displays correctly on tablet (768px, 1024px)
- [ ] Hero displays correctly on mobile (375px, 414px)
- [ ] Animations trigger on page load when `animated: true`
- [ ] Animations disabled when `animated: false`
- [ ] Badges display horizontally, wrap on narrow screens
- [ ] Primary action button has prominent styling
- [ ] Secondary action button has subtle styling
- [ ] Gradient text animates smoothly
- [ ] Dark mode colors are appropriate
- [ ] Links navigate correctly

### Accessibility Tests

- [ ] Heading hierarchy is correct (h1 for title)
- [ ] Color contrast meets WCAG AA (4.5:1 for text)
- [ ] Focus states visible on all interactive elements
- [ ] Animations respect `prefers-reduced-motion`
- [ ] Screen reader announces hero content properly

### Performance Tests

- [ ] No layout shift during animation (CLS < 0.1)
- [ ] Animation runs at 60fps
- [ ] Component hydrates in < 50ms

### Integration Tests

- [ ] VitePress builds without errors: `npm run docs:build`
- [ ] Dev server works: `npm run docs:dev`
- [ ] No console errors in browser
- [ ] Hero props passed correctly from index.md

### Browser Compatibility

- [ ] Chrome (latest)
- [ ] Firefox (latest)
- [ ] Safari (latest)
- [ ] Edge (latest)

## Rollback Plan

### If Component Changes Break Build

1. **Immediate**: Revert `HeroSection.vue` to previous version
   ```bash
   git checkout main -- docs/.vitepress/theme/components/HeroSection.vue
   ```

2. **Keep**: Type and style changes (backward compatible)

### If Style Changes Break Layout

1. **Immediate**: Revert style files
   ```bash
   git checkout main -- docs/.vitepress/theme/styles/components.css
   git checkout main -- docs/.vitepress/theme/styles/custom.css
   ```

### If Type Changes Break Build

1. **Immediate**: Revert types and component together
   ```bash
   git checkout main -- docs/.vitepress/theme/types.ts
   git checkout main -- docs/.vitepress/theme/components/HeroSection.vue
   ```

### Full Rollback

If all changes need reverting:
```bash
git checkout main -- docs/.vitepress/theme/
git checkout main -- docs/index.md
```

### Partial Rollback Strategy

The implementation is designed for partial rollback:

| Layer | Can Rollback Independently | Notes |
|-------|---------------------------|-------|
| Styles | Yes | Existing styles still work |
| Types | No | Component depends on types |
| Component | No | index.md depends on component |
| index.md | Yes | Can use old props format |

### Feature Flags

If needed, add feature flag support:

```javascript
// In index.md
const heroProps = {
  // ... existing props
  animated: import.meta.env.VITE_HERO_ANIMATIONS !== 'false'
}
```

Then disable animations without code changes:
```bash
VITE_HERO_ANIMATIONS=false npm run docs:build
```

## Estimated Effort

| Task | Effort | Assignee |
|------|--------|----------|
| Task 1: Types | 15 min | - |
| Task 2: CSS Props | 10 min | - |
| Task 3: Component Styles | 30 min | - |
| Task 4: Vue Component | 45 min | - |
| Task 5: Integration | 15 min | - |
| Task 6: Responsive | 20 min | - |
| Testing | 30 min | - |
| **Total** | **~2.5 hours** | - |

## Success Criteria

1. Hero section renders with all new features
2. All tests in checklist pass
3. No regressions in existing functionality
4. VitePress build succeeds
5. Animations are smooth and optional
6. Mobile experience is improved
