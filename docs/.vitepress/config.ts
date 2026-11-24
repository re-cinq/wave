import { defineConfig } from 'vitepress'
import { withMermaid } from 'vitepress-plugin-mermaid'

export default withMermaid(
  defineConfig({
    title: 'Wave',
    titleTemplate: ':title - declarative, contract-checked AI agent pipelines',
    description: 'Declare AI coding agents in YAML, validate every handoff with typed contracts.',
    base: '/wave/',

    head: [
      ['meta', { name: 'keywords', content: 'AI, pipelines, orchestration, LLM, Claude, automation, YAML, DevOps' }],
      ['meta', { name: 'author', content: 'Michael W. Czechowski' }],
      ['meta', { property: 'og:title', content: 'Wave - declarative, contract-checked AI agent pipelines' }],
      ['meta', { property: 'og:description', content: 'Declare AI coding agents in YAML, validate every handoff with typed contracts.' }],
      ['meta', { property: 'og:type', content: 'website' }],
      ['meta', { property: 'og:image', content: '/wave/og-image.png' }],
      ['meta', { name: 'twitter:card', content: 'summary_large_image' }],
      ['meta', { name: 'twitter:title', content: 'Wave - declarative, contract-checked AI agent pipelines' }],
      ['meta', { name: 'twitter:description', content: 'Declare AI coding agents in YAML, validate every handoff with typed contracts.' }],
      ['link', { rel: 'icon', type: 'image/svg+xml', href: '/wave/favicon.svg' }],
      ['link', { rel: 'preconnect', href: 'https://fonts.googleapis.com' }],
      ['link', { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' }],
      ['link', { href: 'https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;600&display=swap', rel: 'stylesheet' }]
    ],

    srcExclude: ['adr/**'],

    // Wave docs reference Wave template syntax (e.g. `{{ step.out.<name> }}`)
    // throughout. VitePress runs the rendered markdown HTML through Vue's
    // SFC compiler, which treats `{{ }}` as expression interpolation. Escape
    // every mustache in markdown content to HTML entities so Vue never sees
    // them as expressions; theme `.vue` files are unaffected.
    markdown: {
      config: (md) => {
        const escape = (s: string) =>
          s.replace(/\{\{/g, '&#123;&#123;').replace(/\}\}/g, '&#125;&#125;')
        const wrap = (orig: any) =>
          function (this: any, tokens: any[], idx: number, options: any, env: any, self: any) {
            const out = orig.call(this, tokens, idx, options, env, self)
            return escape(out)
          }
        const rules: any = md.renderer.rules
        ;['text', 'code_inline', 'fence', 'code_block', 'html_block', 'html_inline'].forEach((rule) => {
          const prev = rules[rule] || md.renderer.renderToken.bind(md.renderer)
          rules[rule] = wrap(prev)
        })
      }
    },

    themeConfig: {
      logo: '/logo.svg',
      nav: [
        { text: 'Quickstart', link: '/quickstart' },
        { text: 'Concepts', items: [
            { text: 'Pipelines', link: '/concepts/pipelines' },
            { text: 'Personas', link: '/concepts/personas' },
            { text: 'Contracts', link: '/concepts/contracts' },
            { text: 'Artifacts', link: '/concepts/artifacts' },
            { text: 'Workspaces', link: '/concepts/workspaces' },
            { text: 'Adapters', link: '/concepts/adapters' },
            { text: 'Manifests', link: '/concepts/manifests' }
          ] },
        { text: 'Guide', link: '/guide/installation' },
        { text: 'Reference', link: '/reference/cli' },
        { text: 'Releases', link: 'https://github.com/re-cinq/wave/releases' }
      ],
      sidebar: {
        '/concepts/': [{ text: 'Concepts', items: [
            { text: 'Pipelines', link: '/concepts/pipelines' },
            { text: 'Personas', link: '/concepts/personas' },
            { text: 'Contracts', link: '/concepts/contracts' },
            { text: 'Artifacts', link: '/concepts/artifacts' },
            { text: 'Workspaces', link: '/concepts/workspaces' },
            { text: 'Adapters', link: '/concepts/adapters' },
            { text: 'Manifests', link: '/concepts/manifests' }
          ] }],
        '/guide/': [
          {
            text: 'Getting Started',
            items: [
              { text: 'Installation', link: '/guide/installation' },
              { text: 'Quick Start', link: '/guide/quick-start' },
              { text: 'Configuration', link: '/guide/configuration' }
            ]
          },
          {
            text: 'Building Pipelines',
            items: [
              { text: 'Pipelines', link: '/guide/pipelines' },
              { text: 'Pipeline Configuration', link: '/guide/pipeline-configuration' },
              { text: 'Contracts', link: '/guide/contracts' },
              { text: 'Validation', link: '/guide/validation' },
              { text: 'Retry Policies', link: '/guide/retry-policies' },
              { text: 'Personas', link: '/guide/personas' },
              { text: 'Custom Personas', link: '/guide/custom-personas' }
            ]
          },
          {
            text: 'Advanced',
            items: [
              { text: 'Meta-Pipelines', link: '/guide/meta-pipelines' },
              { text: 'Composition', link: '/guide/composition' },
              { text: 'Human Gates', link: '/guide/human-gates' },
              { text: 'Model Routing', link: '/guide/model-routing' },
              { text: 'State & Resumption', link: '/guide/state-resumption' },
              { text: 'Sandbox Setup', link: '/guide/sandbox-setup' },
              { text: 'Forge Setup', link: '/guide/forge-setup' }
            ]
          }
        ],
        '/reference/': [{ text: 'Reference', items: [
              { text: 'CLI Commands', link: '/reference/cli' },
              { text: 'Manifest Schema', link: '/reference/manifest-schema' },
              { text: 'Pipeline Schema', link: '/reference/pipeline-schema' },
              { text: 'Contract Types', link: '/reference/contract-types' },
              { text: 'Adapters', link: '/reference/adapters' },
              { text: 'Events', link: '/reference/events' },
              { text: 'Environment', link: '/reference/environment' },
              { text: 'Error Codes', link: '/reference/error-codes' },
              { text: 'Troubleshooting', link: '/reference/troubleshooting' }
            ] }],
        '/examples/': [{ text: 'Examples', items: [{ text: 'Overview', link: '/examples/' }] }]
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
