<script setup lang="ts">
import { computed, ref, onMounted, watch } from 'vue'

interface OutputArtifact {
  name: string
  path: string
  type?: string
}

interface InjectArtifact {
  step: string
  artifact: string
  as?: string
}

interface Memory {
  strategy?: string
  inject_artifacts?: InjectArtifact[]
}

interface PipelineStep {
  id: string
  persona: string
  dependencies?: string[]
  output_artifacts?: OutputArtifact[]
  memory?: Memory
}

interface PipelineMetadata {
  name: string
  description?: string
}

interface Pipeline {
  kind: string
  metadata: PipelineMetadata
  steps: PipelineStep[]
}

const props = defineProps<{
  pipeline: Pipeline | string
  interactive?: boolean
  showArtifacts?: boolean
}>()

const selectedStep = ref<string | null>(null)
const mermaidContainer = ref<HTMLElement | null>(null)

// Parse pipeline if string (YAML needs to be pre-parsed)
const parsedPipeline = computed(() => {
  if (typeof props.pipeline === 'string') {
    try {
      // Assume it's already JSON if string
      return JSON.parse(props.pipeline)
    } catch {
      return null
    }
  }
  return props.pipeline
})

// Generate Mermaid flowchart from pipeline
const mermaidCode = computed(() => {
  const pipeline = parsedPipeline.value
  if (!pipeline || !pipeline.steps) return ''

  const lines: string[] = ['flowchart TD']
  const stepMap = new Map<string, PipelineStep>()

  // Build step map
  pipeline.steps.forEach((step: PipelineStep) => {
    stepMap.set(step.id, step)
  })

  // Add step nodes with persona labels
  pipeline.steps.forEach((step: PipelineStep) => {
    const label = `${step.id}<br/><small>${step.persona}</small>`
    lines.push(`  ${step.id}["${label}"]`)
  })

  // Add dependency edges
  pipeline.steps.forEach((step: PipelineStep) => {
    if (step.dependencies && step.dependencies.length > 0) {
      step.dependencies.forEach((dep: string) => {
        lines.push(`  ${dep} --> ${step.id}`)
      })
    }
  })

  // Add artifact flow annotations if enabled
  if (props.showArtifacts !== false) {
    pipeline.steps.forEach((step: PipelineStep) => {
      if (step.memory?.inject_artifacts) {
        step.memory.inject_artifacts.forEach((artifact: InjectArtifact) => {
          const sourceStep = stepMap.get(artifact.step)
          if (sourceStep) {
            // Find the matching output artifact
            const outputArtifact = sourceStep.output_artifacts?.find(
              (a: OutputArtifact) => a.name === artifact.artifact
            )
            if (outputArtifact) {
              lines.push(`  ${artifact.step} -.->|"${artifact.artifact}"| ${step.id}`)
            }
          }
        })
      }
    })
  }

  // Style nodes by persona
  const personaColors: Record<string, string> = {
    navigator: '#4a90d9',
    auditor: '#d94a4a',
    craftsman: '#4ad94a',
    philosopher: '#d9a44a',
    summarizer: '#9a4ad9'
  }

  pipeline.steps.forEach((step: PipelineStep) => {
    const color = personaColors[step.persona] || '#666'
    lines.push(`  style ${step.id} fill:${color},color:#fff`)
  })

  return lines.join('\n')
})

// Step details for interactive mode
const stepDetails = computed(() => {
  if (!selectedStep.value || !parsedPipeline.value) return null
  return parsedPipeline.value.steps.find(
    (s: PipelineStep) => s.id === selectedStep.value
  )
})

// Render Mermaid diagram
const renderMermaid = async () => {
  if (!mermaidContainer.value || !mermaidCode.value) return

  // Check if mermaid is available (loaded by vitepress-plugin-mermaid)
  if (typeof window !== 'undefined' && (window as any).mermaid) {
    const mermaid = (window as any).mermaid
    try {
      const { svg } = await mermaid.render('pipeline-diagram', mermaidCode.value)
      mermaidContainer.value.innerHTML = svg

      // Add click handlers for interactive mode
      if (props.interactive) {
        const nodes = mermaidContainer.value.querySelectorAll('.node')
        nodes.forEach((node) => {
          node.style.cursor = 'pointer'
          node.addEventListener('click', () => {
            const id = node.id.replace('flowchart-', '').split('-')[0]
            selectedStep.value = selectedStep.value === id ? null : id
          })
        })
      }
    } catch (e) {
      console.error('Mermaid rendering error:', e)
      mermaidContainer.value.innerHTML = `<pre class="mermaid-fallback">${mermaidCode.value}</pre>`
    }
  } else {
    // Fallback: show raw mermaid code
    mermaidContainer.value.innerHTML = `<pre class="mermaid">${mermaidCode.value}</pre>`
  }
}

onMounted(() => {
  renderMermaid()
})

watch(mermaidCode, () => {
  renderMermaid()
})
</script>

<template>
  <div class="pipeline-visualizer">
    <div class="visualizer-header" v-if="parsedPipeline?.metadata">
      <h3>{{ parsedPipeline.metadata.name }}</h3>
      <p v-if="parsedPipeline.metadata.description">
        {{ parsedPipeline.metadata.description }}
      </p>
    </div>

    <div class="diagram-container" ref="mermaidContainer">
      <div class="loading">Loading diagram...</div>
    </div>

    <div class="step-details" v-if="interactive && stepDetails">
      <h4>Step: {{ stepDetails.id }}</h4>
      <dl>
        <dt>Persona</dt>
        <dd>{{ stepDetails.persona }}</dd>

        <template v-if="stepDetails.dependencies?.length">
          <dt>Dependencies</dt>
          <dd>{{ stepDetails.dependencies.join(', ') }}</dd>
        </template>

        <template v-if="stepDetails.output_artifacts?.length">
          <dt>Output Artifacts</dt>
          <dd>
            <ul>
              <li v-for="artifact in stepDetails.output_artifacts" :key="artifact.name">
                <code>{{ artifact.name }}</code> ({{ artifact.type || 'file' }})
              </li>
            </ul>
          </dd>
        </template>

        <template v-if="stepDetails.memory?.inject_artifacts?.length">
          <dt>Injected Artifacts</dt>
          <dd>
            <ul>
              <li v-for="artifact in stepDetails.memory.inject_artifacts" :key="artifact.artifact">
                <code>{{ artifact.artifact }}</code> from {{ artifact.step }}
              </li>
            </ul>
          </dd>
        </template>
      </dl>
    </div>

    <div class="legend">
      <span class="legend-item">
        <span class="legend-color navigator"></span> Navigator
      </span>
      <span class="legend-item">
        <span class="legend-color auditor"></span> Auditor
      </span>
      <span class="legend-item">
        <span class="legend-color craftsman"></span> Craftsman
      </span>
      <span class="legend-item">
        <span class="legend-color philosopher"></span> Philosopher
      </span>
      <span class="legend-item">
        <span class="legend-color summarizer"></span> Summarizer
      </span>
    </div>
  </div>
</template>

<style scoped>
.pipeline-visualizer {
  padding: 24px;
  background: var(--vp-c-bg-soft);
  border: 1px solid var(--vp-c-divider);
  border-radius: 12px;
  margin: 24px 0;
}

.visualizer-header {
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--vp-c-divider);
}

.visualizer-header h3 {
  margin: 0 0 8px 0;
  font-size: 1.25rem;
  font-weight: 600;
}

.visualizer-header p {
  margin: 0;
  color: var(--vp-c-text-2);
  font-size: 0.9rem;
}

.diagram-container {
  display: flex;
  justify-content: center;
  padding: 20px 0;
  overflow-x: auto;
}

.diagram-container :deep(svg) {
  max-width: 100%;
  height: auto;
}

.diagram-container .loading {
  color: var(--vp-c-text-2);
  font-style: italic;
}

.mermaid-fallback {
  background: var(--vp-c-bg);
  padding: 16px;
  border-radius: 8px;
  font-size: 12px;
  overflow-x: auto;
}

.step-details {
  margin-top: 20px;
  padding: 16px;
  background: var(--vp-c-bg);
  border: 1px solid var(--vp-c-divider);
  border-radius: 8px;
}

.step-details h4 {
  margin: 0 0 12px 0;
  font-size: 1rem;
  font-weight: 600;
  color: var(--vp-c-brand-1);
}

.step-details dl {
  margin: 0;
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 8px 16px;
}

.step-details dt {
  font-weight: 600;
  color: var(--vp-c-text-2);
}

.step-details dd {
  margin: 0;
}

.step-details ul {
  margin: 0;
  padding-left: 20px;
}

.step-details code {
  background: var(--vp-c-bg-soft);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 0.85em;
}

.legend {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid var(--vp-c-divider);
  justify-content: center;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.85rem;
  color: var(--vp-c-text-2);
}

.legend-color {
  width: 14px;
  height: 14px;
  border-radius: 4px;
}

.legend-color.navigator { background: #4a90d9; }
.legend-color.auditor { background: #d94a4a; }
.legend-color.craftsman { background: #4ad94a; }
.legend-color.philosopher { background: #d9a44a; }
.legend-color.summarizer { background: #9a4ad9; }
</style>
