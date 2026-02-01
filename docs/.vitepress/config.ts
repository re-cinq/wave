import { defineConfig } from 'vitepress'
import { withMermaid } from 'vitepress-plugin-mermaid'

export default withMermaid(
  defineConfig({
    title: 'Wave',
    description: 'Multi-Agent Orchestrator for Claude Code',
    themeConfig: {
      logo: '/logo.svg',
      nav: [
        { text: 'Guide', link: '/guide/installation' },
        {
          text: 'Concepts',
          items: [
            { text: 'Architecture', link: '/concepts/architecture' },
            { text: 'Manifests', link: '/concepts/manifests' },
            { text: 'Pipelines', link: '/concepts/pipelines' },
            { text: 'Personas', link: '/concepts/personas' },
            { text: 'Adapters', link: '/concepts/adapters' },
            { text: 'Workspaces', link: '/concepts/workspaces' },
            { text: 'Contracts', link: '/concepts/contracts' }
          ]
        },
        {
          text: 'Reference',
          items: [
            { text: 'CLI Commands', link: '/reference/cli' },
            { text: 'Manifest Schema', link: '/reference/manifest-schema' },
            { text: 'Pipeline Schema', link: '/reference/pipeline-schema' },
            { text: 'Event Format', link: '/reference/events' },
            { text: 'Environment', link: '/reference/environment' }
          ]
        },
        { text: 'Examples', link: '/examples/' }
      ],
      sidebar: {
        '/guide/': [
          {
            text: 'Getting Started',
            items: [
              { text: 'Installation', link: '/guide/installation' },
              { text: 'Quick Start', link: '/guide/quick-start' },
              { text: 'Configuration', link: '/guide/configuration' }
            ]
          }
        ],
        '/concepts/': [
          {
            text: 'Concepts',
            items: [
              { text: 'Architecture', link: '/concepts/architecture' },
              { text: 'Manifests', link: '/concepts/manifests' },
              { text: 'Pipelines', link: '/concepts/pipelines' },
              { text: 'Personas', link: '/concepts/personas' },
              { text: 'Adapters', link: '/concepts/adapters' },
              { text: 'Workspaces', link: '/concepts/workspaces' },
              { text: 'Contracts', link: '/concepts/contracts' }
            ]
          }
        ],
        '/guides/': [
          {
            text: 'Guides',
            items: [
              { text: 'Context Relay', link: '/guides/relay-compaction' },
              { text: 'Matrix Strategies', link: '/guides/matrix-strategies' },
              { text: 'Meta-Pipelines', link: '/guides/meta-pipelines' },
              { text: 'State & Resumption', link: '/guides/state-resumption' },
              { text: 'Audit Logging', link: '/guides/audit-logging' }
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
              { text: 'Event Format', link: '/reference/events' },
              { text: 'Environment', link: '/reference/environment' }
            ]
          }
        ],
        '/examples/': [
          {
            text: 'Examples',
            items: [
              { text: 'Overview', link: '/examples/' },
              { text: 'Speckit Flow', link: '/examples/speckit-flow' },
              { text: 'Hotfix Pipeline', link: '/examples/hotfix-pipeline' },
              { text: 'Custom Adapter', link: '/examples/custom-adapter' }
            ]
          }
        ]
      },
        socialLinks: [
        { icon: 'github', link: 'https://github.com/recinq/wave' }
      ],
      search: {
        provider: 'local'
      },
      editLink: {
        pattern: 'https://github.com/recinq/wave/edit/main/docs/:path',
        text: 'Edit this page on GitHub'
      },
      footer: {
        message: 'Released under the MIT License.',
        copyright: 'Copyright 2026 Wave Contributors'
      }
    }
  })
)
