<script setup lang="ts">
import { ref } from 'vue'

const items = ref([
  { id: 1, name: 'foo' },
  { id: 2, name: 'bar' },
])
</script>
<template>
  <ul>
    <li v-for="{ id, name } in items" :key="id">
      {{ name }} - {{ id }}
    </li>
  </ul>
  <slot name="footer" :count="items.length" />
</template>
