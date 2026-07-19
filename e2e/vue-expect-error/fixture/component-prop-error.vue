<template>
	<div v-if="false"></div>
	<template v-else>
		<!-- @vue-expect-error -->
		<StringPropComponent :foo="() => 'foo'" />
	</template>
</template>

<script setup lang="ts">
	import StringPropComponent from './string-prop-component.vue'
</script>
