/**
 * Wave Documentation - Enhanced HeroSection Types
 * TypeScript interfaces for the redesigned HeroSection component
 *
 * Design Goals:
 * - Backwards compatible with existing HeroSectionProps
 * - Support for terminal preview with typing animation
 * - Value proposition pills for quick feature highlights
 * - Social proof integration (GitHub stars, badges)
 * - Flexible background and layout variants
 */

// =============================================================================
// Action Types (unchanged for compatibility)
// =============================================================================

export interface HeroAction {
  text: string
  link: string
}

// =============================================================================
// Terminal Preview Types
// =============================================================================

/**
 * Configuration for animated terminal preview
 * Shows a simulated CLI interaction to demonstrate Wave usage
 */
export interface TerminalPreviewConfig {
  /** The command being typed/executed (e.g., "wave run pipeline.yaml") */
  command: string

  /** Lines of output displayed after command execution */
  outputLines: TerminalOutputLine[]

  /** Enable typing animation for the command (default: true) */
  typingAnimation?: boolean

  /** Typing speed in milliseconds per character (default: 50) */
  typingSpeed?: number

  /** Delay before showing output after command is typed (default: 500) */
  outputDelay?: number

  /** Terminal prompt prefix (default: "$") */
  prompt?: string

  /** Optional title for the terminal window */
  title?: string
}

/**
 * A single line of terminal output with optional styling
 */
export interface TerminalOutputLine {
  /** The text content of the line */
  text: string

  /** Optional color variant for the line */
  variant?: TerminalLineVariant

  /** Delay before showing this line (for staggered animation) */
  delay?: number

  /** Optional icon prefix (e.g., checkmark, spinner) */
  icon?: TerminalIcon
}

export type TerminalLineVariant =
  | 'default'   // Standard output
  | 'success'   // Green text (e.g., "Done!")
  | 'error'     // Red text
  | 'warning'   // Yellow text
  | 'info'      // Blue text
  | 'muted'     // Dimmed/gray text
  | 'highlight' // Emphasized text

export type TerminalIcon =
  | 'check'     // Checkmark for success
  | 'cross'     // X for failure
  | 'spinner'   // Loading spinner
  | 'arrow'     // Arrow indicator
  | 'dot'       // Bullet point

// =============================================================================
// Value Proposition Types
// =============================================================================

/**
 * A small pill/badge highlighting a key feature or value prop
 * Displayed as a horizontal row of compact badges
 */
export interface ValuePropositionPill {
  /** Short label text (e.g., "Zero Config", "Type Safe") */
  label: string

  /** Optional icon name or emoji */
  icon?: string

  /** Optional tooltip for more detail */
  tooltip?: string

  /** Optional link to learn more */
  link?: string
}

// =============================================================================
// Social Proof Types
// =============================================================================

/**
 * Social proof configuration for the hero section
 * Can show GitHub stars, download counts, or custom badges
 */
export interface SocialProofConfig {
  /** Type of social proof to display */
  type: SocialProofType

  /** Configuration specific to the type */
  config: GitHubStarsConfig | CustomBadgeConfig
}

export type SocialProofType = 'github-stars' | 'custom-badge'

/**
 * GitHub stars badge configuration
 * Fetches and displays live star count from GitHub API
 */
export interface GitHubStarsConfig {
  type: 'github-stars'

  /** GitHub repository URL (e.g., "https://github.com/re-cinq/wave") */
  repoUrl: string

  /** Optional label override (default: "GitHub Stars") */
  label?: string

  /** Cache duration in seconds (default: 3600) */
  cacheDuration?: number
}

/**
 * Custom badge configuration for arbitrary social proof
 */
export interface CustomBadgeConfig {
  type: 'custom-badge'

  /** Badge label text */
  label: string

  /** Badge value/count */
  value: string | number

  /** Optional icon */
  icon?: string

  /** Optional link */
  link?: string
}

// =============================================================================
// Visual Variant Types
// =============================================================================

/**
 * Background style variant for the hero section
 */
export type HeroBackgroundVariant =
  | 'grid'     // Subtle grid pattern
  | 'dots'     // Dot matrix pattern
  | 'gradient' // Gradient background
  | 'none'     // No background decoration

/**
 * Layout variant for the hero section
 */
export type HeroLayoutVariant =
  | 'centered'   // Content centered, stacked vertically
  | 'two-column' // Text left, terminal preview right

// =============================================================================
// Background Configuration (advanced)
// =============================================================================

/**
 * Extended background configuration for custom styling
 */
export interface HeroBackgroundConfig {
  /** Base variant */
  variant: HeroBackgroundVariant

  /** Custom gradient colors (for 'gradient' variant) */
  gradientColors?: {
    from: string
    to: string
    direction?: 'to-right' | 'to-bottom' | 'to-bottom-right'
  }

  /** Grid/dots opacity (0-1, default: 0.1) */
  patternOpacity?: number

  /** Enable animated gradient shift */
  animated?: boolean
}

// =============================================================================
// Enhanced HeroSection Props
// =============================================================================

/**
 * Enhanced HeroSection component props
 *
 * Backwards compatible with existing HeroSectionProps:
 * - title, tagline, primaryAction, secondaryAction remain unchanged
 *
 * New capabilities:
 * - Terminal preview with typing animation
 * - Value proposition pills
 * - Social proof integration
 * - Background and layout variants
 */
export interface EnhancedHeroSectionProps {
  // -------------------------------------------------------------------------
  // Core Content (existing, unchanged for compatibility)
  // -------------------------------------------------------------------------

  /** Main headline text */
  title: string

  /** Supporting tagline/description */
  tagline: string

  /** Primary CTA button */
  primaryAction: HeroAction

  /** Optional secondary CTA button */
  secondaryAction?: HeroAction

  // -------------------------------------------------------------------------
  // Terminal Preview (new)
  // -------------------------------------------------------------------------

  /** Terminal preview configuration */
  terminal?: TerminalPreviewConfig

  // -------------------------------------------------------------------------
  // Value Propositions (new)
  // -------------------------------------------------------------------------

  /** Array of value proposition pills to display */
  valuePills?: ValuePropositionPill[]

  // -------------------------------------------------------------------------
  // Social Proof (new)
  // -------------------------------------------------------------------------

  /** Social proof badge configuration */
  socialProof?: SocialProofConfig

  // -------------------------------------------------------------------------
  // Visual Configuration (new)
  // -------------------------------------------------------------------------

  /** Background variant (default: 'grid') */
  background?: HeroBackgroundVariant | HeroBackgroundConfig

  /** Layout variant (default: 'centered', 'two-column' when terminal provided) */
  layout?: HeroLayoutVariant
}

// =============================================================================
// Backwards Compatibility Type
// =============================================================================

/**
 * Original HeroSectionProps type for reference
 * EnhancedHeroSectionProps extends this with additional optional properties
 */
export interface HeroSectionProps {
  title: string
  tagline: string
  primaryAction: {
    text: string
    link: string
  }
  secondaryAction?: {
    text: string
    link: string
  }
}

/**
 * Type guard to check if props are enhanced
 */
export function isEnhancedHeroProps(
  props: HeroSectionProps | EnhancedHeroSectionProps
): props is EnhancedHeroSectionProps {
  return (
    'terminal' in props ||
    'valuePills' in props ||
    'socialProof' in props ||
    'background' in props ||
    'layout' in props
  )
}

// =============================================================================
// Component State Types (for internal use)
// =============================================================================

/**
 * Internal state for terminal animation
 */
export interface TerminalAnimationState {
  /** Current phase of animation */
  phase: 'idle' | 'typing' | 'executing' | 'complete'

  /** Characters typed so far */
  typedChars: number

  /** Output lines revealed so far */
  revealedLines: number

  /** Whether animation is paused */
  paused: boolean
}

// =============================================================================
// Default Values
// =============================================================================

export const HERO_DEFAULTS = {
  background: 'grid' as HeroBackgroundVariant,
  layout: 'centered' as HeroLayoutVariant,
  terminal: {
    typingAnimation: true,
    typingSpeed: 50,
    outputDelay: 500,
    prompt: '$',
  },
} as const
