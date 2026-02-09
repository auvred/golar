package vue_codegen

import (
	_ "embed"
	"strings"

	"github.com/auvred/golar/internal/mapping"
	"github.com/auvred/golar/internal/utils"
	"github.com/auvred/golar/internal/vue/ast"
	"github.com/auvred/golar/internal/vue/diagnostics"
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/core"
	"github.com/microsoft/typescript-go/shim/diagnostics"
)

const GlobalTypesPath = utils.GolarVirtualScheme + "vue-global-types.d.ts"
const globalTypesReference = "/// <reference types=\"" + GlobalTypesPath + "\" />\n"

// GlobalTypes contains the type definitions required for Vue template type checking.
// This is copied from Volar's template-helpers.d.ts to ensure 1:1 compatibility.
// Source: https://github.com/vuejs/language-tools/blob/master/packages/language-core/types/template-helpers.d.ts
//
//go:embed types/template-helpers.d.ts
var GlobalTypes string

func Codegen(sourceText string, root *vue_ast.RootNode) (string, []mapping.Mapping, []*ast.Diagnostic) {
	ctx := newCodegenCtx(root, sourceText)
	ctx.serviceText.WriteString(globalTypesReference)

	var scriptEl *vue_ast.ElementNode
	var scriptSetupEl *vue_ast.ElementNode
	var templateEl *vue_ast.ElementNode

RootChild:
	for _, child := range root.Children {
		if child.Kind != vue_ast.KindElement {
			continue
		}

		el := child.AsElement()

		if el.Tag == "script" {
			for _, prop := range el.Props {
				if prop.Kind == vue_ast.KindAttribute {
					attr := prop.AsAttribute()
					if attr.Name == "setup" {
						if scriptSetupEl != nil {
							ctx.reportDiagnostic(el.Loc.WithEnd(el.InnerLoc.Pos()), vue_diagnostics.Single_file_component_can_contain_only_one_script_setup_element)
						} else {
							scriptSetupEl = el
						}
						continue RootChild
					}
				}
			}

			if scriptEl != nil {
				ctx.reportDiagnostic(el.Loc.WithEnd(el.InnerLoc.Pos()), vue_diagnostics.Single_file_component_can_contain_only_one_script_element)
			} else {
				scriptEl = el
			}
			continue RootChild
		}

		if el.Tag == "template" {
			if templateEl != nil {
				ctx.reportDiagnostic(el.Loc.WithEnd(el.InnerLoc.Pos()), vue_diagnostics.Single_file_component_can_contain_only_one_template_element)
				continue
			}
			templateEl = el
		}
	}

	// Preserve line count for basic line correspondence (errors show on correct line)
	// We only preserve newlines, not character counts - this prevents issues with
	// very long lines (like minified CSS) causing TypeScript parse errors.
	// Character-level mapping is handled by the source map system.
	for _, ch := range sourceText {
		if ch == '\n' {
			ctx.serviceText.WriteByte('\n')
		}
	}

	{
		c := newCodegenCtx(root, sourceText)
		generateScript(&c, scriptSetupEl, scriptEl, templateEl)
		newMappingsStart := len(ctx.mappings)
		ctx.mappings = append(ctx.mappings, c.mappings...)
		for i := newMappingsStart; i < len(ctx.mappings); i++ {
			ctx.mappings[i].ServiceOffset += ctx.serviceText.Len()
		}
		ctx.serviceText.Write([]byte(c.serviceText.String()))
		ctx.diagnostics = append(ctx.diagnostics, c.diagnostics...)
	}

	return ctx.serviceText.String(), ctx.mappings, ctx.diagnostics
}

type codegenCtx struct {
	ast         *vue_ast.RootNode
	sourceText  string
	serviceText strings.Builder
	mappings    []mapping.Mapping
	diagnostics []*ast.Diagnostic
	// setupConsts contains identifiers defined in <script setup> (imports, variables, functions)
	// Used to determine if a component tag matches a local binding vs a global component
	setupConsts map[string]bool
}

func newCodegenCtx(root *vue_ast.RootNode, sourceText string) codegenCtx {
	return codegenCtx{
		ast:         root,
		sourceText:  sourceText,
		serviceText: strings.Builder{},
		mappings:    []mapping.Mapping{},
		diagnostics: []*ast.Diagnostic{},
		setupConsts: make(map[string]bool),
	}
}

func (c *codegenCtx) reportDiagnostic(loc core.TextRange, message *diagnostics.Message, args ...any) {
	c.diagnostics = append(c.diagnostics, ast.NewDiagnostic(nil, loc, message, args...))
}

func (c *codegenCtx) mapText(from, to int) {
	serviceOffset := c.serviceText.Len()
	c.serviceText.WriteString(c.sourceText[from:to])
	c.mappings = append(c.mappings, mapping.Mapping{
		SourceOffset:  from,
		ServiceOffset: serviceOffset,
		Length:        to - from,
	})
}
