import { defineConfig } from 'vitepress'
import { withMermaid } from 'vitepress-plugin-mermaid'

export default withMermaid(
  defineConfig({
    title: 'Wave',
    titleTemplate: ':title · AI-as-Code for multi-agent pipelines',
    description: 'Define, version, and run AI workflows like you manage infrastructure.',
    base: '/wave/',

    head: [
      ['meta', { name: 'keywords', content: 'AI, pipelines, orchestration, LLM, Claude, automation, YAML, DevOps' }],
      ['meta', { name: 'author', content: 'Michael W. Czechowski' }],
      ['meta', { property: 'og:title', content: 'Wave · AI-as-Code for multi-agent pipelines' }],
      ['meta', { property: 'og:description', content: 'Define, version, and run AI workflows like you manage infrastructure.' }],
      ['meta', { property: 'og:type', content: 'website' }],
      ['meta', { property: 'og:image', content: '/wave/og-image.png' }],
      ['meta', { name: 'twitter:card', content: 'summary_large_image' }],
      ['meta', { name: 'twitter:title', content: 'Wave · AI-as-Code for multi-agent pipelines' }],
      ['meta', { name: 'twitter:description', content: 'Define, version, and run AI workflows like you manage infrastructure.' }],
      ['link', { rel: 'icon', type: 'image/svg+xml', href: '/wave/favicon.svg' }],
      ['link', { rel: 'preconnect', href: 'https://fonts.googleapis.com' }],
      ['link', { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' }],
      ['link', { href: 'https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;600&display=swap', rel: 'stylesheet' }]
    ],

    srcExclude: ['future/**', '.archive/**'],

    themeConfig: {
      logo: '/logo.svg',
      nav: [
        { text: 'Quickstart', link: '/quickstart' },
        { text: 'Use Cases', link: '/use-cases/' },
        {
          text: 'Concepts',
          items: [
            { text: 'Overview', link: '/concepts/' },
            { text: 'AI-as-Code', link: '/concepts/ai-as-code' },
            { text: 'Pipelines', link: '/concepts/pipelines' },
            { text: 'Personas', link: '/concepts/personas' },
            { text: 'Contracts', link: '/concepts/contracts' },
            { text: 'Artifacts', link: '/concepts/artifacts' },
            { text: 'Workspaces', link: '/concepts/workspaces' }
          ]
        },
        {
          text: 'Guides',
          items: [
            { text: 'Pipeline Configuration', link: '/guides/pipeline-configuration' },
            { text: 'Sandbox Setup', link: '/guides/sandbox-setup' },
            { text: 'CI/CD Integration', link: '/guides/ci-cd' },
            { text: 'GitHub Integration', link: '/guides/github-integration' },
            { text: 'Enterprise Patterns', link: '/guides/enterprise' },
            { text: 'Audit Logging', link: '/guides/audit-logging' },
            { text: 'State & Resumption', link: '/guides/state-resumption' },
            { text: 'Context Relay', link: '/guides/relay-compaction' },
            { text: 'Meta-Pipelines', link: '/guides/meta-pipelines' }
          ]
        },
        {
          text: 'Reference',
          items: [
            { text: 'CLI Commands', link: '/reference/cli' },
            { text: 'Manifest Schema', link: '/reference/manifest-schema' },
            { text: 'Pipeline Schema', link: '/reference/pipeline-schema' },
            { text: 'Adapters', link: '/reference/adapters' },
            { text: 'Events', link: '/reference/events' },
            { text: 'Environment', link: '/reference/environment' },
            { text: 'Error Codes', link: '/reference/error-codes' },
            { text: 'Troubleshooting', link: '/reference/troubleshooting' }
          ]
        },
        { text: 'Trust Center', link: '/trust-center/' },
        { text: 'Changelog', link: '/changelog' }
      ],
      sidebar: {
        '/use-cases/': [
          {
            text: 'Use Cases',
            items: [
              { text: 'Overview', link: '/use-cases/' },
              { text: 'Code Review', link: '/use-cases/ops-pr-review' },
              { text: 'Doc Consistency', link: '/use-cases/audit-doc' },
              { text: 'Issue Enhancement', link: '/use-cases/github-issue-enhancer' },
              { text: 'Issue Research', link: '/use-cases/plan-research' },
              { text: 'Test Generation', link: '/use-cases/test-generation' },
              { text: 'Refactoring', link: '/use-cases/refactoring' },
              { text: 'Work Supervision', link: '/use-cases/ops-supervise' },
              { text: 'Recinq', link: '/use-cases/recinq' }
            ]
          }
        ],
        '/concepts/': [
          {
            text: 'Concepts',
            items: [
              { text: 'Overview', link: '/concepts/' },
              { text: 'AI-as-Code', link: '/concepts/ai-as-code' },
              { text: 'Pipelines', link: '/concepts/pipelines' },
              { text: 'Personas', link: '/concepts/personas' },
              { text: 'Contracts', link: '/concepts/contracts' },
              { text: 'Artifacts', link: '/concepts/artifacts' },

              { text: 'Workspaces', link: '/concepts/workspaces' },
              { text: 'Adapters', link: '/concepts/adapters' },
              { text: 'Manifests', link: '/concepts/manifests' },
              { text: 'Architecture', link: '/concepts/architecture' }
            ]
          }
        ],
        '/guides/': [
          {
            text: 'Getting Started',
            items: [
              { text: 'Pipeline Configuration', link: '/guides/pipeline-configuration' },
              { text: 'Sandbox Setup', link: '/guides/sandbox-setup' },
              { text: 'Custom Personas', link: '/guides/custom-personas' }
            ]
          },
          {
            text: 'Adoption',
            items: [
              { text: 'CI/CD Integration', link: '/guides/ci-cd' },
              { text: 'GitHub Integration', link: '/guides/github-integration' },
              { text: 'Enterprise Patterns', link: '/guides/enterprise' },
              { text: 'Web Dashboard', link: '/guides/web-dashboard' },
              { text: 'Secure CLI Patterns', link: '/guides/secure-cli-patterns' }
            ]
          },
          {
            text: 'Advanced',
            items: [
              { text: 'Audit Logging', link: '/guides/audit-logging' },
              { text: 'State & Resumption', link: '/guides/state-resumption' },
              { text: 'Context Relay', link: '/guides/relay-compaction' },
              { text: 'Meta-Pipelines', link: '/guides/meta-pipelines' },
              { text: 'Contract Chaining', link: '/guides/contract-chaining' },
              { text: 'V&V Patterns', link: '/guides/vv-patterns' },
              { text: 'Adapter Development', link: '/guides/adapter-development' }
            ]
          },
          {
            text: 'Platform Setup',
            items: [
              { text: 'Forge Setup', link: '/guides/forge-setup' }
            ]
          },
          {
            text: 'Maintenance',
            items: [
              { text: 'Upgrade Guide', link: '/guides/upgrade-guide' }
            ]
          }
        ],
        '/guide/': [
          {
            text: 'Core Concepts',
            items: [
              { text: 'Pipelines', link: '/guide/pipelines' },
              { text: 'Contracts', link: '/guide/contracts' },
              { text: 'Personas', link: '/guide/personas' },
              { text: 'Outcomes', link: '/guide/outcomes' },
              { text: 'Validation', link: '/guide/validation' }
            ]
          },
          {
            text: 'Configuration',
            items: [
              { text: 'Configuration', link: '/guide/configuration' },
              { text: 'Installation', link: '/guide/installation' },
              { text: 'Quick Start', link: '/guide/quick-start' },
              { text: 'Relay', link: '/guide/relay' }
            ]
          },
          {
            text: 'Advanced Patterns',
            items: [
              { text: 'Composition', link: '/guide/composition' },
              { text: 'Graph Loops', link: '/guide/graph-loops' },
              { text: 'Human Gates', link: '/guide/human-gates' },
              { text: 'Threads', link: '/guide/threads' },
              { text: 'Retry Policies', link: '/guide/retry-policies' },
              { text: 'Model Routing', link: '/guide/model-routing' }
            ]
          },
          {
            text: 'Skills',
            items: [
              { text: 'Skills', link: '/guide/skills' },
              { text: 'TUI', link: '/guide/tui' },
              { text: 'Chat Context', link: '/guide/chat-context' },
              { text: 'Pipeline Outputs', link: '/guide/pipeline-outputs' }
            ]
          }
        ],
        '/reference/': [
          {
            text: 'Reference',
            items: [
              { text: 'CLI Commands', link: '/reference/cli' },
              { text: 'Manifest Schema', link: '/reference/manifest-schema' },
              { text: 'Pipeline Schema', link: '/reference/pipeline-schema' },
              { text: 'Adapters', link: '/reference/adapters' },
              { text: 'Events', link: '/reference/events' },
              { text: 'Environment', link: '/reference/environment' },
              { text: 'Contract Types', link: '/reference/contract-types' },
              { text: 'Error Codes', link: '/reference/error-codes' },
              { text: 'Troubleshooting', link: '/reference/troubleshooting' }
            ]
          }
        ],
      },
      socialLinks: [
        { icon: 'github', link: 'https://github.com/re-cinq/wave' }
      ],
      search: {
        provider: 'local'
      },
      editLink: {
        pattern: 'https://github.com/re-cinq/wave/edit/main/docs/:path',
        text: 'Edit this page on GitHub'
      },
      footer: {
        message: 'Released under the MIT License.',
        copyright: 'Copyright 2026 Michael W. Czechowski at re:cinq ApS'
      }
    }
  })
)
