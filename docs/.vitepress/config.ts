import { defineConfig } from 'vitepress'
import { withMermaid } from 'vitepress-plugin-mermaid'

export default withMermaid(
  defineConfig({
    title: 'Wave',
    description: 'Infrastructure as Code for AI - Define reproducible AI workflows with declarative configuration',
    themeConfig: {
      logo: '/logo.svg',
      nav: [
        { text: 'Workflows', link: '/workflows/creating-workflows' },
        {
          text: 'AI as Code',
          items: [
            { text: 'Core Paradigm', link: '/paradigm/ai-as-code' },
            { text: 'Infrastructure Parallels', link: '/paradigm/infrastructure-parallels' },
            { text: 'Deliverables + Contracts', link: '/paradigm/deliverables-contracts' }
          ]
        },
        {
          text: 'Concepts',
          items: [
            { text: 'Contracts', link: '/concepts/contracts' },
            { text: 'Pipeline Execution', link: '/concepts/pipeline-execution' },
            { text: 'Personas', link: '/concepts/personas' },
            { text: 'Workspaces', link: '/concepts/workspaces' }
          ]
        },
        {
          text: 'Reference',
          items: [
            { text: 'YAML Schema', link: '/reference/yaml-schema' },
            { text: 'CLI Commands', link: '/reference/cli-commands' },
            { text: 'Troubleshooting', link: '/reference/troubleshooting' }
          ]
        }
      ],
      sidebar: {
        '/workflows/': [
          {
            text: 'Workflows',
            items: [
              { text: 'Creating Workflows', link: '/workflows/creating-workflows' },
              { text: 'Sharing Workflows', link: '/workflows/sharing-workflows' },
              { text: 'Community Library', link: '/workflows/community-library' },
              { text: 'Examples', link: '/workflows/examples/' }
            ]
          }
        ],
        '/paradigm/': [
          {
            text: 'AI as Code',
            items: [
              { text: 'Core Paradigm', link: '/paradigm/ai-as-code' },
              { text: 'Infrastructure Parallels', link: '/paradigm/infrastructure-parallels' },
              { text: 'Deliverables + Contracts', link: '/paradigm/deliverables-contracts' }
            ]
          }
        ],
        '/concepts/': [
          {
            text: 'Concepts',
            items: [
              { text: 'Contracts', link: '/concepts/contracts' },
              { text: 'Pipeline Execution', link: '/concepts/pipeline-execution' },
              { text: 'Personas', link: '/concepts/personas' },
              { text: 'Workspaces', link: '/concepts/workspaces' }
            ]
          }
        ],
        '/reference/': [
          {
            text: 'Reference',
            items: [
              { text: 'YAML Schema', link: '/reference/yaml-schema' },
              { text: 'CLI Commands', link: '/reference/cli-commands' },
              { text: 'Troubleshooting', link: '/reference/troubleshooting' }
            ]
          }
        ],
        '/migration/': [
          {
            text: 'Adoption',
            items: [
              { text: 'From Personas to Workflows', link: '/migration/from-personas-to-workflows' },
              { text: 'Team Adoption', link: '/migration/team-adoption' },
              { text: 'Enterprise Patterns', link: '/migration/enterprise-patterns' }
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
