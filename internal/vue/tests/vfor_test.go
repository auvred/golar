package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
)

func TestVForStringSource(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script setup lang="ts">
	const str = 'hello'
</script>

<template>
	<div v-for="value/*1*/ in str/*2*/">
		{{ value/*3*/ }}
	</div>
	<div v-for="(value/*4*/, idx/*5*/) in str">
		{{ value/*6*/ }}
		{{ idx/*7*/ }}
	</div>
	<div v-for="(value/*8*/, key/*9*/, [|idx/*10*/|]) in str">
		{{ value/*11*/ }}
		{{ key/*12*/ }}
		{{ idx/*13*/ }}
	</div>
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {

		f.VerifyQuickInfoAt(t, "1", "const value: string", "")
		f.VerifyQuickInfoAt(t, "2", `(property) str: "hello"`, "")
		f.VerifyQuickInfoAt(t, "3", "const value: string", "")
		f.VerifyQuickInfoAt(t, "4", "const value: string", "")
		f.VerifyQuickInfoAt(t, "5", "const idx: number", "")
		f.VerifyQuickInfoAt(t, "6", "const value: string", "")
		f.VerifyQuickInfoAt(t, "7", "const idx: number", "")
		f.VerifyQuickInfoAt(t, "8", "const value: string", "")
		f.VerifyQuickInfoAt(t, "9", "const key: number", "")
		f.VerifyQuickInfoAt(t, "10", "const idx: undefined", "")
		f.VerifyQuickInfoAt(t, "11", "const value: string", "")
		f.VerifyQuickInfoAt(t, "12", "const key: number", "")
		f.VerifyQuickInfoAt(t, "13", "const idx: undefined", "")

		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2493)},
				Message: "Tuple type '[string, number]' of length '2' has no element at index '2'.",
			},
		})
	})
}

func TestVForNumberSource(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script setup lang="ts">
	const count = 5
</script>

<template>
	<div v-for="value/*1*/ in count">
		{{ value/*2*/ }}
	</div>
	<div v-for="(value/*3*/, idx/*4*/) in count">
		{{ value/*5*/ }}
		{{ idx/*6*/ }}
	</div>
	<div v-for="(value/*7*/, key/*8*/, [|idx/*9*/|]) in count">
		{{ value/*10*/ }}
		{{ key/*11*/ }}
		{{ idx/*12*/ }}
	</div>
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {

		f.VerifyQuickInfoAt(t, "1", "const value: number", "")
		f.VerifyQuickInfoAt(t, "2", "const value: number", "")
		f.VerifyQuickInfoAt(t, "3", "const value: number", "")
		f.VerifyQuickInfoAt(t, "4", "const idx: number", "")
		f.VerifyQuickInfoAt(t, "5", "const value: number", "")
		f.VerifyQuickInfoAt(t, "6", "const idx: number", "")
		f.VerifyQuickInfoAt(t, "7", "const value: number", "")
		f.VerifyQuickInfoAt(t, "8", "const key: number", "")
		f.VerifyQuickInfoAt(t, "9", "const idx: undefined", "")
		f.VerifyQuickInfoAt(t, "10", "const value: number", "")
		f.VerifyQuickInfoAt(t, "11", "const key: number", "")
		f.VerifyQuickInfoAt(t, "12", "const idx: undefined", "")

		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2493)},
				Message: "Tuple type '[number, number]' of length '2' has no element at index '2'.",
			},
		})
	})
}

func TestVForArraySource(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script setup lang="ts">
	const arr = ['a', 'b', 'c']
</script>

<template>
	<div v-for="value/*1*/ in arr">
		{{ value/*2*/ }}
	</div>
	<div v-for="(value/*3*/, idx/*4*/) in arr">
		{{ value/*5*/ }}
		{{ idx/*6*/ }}
	</div>
	<div v-for="(value/*7*/, key/*8*/, [|idx/*9*/|]) in arr">
		{{ value/*10*/ }}
		{{ key/*11*/ }}
		{{ idx/*12*/ }}
	</div>
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {

		f.VerifyQuickInfoAt(t, "1", "const value: string", "")
		f.VerifyQuickInfoAt(t, "2", "const value: string", "")
		f.VerifyQuickInfoAt(t, "3", "const value: string", "")
		f.VerifyQuickInfoAt(t, "4", "const idx: number", "")
		f.VerifyQuickInfoAt(t, "5", "const value: string", "")
		f.VerifyQuickInfoAt(t, "6", "const idx: number", "")
		f.VerifyQuickInfoAt(t, "7", "const value: string", "")
		f.VerifyQuickInfoAt(t, "8", "const key: number", "")
		f.VerifyQuickInfoAt(t, "9", "const idx: undefined", "")
		f.VerifyQuickInfoAt(t, "10", "const value: string", "")
		f.VerifyQuickInfoAt(t, "11", "const key: number", "")
		f.VerifyQuickInfoAt(t, "12", "const idx: undefined", "")

		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2493)},
				Message: "Tuple type '[string, number]' of length '2' has no element at index '2'.",
			},
		})
	})
}

func TestVForObjectSource(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script setup lang="ts">
	const [|obj|] = { alpha: true }
</script>

<template>
	<div v-for="value/*1*/ in obj">
		{{ value/*2*/ }}
	</div>
	<div v-for="(value/*3*/, idx/*4*/) in obj">
		{{ value/*5*/ }}
		{{ idx/*6*/ }}
	</div>
	<div v-for="(value/*7*/, key/*8*/, idx/*9*/) in obj">
		{{ value/*10*/ }}
		{{ key/*11*/ }}
		{{ idx/*12*/ }}
	</div>
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {

		f.VerifyQuickInfoAt(t, "1", "const value: boolean", "")
		f.VerifyQuickInfoAt(t, "2", "const value: boolean", "")
		f.VerifyQuickInfoAt(t, "3", "const value: boolean", "")
		f.VerifyQuickInfoAt(t, "4", `const idx: "alpha"`, "")
		f.VerifyQuickInfoAt(t, "5", "const value: boolean", "")
		f.VerifyQuickInfoAt(t, "6", `const idx: "alpha"`, "")
		f.VerifyQuickInfoAt(t, "7", "const value: boolean", "")
		f.VerifyQuickInfoAt(t, "8", `const key: "alpha"`, "")
		f.VerifyQuickInfoAt(t, "9", "const idx: number", "")
		f.VerifyQuickInfoAt(t, "10", "const value: boolean", "")
		f.VerifyQuickInfoAt(t, "11", `const key: "alpha"`, "")
		f.VerifyQuickInfoAt(t, "12", "const idx: number", "")

		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}

func TestVForIterableSource(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script setup lang="ts">
	const set = new Set([true, false])
</script>

<template>
	<div v-for="value/*1*/ in set">
		{{ value/*2*/ }}
	</div>
	<div v-for="(value/*3*/, idx/*4*/) in set">
		{{ value/*5*/ }}
		{{ idx/*6*/ }}
	</div>
	<div v-for="value/*7*/, key/*8*/, [|idx/*9*/|] in set">
		{{ value/*10*/ }}
		{{ key/*11*/ }}
		{{ idx/*12*/ }}
	</div>
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {

		f.VerifyQuickInfoAt(t, "1", "const value: boolean", "")
		f.VerifyQuickInfoAt(t, "2", "const value: boolean", "")
		f.VerifyQuickInfoAt(t, "3", "const value: boolean", "")
		f.VerifyQuickInfoAt(t, "4", "const idx: number", "")
		f.VerifyQuickInfoAt(t, "5", "const value: boolean", "")
		f.VerifyQuickInfoAt(t, "6", "const idx: number", "")
		f.VerifyQuickInfoAt(t, "7", "const value: boolean", "")
		f.VerifyQuickInfoAt(t, "8", "const key: number", "")
		f.VerifyQuickInfoAt(t, "9", "const idx: undefined", "")
		f.VerifyQuickInfoAt(t, "10", "const value: boolean", "")
		f.VerifyQuickInfoAt(t, "11", "const key: number", "")
		f.VerifyQuickInfoAt(t, "12", "const idx: undefined", "")

		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2493)},
				Message: "Tuple type '[boolean, number]' of length '2' has no element at index '2'.",
			},
		})
	})
}
