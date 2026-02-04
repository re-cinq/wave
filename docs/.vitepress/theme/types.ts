/**
 * Wave Documentation - Shared Type Definitions
 * TypeScript types for Vue components
 */

// Feature Card Types
export interface FeatureCard {
  icon: string
  title: string
  description: string
  link?: string
}

// Trust Signal Types
export type ComplianceStatus = 'certified' | 'in-progress' | 'planned'

export interface TrustBadge {
  name: string
  status: ComplianceStatus
  description?: string
  link?: string
}

// Platform Tab Types
export type Platform = 'macos' | 'linux' | 'windows'

export interface PlatformContent {
  platform: Platform
  label: string
  content: string
}

// Use Case Types
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

// Permission Matrix Types
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

// Pipeline Visualizer Types
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

// YAML Playground Types
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

// Navigation Types
export interface BreadcrumbItem {
  text: string
  link?: string
}

// Component Props Types
export interface CopyButtonProps {
  code: string
  lang?: string
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
