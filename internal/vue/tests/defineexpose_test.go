package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
)

func TestDefineExposeCallExpression(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts" setup>
	import Comp from './file-foo.vue'

	const instance = {} as unknown as InstanceType<typeof Comp>
	instance.foo/*1*/
</script>

// @filename: file-foo.vue
<script lang="ts" setup>
	defineExpose({
		foo: 1
	})
</script>
`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyQuickInfoAt(t, "1", "(property) foo: number", "")
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}

