# Hero Section Redesign Specification

**Spec ID:** 022-hero-section-redesign
**Status:** Draft
**Author:** Wave Team
**Created:** 2026-02-05

---

## 1. Overview

### What We're Building

A redesigned hero section for the Wave documentation site that transforms the current minimal implementation into an engaging, modern hero that effectively communicates Wave's value proposition and drives conversions.

### Why We're Building It

The current hero section is too simple:
- Just a title, tagline, and two buttons
- No visual demonstration of the product
- No social proof or trust signals
- Fails to differentiate from competitors

Competitor analysis shows modern DevTool landing pages (ngrok, Vercel, Railway, Dagger, Pulumi) feature:
- Interactive code/terminal previews
- Social proof badges (GitHub stars, user counts)
- Value proposition pills/badges
- Two-column layouts with visual balance
- Subtle background treatments for depth

### Goals

1. **Increase engagement** - Terminal preview demonstrates product value immediately
2. **Build trust** - GitHub stars badge provides social proof
3. **Communicate value** - Pills highlight core differentiators at a glance
4. **Modern aesthetics** - Two-column layout with background pattern matches industry standards

---

## 2. Requirements

### Must-Have Elements

#### 2.1 Terminal Preview

A mock terminal window showing a `wave run` command with realistic output:

```
$ wave run code-review --issue 42

[wave] Loading pipeline: code-review
[wave] Persona: reviewer (read-only, no network)
[wave] Step 1/3: analyze-diff .............. done
[wave] Step 2/3: check-contracts ........... done
[wave] Step 3/3: generate-report ........... done
[wave] Output: artifacts/review-report.md
[wave] Pipeline completed in 23.4s
```

**Requirements:**
- macOS-style window chrome (red/yellow/green dots)
- Syntax highlighting for prompt vs output
- Subtle typing animation on initial load (optional enhancement)
- Copy button in top-right corner

#### 2.2 Value Proposition Pills

Four pills highlighting core Wave features:

| Pill | Icon | Description |
|------|------|-------------|
| Declarative | `yaml` icon | YAML-based configuration |
| Contracts | `shield-check` icon | Output validation |
| Isolation | `lock` icon | Secure workspaces |
| Audit | `scroll` icon | Full traceability |

**Requirements:**
- Horizontal layout on desktop, 2x2 grid on mobile
- Subtle hover effect (scale + shadow)
- Each pill links to relevant docs section

#### 2.3 GitHub Stars Badge

Display current star count using shields.io:

```
https://img.shields.io/github/stars/re-cinq/wave?style=social
```

**Requirements:**
- Positioned near CTA buttons
- Links to GitHub repository
- Fallback to static badge if API fails

#### 2.4 Two-Column Layout

| Left Column (55%) | Right Column (45%) |
|-------------------|-------------------|
| Title | Terminal Preview |
| Tagline | |
| Value Pills | |
| CTA Buttons + GitHub Badge | |

**Requirements:**
- Left column is text-focused, right is visual
- Columns stack vertically on mobile (text first, terminal second)
- Maintain visual balance with proper spacing

#### 2.5 Background Pattern

Subtle grid/dot pattern for visual depth:

**Requirements:**
- Very low opacity (0.03-0.05) to not distract
- Dot grid pattern with ~40px spacing
- Respects dark/light mode color variables
- CSS-only implementation (no images)

---

## 3. Component Props Interface

```typescript
/**
 * Terminal line types for syntax highlighting
 */
export type TerminalLineType = 'command' | 'output' | 'success' | 'error' | 'info'

export interface TerminalLine {
  text: string
  type: TerminalLineType
  /** Optional delay before showing this line (ms) - for typing animation */
  delay?: number
}

/**
 * Value proposition pill configuration
 */
export interface ValuePill {
  /** Display label */
  label: string
  /** Icon name (from iconify or local icon set) */
  icon: string
  /** Link to relevant documentation */
  link: string
  /** Tooltip text on hover */
  tooltip?: string
}

/**
 * GitHub badge configuration
 */
export interface GitHubBadge {
  /** GitHub org/repo path */
  repo: string
  /** Badge style: 'social' | 'flat' | 'flat-square' */
  style?: 'social' | 'flat' | 'flat-square'
}

/**
 * Action button configuration
 */
export interface HeroAction {
  text: string
  link: string
  /** Button variant */
  variant?: 'primary' | 'secondary'
}

/**
 * Enhanced HeroSection props
 */
export interface HeroSectionProps {
  /** Main headline */
  title: string
  /** Supporting tagline */
  tagline: string
  /** Primary CTA button */
  primaryAction: HeroAction
  /** Secondary CTA button (optional) */
  secondaryAction?: HeroAction

  /** Terminal preview configuration */
  terminal?: {
    /** Lines to display in the terminal */
    lines: TerminalLine[]
    /** Terminal window title */
    title?: string
    /** Enable typing animation */
    animate?: boolean
  }

  /** Value proposition pills */
  valuePills?: ValuePill[]

  /** GitHub stars badge */
  github?: GitHubBadge

  /** Show background pattern */
  showBackground?: boolean
}
```

---

## 4. Visual Mockup

### Desktop Layout (>= 1024px)

```
+-----------------------------------------------------------------------------------+
|  . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . .   |
|  . . . . . . . . . . . . . (subtle dot pattern background) . . . . . . . . . .   |
|  . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . .   |
|  +----------------------------------+  +--------------------------------------+   |
|  |                                  |  |  +--------------------------------+  |   |
|  |  AI-as-Code for Multi-Agent     |  |  | $ wave run code-review --issue |  |   |
|  |  Pipelines                       |  |  |                                |  |   |
|  |                                  |  |  | [wave] Loading pipeline...     |  |   |
|  |  Orchestrate AI agents with      |  |  | [wave] Persona: reviewer       |  |   |
|  |  declarative YAML. Enforce       |  |  | [wave] Step 1/3: analyze-diff  |  |   |
|  |  contracts. Ship with confidence.|  |  | [wave] Step 2/3: check-contra  |  |   |
|  |                                  |  |  | [wave] Step 3/3: generate-rep  |  |   |
|  |  +----------+ +----------+       |  |  | [wave] Pipeline completed      |  |   |
|  |  |Declarativ| |Contracts |       |  |  +--------------------------------+  |   |
|  |  +----------+ +----------+       |  |                                      |   |
|  |  +----------+ +----------+       |  +--------------------------------------+   |
|  |  |Isolation | | Audit    |       |                                            |
|  |  +----------+ +----------+       |                                            |
|  |                                  |                                            |
|  |  [Get Started]  [View on GitHub] |                                            |
|  |                   * 1.2k         |                                            |
|  +----------------------------------+                                            |
|  . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . .   |
+-----------------------------------------------------------------------------------+
```

### Tablet Layout (768px - 1023px)

```
+-----------------------------------------------+
|  . . . . . . . . . . . . . . . . . . . . . .  |
|  +------------------------------------------+ |
|  |                                          | |
|  |     AI-as-Code for Multi-Agent           | |
|  |     Pipelines                            | |
|  |                                          | |
|  |     Orchestrate AI agents with           | |
|  |     declarative YAML...                  | |
|  |                                          | |
|  |  +-----------+ +-----------+             | |
|  |  |Declarative| | Contracts |             | |
|  |  +-----------+ +-----------+             | |
|  |  +-----------+ +-----------+             | |
|  |  | Isolation | |   Audit   |             | |
|  |  +-----------+ +-----------+             | |
|  |                                          | |
|  |     [Get Started] [GitHub] * 1.2k        | |
|  |                                          | |
|  +------------------------------------------+ |
|  +------------------------------------------+ |
|  | +--------------------------------------+ | |
|  | | $ wave run code-review --issue 42   | | |
|  | | [wave] Loading pipeline...          | | |
|  | | [wave] Step 1/3: analyze-diff       | | |
|  | +--------------------------------------+ | |
|  +------------------------------------------+ |
+-----------------------------------------------+
```

### Mobile Layout (< 768px)

```
+---------------------------+
|  . . . . . . . . . . . .  |
|  +-----------------------+ |
|  |                       | |
|  | AI-as-Code for        | |
|  | Multi-Agent Pipelines | |
|  |                       | |
|  | Orchestrate AI agents | |
|  | with declarative...   | |
|  |                       | |
|  | +--------+ +--------+ | |
|  | |Declara.| |Contract| | |
|  | +--------+ +--------+ | |
|  | +--------+ +--------+ | |
|  | |Isolat. | | Audit  | | |
|  | +--------+ +--------+ | |
|  |                       | |
|  |    [Get Started]      | |
|  |    [GitHub] * 1.2k    | |
|  +-----------------------+ |
|  +-----------------------+ |
|  | $ wave run...         | |
|  | [wave] Loading...     | |
|  +-----------------------+ |
+---------------------------+
```

---

## 5. Responsive Behavior

| Breakpoint | Layout | Changes |
|------------|--------|---------|
| >= 1024px | Two-column (55/45) | Full terminal preview, all pills visible |
| 768-1023px | Single-column stacked | Terminal below content, full width |
| < 768px | Single-column stacked | Smaller title (2.5rem), condensed pills (2x2), abbreviated terminal |

### Detailed Responsive Rules

#### Typography
- **Desktop:** Title 3.5rem, tagline 1.5rem
- **Tablet:** Title 3rem, tagline 1.35rem
- **Mobile:** Title 2.5rem, tagline 1.2rem

#### Value Pills
- **Desktop:** Single row, 4 pills
- **Tablet/Mobile:** 2x2 grid

#### Terminal Preview
- **Desktop:** Full height (~280px), 8+ lines visible
- **Tablet:** Medium height (~220px), 6 lines visible
- **Mobile:** Compact height (~160px), 4 lines visible with scroll

#### CTA Buttons
- **Desktop:** Inline with GitHub badge
- **Mobile:** Stacked vertically, full width

---

## 6. Acceptance Criteria

### Functional Requirements

- [ ] Two-column layout renders correctly on desktop (>= 1024px)
- [ ] Layout stacks to single column on tablet and mobile
- [ ] Terminal preview displays with macOS-style window chrome
- [ ] Terminal content syntax highlights commands vs output
- [ ] Copy button in terminal copies all commands
- [ ] Four value proposition pills render with icons
- [ ] Each pill links to corresponding documentation page
- [ ] GitHub stars badge loads from shields.io
- [ ] GitHub badge links to repository
- [ ] Primary CTA button navigates to getting started
- [ ] Secondary CTA button navigates to GitHub
- [ ] Dot pattern background is visible but subtle
- [ ] Background respects dark/light mode

### Visual/Design Requirements

- [ ] Title uses gradient text effect (existing Wave brand)
- [ ] Terminal has proper monospace font styling
- [ ] Pills have hover states (scale + shadow)
- [ ] Buttons maintain existing Wave styling
- [ ] Spacing is balanced and consistent
- [ ] No horizontal scroll at any breakpoint

### Performance Requirements

- [ ] No layout shift on initial load
- [ ] Terminal animation (if enabled) is smooth 60fps
- [ ] Total hero section bundle size < 10KB gzipped
- [ ] Background pattern is CSS-only (no image requests)

### Accessibility Requirements

- [ ] All interactive elements are keyboard accessible
- [ ] Terminal content is in an `aria-label` or accessible container
- [ ] Color contrast meets WCAG AA standards
- [ ] Reduced motion preference disables animations

### Browser Support

- [ ] Chrome/Edge (latest 2 versions)
- [ ] Firefox (latest 2 versions)
- [ ] Safari (latest 2 versions)
- [ ] Mobile Safari iOS 15+
- [ ] Chrome Android (latest)

---

## 7. Implementation Notes

### File Changes Required

1. **`docs/.vitepress/theme/types.ts`** - Add new interfaces
2. **`docs/.vitepress/theme/components/HeroSection.vue`** - Complete rewrite
3. **`docs/.vitepress/theme/styles/components.css`** - Add hero styles
4. **`docs/index.md`** - Update hero configuration

### Dependencies

- No new npm dependencies required
- Uses existing VitePress CSS variables
- Icons from existing icon system or inline SVGs

### CSS Variables to Use

```css
--wave-primary          /* Brand blue */
--wave-primary-dark     /* Hover state */
--wave-accent           /* Gradient end */
--vp-c-bg              /* Background */
--vp-c-bg-soft         /* Soft background */
--vp-c-text-1          /* Primary text */
--vp-c-text-2          /* Secondary text */
--vp-c-divider         /* Borders */
--wave-font-mono       /* Monospace font */
```

---

## 8. Out of Scope

The following are explicitly NOT part of this spec:

- Animated typing effect (nice-to-have, can be added later)
- Trust logos from companies (no social proof logos yet)
- Video demo (future enhancement)
- A/B testing infrastructure
- Analytics event tracking

---

## 9. References

- Current component: `/docs/.vitepress/theme/components/HeroSection.vue`
- Current styles: `/docs/.vitepress/theme/styles/components.css`
- Type definitions: `/docs/.vitepress/theme/types.ts`
- Competitor examples: ngrok.com, vercel.com, railway.app, dagger.io, pulumi.com
