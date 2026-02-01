<script setup>
defineProps({
  modelValue: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['update'])

function updateValue(event) {
  emit('update', {
    path: event.target.dataset.path,
    value: event.target.value
  })
}
</script>

<template>
  <div class="wave-config">
    <h3>Manifest Configuration</h3>
    <div class="config-section">
      <label>API Version:</label>
      <select :data-path="'apiVersion'" :value="modelValue.apiVersion" @input="updateValue">
        <option value="v1">v1</option>
      </select>
      <label>Project Name:</label>
      <input
        type="text"
        :data-path="'metadata.name'"
        :value="modelValue.metadata?.name || ''"
        @input="updateValue"
      />
    </div>
  </div>
</template>

<style scoped>
.wave-config {
  border: 1px solid var(--vp-c-divider);
  border-radius: 8px;
  padding: 16px;
  margin: 1rem 0;
}
.config-section {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 8px;
  align-items: center;
}
.config-section label {
  font-weight: 600;
}
.config-section select,
.config-section input {
  padding: 4px 8px;
  border: 1px solid var(--vp-c-divider);
  border-radius: 4px;
}
</style>
