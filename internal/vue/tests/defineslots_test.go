package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
)

func TestDuplicateDefineSlotsCallExpression(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts" setup>
	const p = defineSlots<{ default(props: { msg: string }): any}>()
	[|defineSlots|]<{ default(props: { msg: string }): any}>()
</script>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		switch version {
		case vue_3_2:
			f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
				{
					Range:   lsproto.Range{Start: lsproto.Position{Line: 1, Character: 11}, End: lsproto.Position{Line: 1, Character: 22}},
					Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2304)},
					Message: "Cannot find name 'defineSlots'.",
				},
				{
					Range:   lsproto.Range{Start: lsproto.Position{Line: 2, Character: 1}, End: lsproto.Position{Line: 2, Character: 12}},
					Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2304)},
					Message: "Cannot find name 'defineSlots'.",
				},
			})
		default:
			f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
				{
					Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_007)},
					Message: "Duplicate defineSlots call.",
				},
			})
		}
	})
}

func TestDuplicateDefineSlotsVariableDecl(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts" setup>
	defineSlots<{ default(props: { msg: string }): any }>()
	const p = [|defineSlots|]<{ default(props: { msg: string }): any }>()
</script>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		switch version {
		case vue_3_2:
			f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
				{
					Range:   lsproto.Range{Start: lsproto.Position{Line: 1, Character: 1}, End: lsproto.Position{Line: 1, Character: 12}},
					Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2304)},
					Message: "Cannot find name 'defineSlots'.",
				},
				{
					Range:   lsproto.Range{Start: lsproto.Position{Line: 2, Character: 11}, End: lsproto.Position{Line: 2, Character: 22}},
					Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2304)},
					Message: "Cannot find name 'defineSlots'.",
				},
			})
		default:
			f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
				{
					Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_007)},
					Message: "Duplicate defineSlots call.",
				},
			})
		}
	})
}
