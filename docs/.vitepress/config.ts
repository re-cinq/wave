import { defineConfig } from 'vitepress'
import { withMermaid } from 'vitepress-plugin-mermaid'

export default withMermaid(
  defineConfig({
    title: 'Wave',
    description: 'Infrastructure as Code for AI - Define reproducible AI workflows with declarative configuration',
    themeConfig: {
      logo: '/logo.svg',
      nav: [
        { text: 'Quickstart', link: '/quickstart' },
        { text: 'Use Cases', link: '/use-cases/' },
        {
          text: 'Concepts',
          items: [
            { text: 'Overview', link: '/concepts/' },
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
            { text: 'Team Adoption', link: '/guides/team-adoption' },
            { text: 'CI/CD Integration', link: '/guides/ci-cd' },
            { text: 'GitHub Integration', link: '/github-integration' },
            { text: 'Enterprise Patterns', link: '/guides/enterprise' },
            { text: 'Audit Logging', link: '/guides/audit-logging' },
            { text: 'Matrix Strategies', link: '/guides/matrix-strategies' },
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
            { text: 'Troubleshooting', link: '/reference/troubleshooting' }
          ]
        }
      ],
      sidebar: {
        '/use-cases/': [
          {
            text: 'Use Cases',
            items: [
              { text: 'Overview', link: '/use-cases/' }
            ]
          }
        ],
        '/concepts/': [
          {
            text: 'Concepts',
            items: [
              { text: 'Overview', link: '/concepts/' },
              { text: 'Pipelines', link: '/concepts/pipelines' },
              { text: 'Personas', link: '/concepts/personas' },
              { text: 'Contracts', link: '/concepts/contracts' },
              { text: 'Artifacts', link: '/concepts/artifacts' },
              { text: 'Execution', link: '/concepts/execution' },
              { text: 'Pipeline Execution', link: '/concepts/pipeline-execution' },
              { text: 'Workspaces', link: '/concepts/workspaces' },
              { text: 'Adapters', link: '/concepts/adapters' },
              { text: 'Manifests', link: '/concepts/manifests' },
              { text: 'Architecture', link: '/concepts/architecture' }
            ]
          }
        ],
        '/guides/': [
          {
            text: 'Adoption',
            items: [
              { text: 'Team Adoption', link: '/guides/team-adoption' },
              { text: 'CI/CD Integration', link: '/guides/ci-cd' },
              { text: 'GitHub Integration', link: '/github-integration' },
              { text: 'Enterprise Patterns', link: '/guides/enterprise' }
            ]
          },
          {
            text: 'Advanced',
            items: [
              { text: 'Audit Logging', link: '/guides/audit-logging' },
              { text: 'Matrix Strategies', link: '/guides/matrix-strategies' },
              { text: 'State & Resumption', link: '/guides/state-resumption' },
              { text: 'Context Relay', link: '/guides/relay-compaction' },
              { text: 'Meta-Pipelines', link: '/guides/meta-pipelines' }
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
              { text: 'Troubleshooting', link: '/reference/troubleshooting' }
            ]
          }
        ]
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
