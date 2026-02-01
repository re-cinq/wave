// VitePress theme entry point
import DefaultTheme from 'vitepress/theme'
import MuzzleConfig from './components/MuzzleConfig.vue'
import TerminalOutput from './components/TerminalOutput.vue'

export default {
  extends: DefaultTheme,
  enhanceApp({ app }) {
    app.component('MuzzleConfig', MuzzleConfig)
    app.component('TerminalOutput', TerminalOutput)
  }
}