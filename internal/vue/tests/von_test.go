package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
	"github.com/microsoft/typescript-go/shim/testutil"
)

func TestVOnSimpleHandler(t *testing.T) {
	t.Parallel()

	defer testutil.RecoverAndFail(t, "Panic on fourslash test")
	content := withVueNodeModules(t, vue_3_5, `// @filename: file.vue
<script setup lang="ts">
const handleClick = () => {}
</script>

<template>
	<button @click="handleClick/*1*/"></button>
</template>`)
	f, done := fourslash.NewFourslash(t, nil, content)
	defer done()

	f.VerifyQuickInfoAt(t, "1", "(property) handleClick: () => void", "")
	f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
}

func TestVOnCompoundExpression(t *testing.T) {
	t.Parallel()

	defer testutil.RecoverAndFail(t, "Panic on fourslash test")
	content := withVueNodeModules(t, vue_3_5, `// @filename: file.vue
<script setup lang="ts">
let count/*1*/ = 0
</script>

<template>
	<button @click="count/*2*/++"></button>
</template>`)
	f, done := fourslash.NewFourslash(t, nil, content)
	defer done()

	f.VerifyQuickInfoAt(t, "1", "let count: number", "")
	f.VerifyQuickInfoAt(t, "2", "(property) count: number", "")
	f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
}

func TestVOnPropertyAccess(t *testing.T) {
	t.Parallel()

	defer testutil.RecoverAndFail(t, "Panic on fourslash test")
	content := withVueNodeModules(t, vue_3_5, `// @filename: file.vue
<script setup lang="ts">
const obj = {
	method: () => {}
}
</script>

<template>
	<button @click="obj/*1*/.method/*2*/"></button>
</template>`)
	f, done := fourslash.NewFourslash(t, nil, content)
	defer done()

	f.VerifyQuickInfoAt(t, "1", "(property) obj: { method: () => void; }", "")
	f.VerifyQuickInfoAt(t, "2", "(property) method: () => void", "")
	f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
}

func TestVOnInlineArrowFunction(t *testing.T) {
	t.Parallel()

	defer testutil.RecoverAndFail(t, "Panic on fourslash test")
	content := withVueNodeModules(t, vue_3_5, `// @filename: file.vue
<script setup lang="ts">
let count = 0
</script>

<template>
	<button @click="() => count/*1*/++"></button>
</template>`)
	f, done := fourslash.NewFourslash(t, nil, content)
	defer done()

	// Inside user-provided arrow function, count is accessed via __VLS_Ctx (property)
	f.VerifyQuickInfoAt(t, "1", "(property) count: number", "")
	f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
}

func TestVOnWithEventParameter(t *testing.T) {
	t.Parallel()

	defer testutil.RecoverAndFail(t, "Panic on fourslash test")
	content := withVueNodeModules(t, vue_3_5, `// @filename: file.vue
<script setup lang="ts">
const handleClick = (e: MouseEvent) => {
	console.log(e.clientX)
}
</script>

<template>
	<button @click="handleClick/*1*/($event/*2*/)"></button>
</template>`)
	f, done := fourslash.NewFourslash(t, nil, content)
	defer done()

	f.VerifyQuickInfoAt(t, "1", "(property) handleClick: (e: MouseEvent) => void", "")
	// $event should be available in compound expressions
	f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
}

func TestVOnLongForm(t *testing.T) {
	t.Parallel()

	defer testutil.RecoverAndFail(t, "Panic on fourslash test")
	content := withVueNodeModules(t, vue_3_5, `// @filename: file.vue
<script setup lang="ts">
const handleClick = () => {}
</script>

<template>
	<button v-on:click="handleClick/*1*/"></button>
</template>`)
	f, done := fourslash.NewFourslash(t, nil, content)
	defer done()

	f.VerifyQuickInfoAt(t, "1", "(property) handleClick: () => void", "")
	f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
}

func TestVOnMultiStatementRefAssign(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script setup lang="ts">
import { ref } from 'vue'
const showMenu = ref(false)
const showConfirm = ref(false)
const count = ref(0)
</script>

<template>
	<button @click="
		showConfirm = false;
		showMenu = false;
		count = 0;
	"></button>
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}

func TestVOnTypeError(t *testing.T) {
	t.Parallel()

	defer testutil.RecoverAndFail(t, "Panic on fourslash test")
	content := withVueNodeModules(t, vue_3_5, `// @filename: file.vue
<script setup lang="ts">
const notAFunction: number = 42
</script>

<template>
	<button @click="[|notAFunction|]()"></button>
</template>`)
	f, done := fourslash.NewFourslash(t, nil, content)
	defer done()

	f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
		{
			Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2349)},
			Message: "This expression is not callable.\n  Type 'Number' has no call signatures.",
		},
	})
}
