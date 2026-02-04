// VitePress theme entry point
import DefaultTheme from 'vitepress/theme'
import type { Theme } from 'vitepress'

// Styles
import './styles/custom.css'
import './styles/components.css'

// Existing Components
import WaveConfig from './components/WaveConfig.vue'
import TerminalOutput from './components/TerminalOutput.vue'

// New Enterprise Components
import CopyButton from './components/CopyButton.vue'

// Plugins
import { setupCopyCode, injectCopyCodeStyles } from '../plugins/copy-code'

export default {
  extends: DefaultTheme,
  enhanceApp({ app }) {
    // Register existing components
    app.component('WaveConfig', WaveConfig)
    app.component('TerminalOutput', TerminalOutput)

    // Register new enterprise components
    app.component('CopyButton', CopyButton)
  },
  setup() {
    // Client-side only setup
    if (typeof window !== 'undefined') {
      injectCopyCodeStyles()
      setupCopyCode()
    }
  }
} satisfies Theme