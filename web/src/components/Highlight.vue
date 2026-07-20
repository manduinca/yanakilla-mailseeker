<script setup>
import { computed } from 'vue'

const props = defineProps({
  text: { type: String, default: '' },
  term: { type: String, default: '' },
})

const parts = computed(() => {
  const term = props.term.trim()
  if (!term || !props.text) return [{ text: props.text, match: false }]

  const escaped = term.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const re = new RegExp(`(${escaped})`, 'gi')

  return props.text
    .split(re)
    .filter((chunk) => chunk !== '')
    .map((chunk) => ({ text: chunk, match: chunk.toLowerCase() === term.toLowerCase() }))
})
</script>

<template>
  <span
    ><template v-for="(part, i) in parts" :key="i"
      ><mark v-if="part.match" class="rounded bg-amber-200 px-0.5 text-slate-900">{{ part.text }}</mark
      ><template v-else>{{ part.text }}</template></template
    ></span
  >
</template>
