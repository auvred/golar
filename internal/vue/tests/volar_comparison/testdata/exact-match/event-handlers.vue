<script setup lang="ts">
import { ref } from 'vue'

const count = ref(0)
const showMenu = ref(false)
const showConfirm = ref(false)

function handleClick() {
  console.log('clicked')
}
</script>
<template>
  <button @click="handleClick">Simple</button>
  <button @click="count++">Increment</button>
  <button @click="
    showConfirm = false;
    showMenu = false;
    count = 0;
  ">Reset</button>
  <button @click.stop.prevent="handleClick">With modifiers</button>
</template>
