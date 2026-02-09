package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
	"github.com/microsoft/typescript-go/shim/testutil"
)

func TestDefinePropsCallExpression(t *testing.T) {
	t.Parallel()

	defer testutil.RecoverAndFail(t, "Panic on fourslash test")
	content := withVueNodeModules(t, `// @filename: file.vue
<script lang="ts" setup>
	defineProps<{ foo: string }>()
</script>

<template>
	{{ foo/*1*/ }}
</template>`)
	f, done := fourslash.NewFourslash(t, nil, content)
	defer done()
	f.VerifyQuickInfoAt(t, "1", "(property) foo: string", "")
	f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
}

func TestDefinePropsVariableDecl(t *testing.T) {
	t.Parallel()

	defer testutil.RecoverAndFail(t, "Panic on fourslash test")
	content := withVueNodeModules(t, `// @filename: file.vue
<script lang="ts" setup>
	const p = defineProps<{ foo: string }>()
	p.foo/*1*/
</script>

<template>
	{{ foo/*2*/ }}
</template>`)
	f, done := fourslash.NewFourslash(t, nil, content)
	defer done()
	f.VerifyQuickInfoAt(t, "1", "(property) foo: string", "")
	f.VerifyQuickInfoAt(t, "2", "(property) foo: string", "")
	f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
}

func TestDuplicateDefinePropsCallExpression(t *testing.T) {
	t.Parallel()

	defer testutil.RecoverAndFail(t, "Panic on fourslash test")
	content := withVueNodeModules(t, `// @filename: file.vue
<script lang="ts" setup>
	const p = defineProps<{ foo: string }>()
	[|defineProps|]<{ foo: string }>()
</script>`)
	f, done := fourslash.NewFourslash(t, nil, content)
	defer done()

	f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
		{
			Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_006)},
			Message: "Duplicate defineProps call.",
		},
	})
}

func TestDuplicateDefinePropsVariableDecl(t *testing.T) {
	t.Parallel()

	defer testutil.RecoverAndFail(t, "Panic on fourslash test")
	content := withVueNodeModules(t, `// @filename: file.vue
<script lang="ts" setup>
	defineProps<{ foo: string }>()
	const p = [|defineProps|]<{ foo: string }>()
</script>`)
	f, done := fourslash.NewFourslash(t, nil, content)
	defer done()

	f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
		{
			Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_006)},
			Message: "Duplicate defineProps call.",
		},
	})
}
