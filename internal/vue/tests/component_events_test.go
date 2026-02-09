package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
)

func TestComponentEventCallback(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script setup lang="ts">
	import CompFoo from './file-foo.vue'

	function foo(id: number) {}
	function bar(name: string) {}
	const callbacks = {
		foo,
		bar,
	}
</script>

<template>
	<CompFoo @foo="id => id/*1*/"/>
	<CompFoo @foo="((id => id/*2*/))"/>
	<CompFoo @foo="foo"/>
	<CompFoo @foo="callbacks.foo"/>
	<CompFoo @foo="function (id) { id/*3*/ }"/>
	<CompFoo @foo="((function (id) { id/*4*/ }))"/>
	<CompFoo @foo="function fn(id) { id/*5*/ }"/>

	<CompFoo [|@foo|]="bar"/>
	<CompFoo [|@foo|]="callbacks.bar"/>
</template>

// @filename: file-foo.vue
<script setup lang="ts">
	defineEmits<{ (e: 'foo', id: number): void }>()
</script>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyQuickInfoAt(t, "1", `(parameter) id: number`, "")
		f.VerifyQuickInfoAt(t, "2", `(parameter) id: number`, "")
		f.VerifyQuickInfoAt(t, "3", `(parameter) id: number`, "")
		f.VerifyQuickInfoAt(t, "4", `(parameter) id: number`, "")
		f.VerifyQuickInfoAt(t, "5", `(parameter) id: number`, "")
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
			{
				Code: &lsproto.IntegerOrString{Integer: ptrTo[int32](2322)},
				Message: `Type '(name: string) => void' is not assignable to type '(id: number) => any'.
  Types of parameters 'name' and 'args' are incompatible.
    Type 'number' is not assignable to type 'string'.`,
			},
			{
				Code: &lsproto.IntegerOrString{Integer: ptrTo[int32](2322)},
				Message: `Type '(name: string) => void' is not assignable to type '(id: number) => any'.
  Types of parameters 'name' and 'args' are incompatible.
    Type 'number' is not assignable to type 'string'.`,
			},
		})
	})
}

func TestComponentEventCompound(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script setup lang="ts">
	import CompFoo from './file-foo.vue'
	let foo: number = 1
	defineProps<{ bar: number }>()
	function handle(id: number) {}
</script>

<template>
	<CompFoo @foo="handle($event/*1*/)"/>
	<CompFoo @foo="foo = $event/*2*/"/>
	<CompFoo @foo="((foo = $event/*3*/))"/>
	<CompFoo @foo="foo = $props.bar; foo += $event"/>
</template>

// @filename: file-foo.vue
<script setup lang="ts">
	defineEmits<{ (e: 'foo', id: number): void }>()
</script>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyQuickInfoAt(t, "1", `var $event: number`, "")
		f.VerifyQuickInfoAt(t, "2", `var $event: number`, "")
		f.VerifyQuickInfoAt(t, "3", `var $event: number`, "")
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}

func TestComponentEventFunctionSetupVars(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script setup lang="ts">
	import CompFoo from './file-foo.vue'

	const foo = 'bar'
</script>

<template>
	<CompFoo @foo="() => foo/*1*/"/>
</template>

// @filename: file-foo.vue
<script setup lang="ts">
	defineEmits<{ (e: 'foo'): void }>()
</script>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyQuickInfoAt(t, "1", `(property) foo: "bar"`, "")
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}

func TestComponentEventEmptyListener(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script setup lang="ts">
	import CompFoo from './file-foo.vue'
</script>

<template>
	<CompFoo @foo=""/>
</template>

// @filename: file-foo.vue
<script setup lang="ts">
	defineEmits<{ (e: 'foo'): void }>()
</script>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}

func TestDefineModelEvent(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script setup lang="ts">
	import CompFoo from './file-foo.vue'
</script>

<template>
	<CompFoo @update:fooBar="v => v/*1*/"/>
</template>

// @filename: file-foo.vue
<script setup lang="ts">
	defineModel<string>('foo-bar')
</script>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		switch version {
		case vue_3_2, vue_3_3:
			return
		}
		f.VerifyQuickInfoAt(t, "1", `(parameter) v: string`, "")
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}
