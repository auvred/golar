<template>
	<!-- @vue-expect-error -->
	<div>
		<!-- @vue-expect-error -->
		{{ missing }}
	</div>
	<!-- @vue-expect-error -->
	<div>
		<!-- @vue-expect-error -->
		{{ 1 + 1 }}
	</div>
	<!-- @vue-expect-error -->
	<div>
		{{ missing }}
	</div>
	<!-- @vue-expect-error -->
	<div>
		<div>
			<!-- @vue-expect-error -->
			{{ 1 + 1 }}
		</div>
	</div>
</template>
