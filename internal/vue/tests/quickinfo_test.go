package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
)

func TestQuickInfo(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts" setup>
	const foo/*1*/ = 'hello'
</script>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyQuickInfoAt(t, "1", `const foo: "hello"`, "")
	})
}
