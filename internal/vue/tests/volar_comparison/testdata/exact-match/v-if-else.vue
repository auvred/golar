<script setup lang="ts">
const type = ref<'a' | 'b' | 'c'>('a')
const showExtra = ref(false)
</script>
<template>
  <div v-if="type === 'a'">Type A</div>
  <div v-else-if="type === 'b'">Type B</div>
  <div v-else>Other: {{ type }}</div>
  <span v-if="showExtra">Extra</span>
</template>
