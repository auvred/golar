<script setup lang="ts">
import ChildComp from './child.vue'

const name = ref('hello')
</script>
<template>
  <ChildComp :foo="name" bar="static" />
  <ChildComp foo="missing-bar" />
</template>
