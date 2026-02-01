// VitePress theme entry point
import DefaultTheme from 'vitepress/theme'
import WaveConfig from './components/WaveConfig.vue'
import TerminalOutput from './components/TerminalOutput.vue'

export default {
  extends: DefaultTheme,
  enhanceApp({ app }) {
    app.component('WaveConfig', WaveConfig)
    app.component('TerminalOutput', TerminalOutput)
  }
}