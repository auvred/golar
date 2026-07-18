<template>
	<div>
		<!-- Regression: the suppressed diagnostic (TS2339) is one of golar's built-in
		     SUPPRESSED_DIAGNOSTIC_CODES. Before the fix, golar baked it into
		     __suppressedDiagnostics, starving the generated `// @ts-expect-error` and
		     reporting a spurious "Unused '@ts-expect-error' directive" (TS2578). -->
		<!-- @vue-expect-error -->
		{{ obj.missing }}
	</div>
</template>

<script setup lang="ts">
	const obj = { n: 1 }
</script>
