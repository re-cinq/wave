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
import PlatformTabs from './components/PlatformTabs.vue'
import PermissionMatrix from './components/PermissionMatrix.vue'
import HeroSection from './components/HeroSection.vue'
import FeatureCards from './components/FeatureCards.vue'
import TrustSignals from './components/TrustSignals.vue'

// Pipeline Learning Components
import PipelineVisualizer from './components/PipelineVisualizer.vue'
import YamlPlayground from './components/YamlPlayground.vue'

// Use Case Discovery Components
import UseCaseGallery from './components/UseCaseGallery.vue'

// Navigation Components
import Breadcrumb from './components/Breadcrumb.vue'
import CardGrid from './components/CardGrid.vue'

// Installation Components
import InstallTabs from './components/InstallTabs.vue'

export default {
  extends: DefaultTheme,
  enhanceApp({ app }) {
    // Register existing components
    app.component('WaveConfig', WaveConfig)
    app.component('TerminalOutput', TerminalOutput)

    // Register new enterprise components
    app.component('CopyButton', CopyButton)
    app.component('PlatformTabs', PlatformTabs)
    app.component('PermissionMatrix', PermissionMatrix)
    app.component('HeroSection', HeroSection)
    app.component('FeatureCards', FeatureCards)
    app.component('TrustSignals', TrustSignals)

    // Register pipeline learning components
    app.component('PipelineVisualizer', PipelineVisualizer)
    app.component('YamlPlayground', YamlPlayground)

    // Register use case discovery components
    app.component('UseCaseGallery', UseCaseGallery)

    // Register navigation components
    app.component('Breadcrumb', Breadcrumb)
    app.component('CardGrid', CardGrid)

    // Register installation components
    app.component('InstallTabs', InstallTabs)
  }
} satisfies Theme