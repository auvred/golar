package vue_codegen

import (
	_ "embed"
	"regexp"
	"strconv"
	"strings"

	"github.com/auvred/golar/internal/mapping"
	"github.com/auvred/golar/internal/vue/ast"
	"github.com/auvred/golar/internal/vue/diagnostics"
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/core"
	"github.com/microsoft/typescript-go/shim/diagnostics"
)

// TODO: relative to cwd or executable location, so vue import works
// const GlobalTypesPath = utils.GolarVirtualScheme + "vue-global-types.d.ts"
const TemplateHelpersPath = "/" + "template-helpers.d.ts"
const PropsFallbackPath = "/" + "props-fallback.d.ts"
const globalTypesReference = `/// <reference types="` + TemplateHelpersPath + `" />
/// <reference types="` + PropsFallbackPath + `" />
`

//go:embed types/template-helpers.d.ts
var TemplateHelpers string

//go:embed types/props-fallback.d.ts
var PropsFallback string

func Codegen(sourceText string, root *vue_ast.RootNode, options VueOptions) (string, []mapping.Mapping, []mapping.IgnoreDirectiveMapping, []mapping.ExpectErrorDirectiveMapping, []*ast.Diagnostic) {
	ctx := newCodegenCtx(root, sourceText, options)
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

		if el.Tag == "style" {
			isScoped := false
			for _, prop := range el.Props {
				if prop.Kind == vue_ast.KindAttribute && prop.AsAttribute().Name == "scoped" {
					isScoped = true
					break
				}
			}
			if isScoped {
				ctx.hasScopedStyle = true
				// Extract CSS class selectors from scoped style content
				if len(el.Children) > 0 && el.Children[0].Kind == vue_ast.KindText {
					cssContent := el.Children[0].AsText().Content
					ctx.cssClasses = extractCSSClasses(cssContent)
				}
			}
		}
	}

	// Volar emits script content inline without space padding.
	// Line correspondence is handled through source mappings, not space padding.
	generateScript(&ctx, scriptSetupEl, scriptEl, templateEl)

	return ctx.serviceText.String(), ctx.mappings, ctx.ignoreDirectives, ctx.expectErrorDirectives, ctx.diagnostics
}

type codegenCtx struct {
	ast                     *vue_ast.RootNode
	sourceText              string
	serviceText             strings.Builder
	mappings                []mapping.Mapping
	ignoreDirectives        []mapping.IgnoreDirectiveMapping
	expectErrorDirectives   []mapping.ExpectErrorDirectiveMapping
	diagnostics             []*ast.Diagnostic
	internalVariableCounter int
	options                 VueOptions
	templateHasSlots        bool
	hasScopedStyle          bool
	cssClasses              []string // CSS class selectors from <style scoped>
	// usedTemplateVars tracks variables referenced in the template for the
	// // @ts-ignore [var1,var2,...]; block that Volar emits after template codegen.
	usedTemplateVars []string
	// allAccessedVars tracks ALL vars ever accessed via __VLS_ctx in the template
	// (never cleared by scope drains), used for filtering __VLS_SetupExposed.
	allAccessedVars []string
	// scopedClasses tracks class names used in template elements for __VLS_StyleScopedClasses type.
	// Ordered by first occurrence, deduplicated.
	scopedClasses    []string
	scopedClassesSet map[string]bool
	// templateRefs tracks ref attribute values to element tags for __VLS_TemplateRefs type.
	templateRefs []templateRefInfo
}

type templateRefInfo struct {
	name    string // ref attribute value
	elemTag string // element tag name
}

type VueVersion int
type VueOptions struct {
	// major * 1_000_000 + minor * 1_000 + patch
	Version VueVersion
}

func NewVueVersionFromSemver(major, minor, patch int) VueVersion {
	return VueVersion(major*1_000_000 + minor*1_000 + patch)
}

// atLeast returns true if the version is unset (0, meaning "assume latest")
// or at least the given version.
func (v VueVersion) atLeast(major, minor, patch int) bool {
	return v == 0 || v >= NewVueVersionFromSemver(major, minor, patch)
}

// https://github.com/vuejs/core/pull/10801
func (v VueVersion) supportsTypeProps() bool {
	return v.atLeast(3, 5, 0)
}
func (v VueVersion) supportsTypeEmits() bool {
	return v.supportsTypeProps()
}

func (v VueVersion) supportsDefineSlots() bool {
	return v.atLeast(3, 3, 0)
}

func (v VueVersion) supportsDefineModel() bool {
	return v.atLeast(3, 4, 0)
}

// https://github.com/vuejs/core/pull/11699
func (v VueVersion) modelRefHasGetterAndSetter() bool {
	return v.atLeast(3, 5, 0)
}

func (v VueVersion) hasPublicPropsType() bool {
	return v.atLeast(3, 4, 0)
}

func (v VueVersion) hasJsxRuntimeTypes() bool {
	return v.atLeast(3, 3, 0)
}

func newCodegenCtx(root *vue_ast.RootNode, sourceText string, options VueOptions) codegenCtx {
	return codegenCtx{
		ast:         root,
		sourceText:  sourceText,
		serviceText: strings.Builder{},
		mappings:    []mapping.Mapping{},
		diagnostics: []*ast.Diagnostic{},
		options:     options,
	}
}

// templateOutput holds the buffered output from template codegen.
// This allows generating the template first (to collect used vars),
// then emitting the script boilerplate with filtered bindings,
// then appending the template output.
type templateOutput struct {
	text              string
	mappings          []mapping.Mapping
	ignoreDirectives  []mapping.IgnoreDirectiveMapping
	expectErrorDirs   []mapping.ExpectErrorDirectiveMapping
	diagnostics       []*ast.Diagnostic
	usedTemplateVars  []string
	allAccessedVars   []string
	templateHasSlots  bool
	internalVarCount  int
	scopedClasses     []string
	templateRefs      []templateRefInfo
}

// generateTemplateBuffered generates template codegen into a separate buffer.
func generateTemplateBuffered(base *codegenCtx, el *vue_ast.ElementNode) templateOutput {
	// Create a temporary codegenCtx with its own serviceText
	tmpCtx := codegenCtx{
		ast:                     base.ast,
		sourceText:              base.sourceText,
		serviceText:             strings.Builder{},
		mappings:                []mapping.Mapping{},
		diagnostics:             []*ast.Diagnostic{},
		internalVariableCounter: base.internalVariableCounter,
		options:                 base.options,
		hasScopedStyle:          base.hasScopedStyle,
	}
	generateTemplate(&tmpCtx, el)
	return templateOutput{
		text:              tmpCtx.serviceText.String(),
		mappings:          tmpCtx.mappings,
		ignoreDirectives:  tmpCtx.ignoreDirectives,
		expectErrorDirs:   tmpCtx.expectErrorDirectives,
		diagnostics:       tmpCtx.diagnostics,
		usedTemplateVars:  tmpCtx.usedTemplateVars,
		allAccessedVars:   tmpCtx.allAccessedVars,
		templateHasSlots:  tmpCtx.templateHasSlots,
		internalVarCount:  tmpCtx.internalVariableCounter,
		scopedClasses:     tmpCtx.scopedClasses,
		templateRefs:      tmpCtx.templateRefs,
	}
}

// mergeTemplateOutput appends the buffered template output to the main context,
// shifting all service offsets by the current position.
func (c *codegenCtx) mergeTemplateOutput(t templateOutput) {
	offset := uint32(c.serviceText.Len())
	c.serviceText.WriteString(t.text)
	for _, m := range t.mappings {
		c.mappings = append(c.mappings, mapping.Mapping{
			SourceOffset:  m.SourceOffset,
			ServiceOffset: m.ServiceOffset + offset,
			SourceLength:  m.SourceLength,
		})
	}
	for _, d := range t.ignoreDirectives {
		c.ignoreDirectives = append(c.ignoreDirectives, mapping.IgnoreDirectiveMapping{
			ServiceOffset: d.ServiceOffset + offset,
			ServiceLength: d.ServiceLength,
		})
	}
	for _, d := range t.expectErrorDirs {
		c.expectErrorDirectives = append(c.expectErrorDirectives, mapping.ExpectErrorDirectiveMapping{
			SourceOffset:  d.SourceOffset,
			ServiceOffset: d.ServiceOffset + offset,
			SourceLength:  d.SourceLength,
			ServiceLength: d.ServiceLength,
		})
	}
	c.diagnostics = append(c.diagnostics, t.diagnostics...)
	c.templateHasSlots = t.templateHasSlots
	c.usedTemplateVars = t.usedTemplateVars
	c.allAccessedVars = t.allAccessedVars
	c.internalVariableCounter = t.internalVarCount
}

func (c *codegenCtx) reportDiagnostic(loc core.TextRange, message *diagnostics.Message, args ...any) {
	c.diagnostics = append(c.diagnostics, ast.NewDiagnostic(nil, loc, message, args...))
}

func (c *codegenCtx) mapText(from, to int) {
	serviceOffset := c.serviceText.Len()
	c.serviceText.WriteString(c.sourceText[from:to])
	c.mappings = append(c.mappings, mapping.Mapping{
		SourceOffset:  uint32(from),
		ServiceOffset: uint32(serviceOffset),
		SourceLength:  uint32(to - from),
	})
}

func (c *codegenCtx) mapRange(sourceStart, sourceEnd, serviceStart, serviceEnd int) {
	c.mappings = append(c.mappings, mapping.Mapping{
		SourceOffset:  uint32(sourceStart),
		ServiceOffset: uint32(serviceStart),
		SourceLength:  uint32(0),
	}, mapping.Mapping{
		SourceOffset:  uint32(sourceEnd),
		ServiceOffset: uint32(serviceEnd),
		SourceLength:  uint32(0),
	})
}

func (c *codegenCtx) mapIgnoreDirective(serviceStart, serviceEnd int) {
	c.ignoreDirectives = append(c.ignoreDirectives, mapping.IgnoreDirectiveMapping{
		ServiceOffset: uint32(serviceStart),
		ServiceLength: uint32(serviceEnd - serviceStart),
	})
}

func (c *codegenCtx) mapExpectErrorDirective(sourceStart, sourceEnd, serviceStart, serviceEnd int) {
	c.expectErrorDirectives = append(c.expectErrorDirectives, mapping.ExpectErrorDirectiveMapping{
		SourceOffset:  uint32(sourceStart),
		ServiceOffset: uint32(serviceStart),
		SourceLength:  uint32(sourceEnd - sourceStart),
		ServiceLength: uint32(serviceEnd - serviceStart),
	})
}

func (c *codegenCtx) newInternalVariable() string {
	n := c.internalVariableCounter
	c.internalVariableCounter++
	return "__VLS_" + strconv.Itoa(n)
}

// cssClassSelectorRe matches CSS class selectors like .foo-bar
var cssClassSelectorRe = regexp.MustCompile(`\.([\w-]+)`)

// extractCSSClasses extracts class names from CSS content, preserving order and deduplicating.
func extractCSSClasses(css string) []string {
	matches := cssClassSelectorRe.FindAllStringSubmatch(css, -1)
	var classes []string
	seen := map[string]bool{}
	for _, m := range matches {
		name := m[1]
		if !seen[name] {
			seen[name] = true
			classes = append(classes, name)
		}
	}
	return classes
}
