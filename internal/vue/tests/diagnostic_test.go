package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
)

func TestDiagnostic(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts" setup>
	const [|foo|]: string = 5
</script>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2322)},
				Message: "Type 'number' is not assignable to type 'string'.",
			},
		})
	})
}

func TestGlobalsNotPrefixed(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts" setup>
	const loading = true
</script>

<template>
	<div :title="loading ? 'yes' : undefined"></div>
	<div :title="loading ? String(42) : undefined"></div>
	<div :title="loading ? JSON.stringify({}) : undefined"></div>
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}

func TestComponentOptionalPropsNoError(t *testing.T) {
	runFourslashTest(t, `// @filename: Comp.vue
<script setup lang="ts">
defineProps<{
	readonly variant: 'block' | 'curved' | 'round'
	readonly direction: 'column' | 'row'
	readonly align?: string
	readonly maxWidth?: string
	readonly height?: string
}>()
</script>

<template><div><slot /></div></template>

// @filename: file.vue
<script lang="ts" setup>
	import Comp from './Comp.vue'
</script>

<template>
	<Comp align="middle-left" max-width="300px" height="hug-content" />
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		// Should only get errors about missing required props (variant, direction),
		// NOT about the optional ones (align, maxWidth, height)
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}

func TestComponentWithDefaultsNoError(t *testing.T) {
	runFourslashTest(t, `// @filename: Comp.vue
<script setup lang="ts">
withDefaults(defineProps<{
	variant?: 'block' | 'curved' | 'round'
	direction?: 'column' | 'row'
	state?: string
	gap?: string
	align?: string
	maxWidth?: string
	height?: string
}>(), {
	variant: 'block',
	direction: 'row',
	state: 'default',
	gap: 'none',
})
</script>

<template><div><slot /></div></template>

// @filename: file.vue
<script lang="ts" setup>
	import Comp from './Comp.vue'
</script>

<template>
	<Comp align="middle-left" max-width="300px" height="hug-content" />
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		// All props have defaults or are optional - should be no errors
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}

func TestComponentDefineComponentPropsNoError(t *testing.T) {
	runFourslashTest(t, `// @filename: Comp.vue
<script lang="ts">
import { defineComponent } from 'vue'
export default defineComponent({
	props: {
		variant: { type: String as () => 'block' | 'curved' | 'round', default: 'block' },
		direction: { type: String as () => 'column' | 'row', default: 'row' },
		align: { type: String, default: undefined },
		maxWidth: { type: String, default: undefined },
		height: { type: String, default: undefined },
	},
})
</script>

<template><div><slot /></div></template>

// @filename: file.vue
<script lang="ts" setup>
	import Comp from './Comp.vue'
</script>

<template>
	<Comp align="middle-left" max-width="300px" height="hug-content" />
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		// All props have defaults - should be no errors about missing props
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}

func TestVueSyntaxError(t *testing.T) {
	// TODO:
	t.Skip()
	runFourslashTest(t, `// @filename: file.vue
<template>
	<div>
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2322)},
				Message: "Type 'number' is not assignable to type 'string'.",
			},
		})
	})
}
