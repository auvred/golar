package vue_tests

import (
	"testing"

	"github.com/auvred/golar/internal/vue/codegen"
	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
)

func TestGlobalTypesNoErrors(t *testing.T) {
	runFourslashTest(t, "// @filename: types.ts\n"+vue_codegen.GlobalTypes, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}
