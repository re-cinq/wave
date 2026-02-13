<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import type { Platform, PlatformContent } from '../types'

const activePlatform = ref<Platform>('macos')

const tabs: PlatformContent[] = [
  {
    platform: 'macos',
    label: 'macOS',
    content: `
<div class="install-option">
<h4>Build from Source <span class="recommended-badge">Recommended</span></h4>
<p>Requires Go 1.25+ and git:</p>
<pre><code>git clone https://github.com/re-cinq/wave.git
cd wave && make build
sudo mv wave /usr/local/bin/</code></pre>
</div>

<div class="install-option">
<h4>Install Script</h4>
<p class="install-note">Requires the repository to be public. Available once open-sourced.</p>
<pre><code>curl -fsSL https://raw.githubusercontent.com/re-cinq/wave/main/scripts/install.sh | sh</code></pre>
</div>
`
  },
  {
    platform: 'linux',
    label: 'Linux',
    content: `
<div class="install-option">
<h4>Build from Source <span class="recommended-badge">Recommended</span></h4>
<p>Requires Go 1.25+ and git:</p>
<pre><code>git clone https://github.com/re-cinq/wave.git
cd wave && make build
sudo mv wave /usr/local/bin/</code></pre>
</div>

<div class="install-option">
<h4>Install Script</h4>
<p class="install-note">Requires the repository to be public. Available once open-sourced.</p>
<pre><code>curl -fsSL https://raw.githubusercontent.com/re-cinq/wave/main/scripts/install.sh | sh</code></pre>
</div>

<div class="install-option">
<h4>Debian/Ubuntu (.deb)</h4>
<p class="install-note">Requires the repository to be public.</p>
<p>Download the <code>.deb</code> package from <a href="https://github.com/re-cinq/wave/releases">GitHub Releases</a>:</p>
<pre><code>curl -LO https://github.com/re-cinq/wave/releases/latest/download/wave_linux_amd64.deb
sudo dpkg -i wave_linux_amd64.deb</code></pre>
</div>
`
  },
  {
    platform: 'windows',
    label: 'Windows',
    content: `
<div class="install-option">
<h4>Not Yet Available</h4>
<p>Windows binaries are not yet available. Wave currently supports Linux and macOS.</p>
<p>See <a href="https://github.com/re-cinq/wave/releases">GitHub Releases</a> for available platforms.</p>
</div>
`
  }
]

// Platform icons as SVG paths
const platformIcons: Record<Platform, string> = {
  macos: 'M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm-1-13h2v6h-2zm0 8h2v2h-2z',
  linux: 'M12.504 0c-.155 0-.311.002-.465.006-.153.004-.305.014-.458.023-.459.035-.891.105-1.312.207-.42.103-.824.237-1.207.403-.383.166-.745.364-1.082.594-.337.23-.649.493-.935.789-.286.295-.546.623-.779.982-.233.358-.439.747-.617 1.165-.178.418-.328.864-.449 1.337-.121.473-.213.973-.274 1.498-.061.525-.091 1.075-.091 1.648 0 .573.03 1.123.091 1.648.061.525.153 1.025.274 1.498.121.473.271.919.449 1.337.178.418.384.807.617 1.165.233.359.493.687.779.982.286.296.598.559.935.789.337.23.699.428 1.082.594.383.166.787.3 1.207.403.421.102.853.172 1.312.207.153.009.305.019.458.023.154.004.31.006.465.006.155 0 .311-.002.465-.006.153-.004.305-.014.458-.023.459-.035.891-.105 1.312-.207.42-.103.824-.237 1.207-.403.383-.166.745-.364 1.082-.594.337-.23.649-.493.935-.789.286-.295.546-.623.779-.982.233-.358.439-.747.617-1.165.178-.418.328-.864.449-1.337.121-.473.213-.973.274-1.498.061-.525.091-1.075.091-1.648 0-.573-.03-1.123-.091-1.648-.061-.525-.153-1.025-.274-1.498-.121-.473-.271-.919-.449-1.337-.178-.418-.384-.807-.617-1.165-.233-.359-.493-.687-.779-.982-.286-.296-.598-.559-.935-.789-.337-.23-.699-.428-1.082-.594-.383-.166-.787-.3-1.207-.403-.421-.102-.853-.172-1.312-.207-.153-.009-.305-.019-.458-.023-.154-.004-.31-.006-.465-.006z',
  windows: 'M0 3.449L9.75 2.1v9.451H0m10.949-9.602L24 0v11.4H10.949M0 12.6h9.75v9.451L0 20.699M10.949 12.6H24V24l-12.9-1.801'
}

// Platform labels
const platformLabels: Record<Platform, string> = {
  macos: 'macOS',
  linux: 'Linux',
  windows: 'Windows'
}

// Auto-detect platform on mount
onMounted(() => {
  if (typeof navigator !== 'undefined') {
    const userAgent = navigator.userAgent.toLowerCase()

    if (userAgent.includes('mac')) {
      activePlatform.value = 'macos'
    } else if (userAgent.includes('win')) {
      activePlatform.value = 'windows'
    } else if (userAgent.includes('linux') || userAgent.includes('x11')) {
      activePlatform.value = 'linux'
    }
  }
})

const activeTab = computed(() => {
  return tabs.find(t => t.platform === activePlatform.value)
})

function setActivePlatform(platform: Platform) {
  activePlatform.value = platform
}
</script>

<template>
  <div class="platform-tabs">
    <div class="tabs-header">
      <button
        v-for="tab in tabs"
        :key="tab.platform"
        class="tab-button"
        :class="{ active: activePlatform === tab.platform }"
        @click="setActivePlatform(tab.platform)"
        :aria-selected="activePlatform === tab.platform"
        role="tab"
      >
        <svg
          class="platform-icon"
          viewBox="0 0 24 24"
          width="16"
          height="16"
          fill="currentColor"
        >
          <path :d="platformIcons[tab.platform]" />
        </svg>
        <span>{{ tab.label || platformLabels[tab.platform] }}</span>
      </button>
    </div>
    <div class="tab-content" role="tabpanel">
      <div v-html="activeTab?.content"></div>
    </div>
  </div>
</template>

<style scoped>
.platform-tabs {
  margin: 24px 0;
  border: 1px solid var(--vp-c-divider);
  border-radius: 8px;
  overflow: hidden;
}

.tabs-header {
  display: flex;
  gap: 0;
  background: var(--vp-c-bg-soft);
  border-bottom: 1px solid var(--vp-c-divider);
}

.tab-button {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 20px;
  font-size: 14px;
  font-weight: 500;
  color: var(--vp-c-text-2);
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  cursor: pointer;
  transition: all 0.15s ease;
  margin-bottom: -1px;
}

.tab-button:hover {
  color: var(--vp-c-text-1);
  background: var(--vp-c-bg-mute);
}

.tab-button.active {
  color: var(--vp-c-brand-1);
  background: var(--vp-c-bg);
  border-bottom-color: var(--vp-c-brand-1);
}

.platform-icon {
  flex-shrink: 0;
}

.tab-content {
  padding: 20px;
  background: var(--vp-c-bg);
}

.tab-content :deep(h4) {
  font-size: 1rem;
  font-weight: 600;
  margin-top: 0;
  margin-bottom: 12px;
  color: var(--vp-c-text-1);
}

.tab-content :deep(h4:not(:first-child)) {
  margin-top: 24px;
}

.tab-content :deep(p) {
  margin: 8px 0;
  color: var(--vp-c-text-2);
}

.tab-content :deep(pre) {
  margin: 12px 0;
  padding: 16px;
  background: var(--vp-c-bg-soft);
  border-radius: 6px;
  overflow-x: auto;
}

.tab-content :deep(code) {
  font-family: var(--vp-font-family-mono);
  font-size: 13px;
  line-height: 1.6;
}

.tab-content :deep(.install-option) {
  margin-bottom: 20px;
  padding-bottom: 20px;
  border-bottom: 1px solid var(--vp-c-divider);
}

.tab-content :deep(.install-option:last-child) {
  margin-bottom: 0;
  padding-bottom: 0;
  border-bottom: none;
}

.tab-content :deep(.install-note) {
  font-size: 13px;
  color: var(--vp-c-warning-1);
  font-style: italic;
}

.tab-content :deep(.recommended-badge) {
  display: inline-block;
  padding: 2px 8px;
  font-size: 11px;
  font-weight: 600;
  color: var(--vp-c-brand-1);
  background: var(--vp-c-brand-soft);
  border-radius: 4px;
  margin-left: 8px;
  vertical-align: middle;
}
</style>
