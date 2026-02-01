import { defineConfig } from 'vitepress'
import { withMermaid } from 'vitepress-plugin-mermaid'

export default withMermaid(
  defineConfig({
    title: 'Muzzle',
    description: 'Multi-Agent Orchestrator for Claude Code',
    themeConfig: {
      nav: [
        { text: 'Guide', link: '/guide/installation' },
        { text: 'Concepts', link: '/concepts/architecture' },
        { text: 'Reference', link: '/reference/cli' },
        { text: 'Examples', link: '/examples/' }
      ],
      sidebar: [
        {
          text: 'Guide',
          items: [
            { text: 'Installation', link: '/guide/installation' },
            { text: 'Quick Start', link: '/guide/quick-start' },
            { text: 'Configuration', link: '/guide/configuration' }
          ]
        },
        {
          text: 'Concepts',
          items: [
            { text: 'Architecture', link: '/concepts/architecture' }
          ]
        },
        {
          text: 'Reference',
          items: [
            { text: 'CLI Commands', link: '/reference/cli' }
          ]
        },
        {
          text: 'Examples',
          items: [
            { text: 'Overview', link: '/examples/' }
          ]
        }
      ],
      socialLinks: [
        { icon: 'github', link: 'https://github.com/recinq/muzzle' }
      ]
    }
  })
)
