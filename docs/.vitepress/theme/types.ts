/**
 * Wave Documentation - Shared Type Definitions
 * TypeScript types for Vue components
 */

// =============================================================================
// Feature Card Types
// =============================================================================
export interface FeatureCard {
  icon: string
  title: string
  description: string
  link?: string
}

// =============================================================================
// Trust Signal Types
// =============================================================================

export type ComplianceStatus = 'certified' | 'in-progress' | 'planned'

export interface TrustBadge {
  name: string
  status: ComplianceStatus
  description?: string
  link?: string
}

// =============================================================================
// Platform Tab Types
// =============================================================================

export type Platform = 'macos' | 'linux' | 'windows'

export interface PlatformContent {
  platform: Platform
  label: string
  content: string
}

// =============================================================================
// Use Case Types
// =============================================================================

export type UseCaseCategory =
  | 'code-quality'
  | 'security'
  | 'documentation'
  | 'testing'
  | 'devops'
  | 'onboarding'

export type ComplexityLevel = 'beginner' | 'intermediate' | 'advanced'

export interface UseCase {
  id: string
  title: string
  description: string
  category: UseCaseCategory
  complexity: ComplexityLevel
  personas: string[]
  tags: string[]
  link: string
}

// =============================================================================
// Permission Matrix Types
// =============================================================================

export type PermissionLevel = 'allow' | 'deny' | 'conditional'

export interface PersonaPermission {
  persona: string
  description: string
  permissions: {
    read: PermissionLevel
    write: PermissionLevel
    execute: PermissionLevel
    network: PermissionLevel
  }
}

// =============================================================================
// Pipeline Visualizer Types
// =============================================================================

export interface PipelineStep {
  id: string
  name: string
  persona: string
  dependencies: string[]
  artifacts?: string[]
}

export interface Pipeline {
  name: string
  description: string
  steps: PipelineStep[]
}

// =============================================================================
// YAML Playground Types
// =============================================================================

export interface ValidationResult {
  valid: boolean
  errors: ValidationError[]
}

export interface ValidationError {
  line: number
  column: number
  message: string
  severity: 'error' | 'warning'
}

// =============================================================================
// Navigation Types
// =============================================================================

export interface BreadcrumbItem {
  text: string
  link?: string
}

// =============================================================================
// Component Props Types
// =============================================================================

export interface CopyButtonProps {
  code: string
  lang?: string
}

// =============================================================================
// Terminal Preview Types (for HeroSection)
// =============================================================================

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

// =============================================================================
// Value Proposition Types (for HeroSection)
// =============================================================================

/**
 * A small pill/badge highlighting a key feature or value prop
 */
export interface ValuePropositionPill {
  /** Short label text */
  label: string
  /** Optional icon name or emoji */
  icon?: string
  /** Optional tooltip for more detail */
  tooltip?: string
  /** Optional link to learn more */
  link?: string
}

// =============================================================================
// Social Proof Types (for HeroSection)
// =============================================================================

export type SocialProofType = 'github-stars' | 'custom-badge'

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
// Visual Variant Types (Enhanced HeroSection)
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
// HeroSection Props
// =============================================================================

/**
 * Simple GitHub badge configuration for HeroSection
 * Uses shields.io to display star count
 */
export interface HeroGitHubBadge {
  /** GitHub org/repo path (e.g., "re-cinq/wave") */
  repo: string
  /** Badge style (default: 'social') */
  style?: 'social' | 'flat' | 'flat-square'
}

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
  /** Terminal preview configuration (enables two-column layout) */
  terminal?: TerminalPreviewConfig
  /** Array of value proposition pills to display */
  valuePills?: ValuePropositionPill[]
  /** GitHub stars badge configuration */
  github?: HeroGitHubBadge
  /** Show background pattern (default: true when terminal is provided) */
  showBackground?: boolean
}

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
export interface EnhancedHeroSectionProps extends HeroSectionProps {
  /** Social proof badge configuration */
  socialProof?: SocialProofConfig
  /** Background variant (default: 'grid') */
  background?: HeroBackgroundVariant | HeroBackgroundConfig
  /** Layout variant (default: 'centered', 'two-column' when terminal provided) */
  layout?: HeroLayoutVariant
}

// =============================================================================
// Terminal Animation State (Internal)
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
// Hero Section Defaults
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

// =============================================================================
// Type Guards (Enhanced HeroSection)
// =============================================================================

/**
 * Type guard to check if props are enhanced
 */
export function isEnhancedHeroProps(
  props: HeroSectionProps | EnhancedHeroSectionProps
): props is EnhancedHeroSectionProps {
  return (
    'socialProof' in props ||
    'background' in props ||
    'layout' in props
  )
}

export interface FeatureCardsProps {
  features: FeatureCard[]
}

export interface TrustSignalsProps {
  badges: TrustBadge[]
}

export interface PlatformTabsProps {
  tabs: PlatformContent[]
  defaultPlatform?: Platform
}

export interface UseCaseGalleryProps {
  useCases: UseCase[]
  showFilters?: boolean
}

export interface PermissionMatrixProps {
  personas: PersonaPermission[]
  showLegend?: boolean
}

export interface PipelineVisualizerProps {
  pipeline: Pipeline
  interactive?: boolean
}

export interface YamlPlaygroundProps {
  initialValue?: string
  schema?: object
}
