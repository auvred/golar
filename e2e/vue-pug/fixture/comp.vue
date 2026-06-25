<script lang="ts" setup>
	function handleClick(message: string) {
		return message
	}
</script>

<template lang="pug">
button(@click="handleClick(123)") Click
</template>
