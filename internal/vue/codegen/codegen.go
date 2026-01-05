package vue_codegen

import (
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

// Iterable requires es2015.iterable
const GlobalTypes = `/// <reference lib="es2015" />
export {}

declare global {
	type __VLS_PickNotAny<A, B> = 0 extends 1 & A ? B : A

	// Unwrap Ref/ComputedRef types for template auto-unwrapping
	// Refs have a .value property with UnwrapRef brand
	// This checks if T has a 'value' property and extracts its type
	type __VLS_UnwrapRef<T> = T extends { readonly value: infer V } ? V : T

	function __VLS_vFor<T>(source: T): T extends number
		? [number, number]
		: T extends string
			? [string, number]
			: T extends any[]
				? [T[number], number]
				: T extends Iterable<infer V>
					? [V, number]
					: [T[keyof T], ` + "`${keyof T}`" + `, number]

	function __VLS_vSlot<S, D extends S>(slot: S, decl?: D): D extends (...args: infer P) => any ? P : any[]
}
`

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
		generateScript(&c, scriptSetupEl, scriptEl)
		newMappingsStart := len(ctx.mappings)
		ctx.mappings = append(ctx.mappings, c.mappings...)
		for i := newMappingsStart; i < len(ctx.mappings); i++ {
			ctx.mappings[i].ServiceOffset += ctx.serviceText.Len()
		}
		ctx.serviceText.Write([]byte(c.serviceText.String()))
		ctx.diagnostics = append(ctx.diagnostics, c.diagnostics...)
	}

	{
		c := newCodegenCtx(root, sourceText)
		generateTemplate(&c, templateEl)
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
}

func newCodegenCtx(root *vue_ast.RootNode, sourceText string) codegenCtx {
	return codegenCtx{
		ast:         root,
		sourceText:  sourceText,
		serviceText: strings.Builder{},
		mappings:    []mapping.Mapping{},
		diagnostics: []*ast.Diagnostic{},
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
