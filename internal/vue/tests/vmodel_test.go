package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
)

func TestVModelBasic(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts" setup>
import { ref } from 'vue'
const text = ref('')
</script>

<template>
	<input v-model="text" />
	{{ text/*1*/ }}
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyQuickInfoAt(t, "1", `(property) text: string`, "") // Refs are auto-unwrapped in templates
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}

func TestVModelWithArg(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts" setup>
import { ref } from 'vue'
const value = ref(42)
</script>

<template>
	<input v-model:value="value" />
	{{ value/*1*/ }}
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyQuickInfoAt(t, "1", `(property) value: number`, "") // Refs are auto-unwrapped in templates
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}

func TestVModelCheckbox(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts" setup>
import { ref } from 'vue'
const checked = ref(false)
</script>

<template>
	<input type="checkbox" v-model="checked" />
	{{ checked/*1*/ }}
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyQuickInfoAt(t, "1", `(property) checked: boolean`, "") // Refs are auto-unwrapped in templates
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}

func TestVModelMultiple(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts" setup>
import { ref } from 'vue'
const text = ref('')
const num = ref(0)
const checked = ref(false)
</script>

<template>
	<input v-model="text" />
	<input v-model:value="num" type="number" />
	<input v-model="checked" type="checkbox" />
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}
