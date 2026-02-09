<script setup lang="ts">
const props = defineProps<{
	foo: string;
}>();

const emit = defineEmits<{
	bar: [value: number];
}>();
</script>

<template>
	<div>{{ foo }}</div>
</template>
