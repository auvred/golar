package vue_codegen

import (
	"github.com/auvred/golar/internal/collections"
	"github.com/auvred/golar/internal/utils"
	"github.com/auvred/golar/internal/vue/ast"
	"github.com/auvred/golar/internal/vue/diagnostics"
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/core"
)

// TODO: <script src="">

// defineModelInfo holds parsed metadata for a single defineModel() call.
type defineModelInfo struct {
	camelizedName            string // "modelValue" for default, or camelized from explicit name
	camelizedModifierName    string // "model" for default, or same as camelizedName
	typeText                 string // source text of the type argument, e.g. "number", or "" for any
	modifierTypeText         string // source text of the 2nd type argument, e.g. "'trim' | 'lazy'", or ""
	required                 bool   // options.required === true
	hasDefault               bool   // options.default present
}

type scriptCodegenCtx struct {
	*codegenCtx
	scriptSetupEl *vue_ast.ElementNode
	scriptEl      *vue_ast.ElementNode
	lastMappedPos int

	bindingNames []string

	seenDefineModels collections.Set[string]
	defineModels     []defineModelInfo
}

func generateScript(base *codegenCtx, scriptSetupEl *vue_ast.ElementNode, scriptEl *vue_ast.ElementNode, templateEl *vue_ast.ElementNode) {
	c := scriptCodegenCtx{
		codegenCtx:    base,
		scriptSetupEl: scriptSetupEl,
		scriptEl:      scriptEl,
	}

	// TODO: without "ts" lang
	// TODO: tsx

	var selfType string
	if c.scriptEl != nil && len(c.scriptEl.Children) == 1 {
		innerStart := c.scriptEl.InnerLoc.Pos()
		text := c.scriptEl.Children[0].AsText()

		c.lastMappedPos = text.Loc.Pos()

		for _, statement := range c.scriptEl.Ast.Statements.Nodes {
			if c.scriptSetupEl != nil {
				c.collectBindingRanges(innerStart, statement)
			}
			if !ast.IsExportAssignment(statement) {
				continue
			}
			// TODO: report export equals? (export = ...)

			export := statement.AsExportAssignment()
			if c.scriptSetupEl == nil {
				c.mapText(c.lastMappedPos, innerStart+export.Expression.Pos())
				c.serviceText.WriteString(" {} as unknown as typeof __VLS_Self\n")
				c.serviceText.WriteString("const __VLS_Self = ")
				selfType = "__VLS_Self"
				expr := export.Expression
				for ast.IsParenthesizedExpression(expr) || ast.KindAsExpression == expr.Kind {
					expr = expr.Expression()
				}
				if ast.IsObjectLiteralExpression(expr) {
					exportLoc := utils.TrimNodeTextRange(c.scriptEl.Ast, export.AsNode())
					c.mapRange(innerStart+exportLoc.Pos(), innerStart+export.Expression.Pos(), c.serviceText.Len(), c.serviceText.Len()+len("(await import('vue')).defineComponent"))
					c.serviceText.WriteString("(await import('vue')).defineComponent(")
				}
				c.mapText(innerStart+export.Expression.Pos(), innerStart+export.Expression.End())
				c.lastMappedPos = innerStart + export.Expression.End()
				if ast.IsObjectLiteralExpression(expr) {
					c.serviceText.WriteString(")")
				}
			} else {
				c.mapText(c.lastMappedPos, innerStart+export.Pos())
				c.serviceText.WriteString(";(")
				c.mapText(innerStart+export.Expression.Pos(), innerStart+export.Expression.End())
				c.serviceText.WriteString(")\n")
				c.lastMappedPos = innerStart + export.Expression.End()
			}

			break
		}

		c.mapText(c.lastMappedPos, text.Loc.End())
		c.serviceText.WriteString("\n\n")

		// TODO: options wrapper - wrap export default |defineComponent(|{}|)|
	}

	if c.scriptSetupEl != nil {
		hasGeneric := false
		for _, prop := range c.scriptSetupEl.Props {
			if prop.Kind != vue_ast.KindAttribute {
				continue
			}

			attr := prop.AsAttribute()
			switch attr.Name {
			case "generic":
				if attr.Value == nil {
					break
				}
				hasGeneric = true
				_ = hasGeneric // TODO: generic support
			}
		}

		// Emit script setup content inline (no IIFE wrapper)
		innerStart := c.scriptSetupEl.InnerLoc.Pos()

		var hasText bool
		if len(c.scriptSetupEl.Children) == 1 {
			text := c.scriptSetupEl.Children[0].AsText()
			c.lastMappedPos = text.Loc.Pos()
			hasText = true
		}

		var (
			propsVariableName string
			emitsVariableName string
			slotsVariableName string
			hasExpose         bool
			// Track the original defineProps/defineEmits type argument names for Volar-style extraction
			propsTypeName string
			emitsTypeName string
		)

		// TODO: report nested compiler macros (vue compiler errors on them)
		// TODO: report incorrect compiler macros arguments
		// TODO: $emits, $props, emitstoprops

		importRanges := []core.TextRange{}
		var statements []*ast.Node
		if c.scriptSetupEl.Ast != nil {
			statements = c.scriptSetupEl.Ast.Statements.Nodes
		}
		for _, statement := range statements {
			c.collectBindingRanges(innerStart, statement)
			switch statement.Kind {
			case ast.KindVariableStatement:
				for _, d := range statement.AsVariableStatement().DeclarationList.AsVariableDeclarationList().Declarations.Nodes {
					decl := d.AsVariableDeclaration()
					name := decl.Name()

					nameIsIdentifier := ast.IsIdentifier(name)
					if decl.Initializer == nil || !ast.IsCallExpression(decl.Initializer) {
						break
					}

					call := decl.Initializer.AsCallExpression()
					callee := call.Expression
					if !ast.IsIdentifier(callee) {
						break
					}
					calleeName := callee.Text()
					switch calleeName {
					case "withDefaults":
						// TODO: report props destructuring?
						if !nameIsIdentifier {
							break
						}
						// TODO: align reporting with vue compiler
						if len(call.Arguments.Nodes) != 2 || !ast.IsCallExpression(call.Arguments.Nodes[0]) || !ast.IsIdentifier(call.Arguments.Nodes[0].Expression()) || call.Arguments.Nodes[0].Expression().Text() != "defineProps" {
							break
						}
						call = call.Arguments.Nodes[0].AsCallExpression()
						callee = call.Expression
						calleeName = callee.Text()
						fallthrough
					case "defineProps":
						// TODO: report props destructuring?
						if !nameIsIdentifier {
							break
						}
						if propsVariableName != "" {
							calleeLoc := utils.TrimNodeTextRange(c.scriptSetupEl.Ast, callee)
							c.reportDiagnostic(utils.MoveTextRange(calleeLoc, innerStart), vue_diagnostics.Duplicate_X_0_call, "defineProps")
							break
						}
						propsVariableName = name.Text()
						// Check if defineProps has a type argument — extract it Volar-style
						if call.TypeArguments != nil && len(call.TypeArguments.Nodes) == 1 {
							typeArg := call.TypeArguments.Nodes[0]
							if ast.IsTypeLiteralNode(typeArg) {
								propsTypeName = "__VLS_Props"
							} else if ast.IsTypeReferenceNode(typeArg) {
								propsTypeName = typeArg.AsTypeReferenceNode().TypeName.Text()
							}
						}
					case "defineEmits":
						// TODO: can there be destructuring
						if !nameIsIdentifier {
							break
						}
						if emitsVariableName != "" {
							calleeLoc := utils.TrimNodeTextRange(c.scriptSetupEl.Ast, callee)
							c.reportDiagnostic(utils.MoveTextRange(calleeLoc, innerStart), vue_diagnostics.Duplicate_X_0_call, "defineEmits")
							break
						}
						emitsVariableName = name.Text()
						// Check if defineEmits has a type argument — extract it Volar-style
						if call.TypeArguments != nil && len(call.TypeArguments.Nodes) == 1 {
							typeArg := call.TypeArguments.Nodes[0]
							if ast.IsTypeLiteralNode(typeArg) {
								emitsTypeName = "__VLS_Emit"
							} else if ast.IsTypeReferenceNode(typeArg) {
								emitsTypeName = typeArg.AsTypeReferenceNode().TypeName.Text()
							}
						}
					case "defineSlots":
						// TODO: can there be destructuring
						if !nameIsIdentifier {
							break
						}
						if !c.options.Version.supportsDefineSlots() {
							break
						}
						if slotsVariableName != "" {
							calleeLoc := utils.TrimNodeTextRange(c.scriptSetupEl.Ast, callee)
							c.reportDiagnostic(utils.MoveTextRange(calleeLoc, innerStart), vue_diagnostics.Duplicate_X_0_call, "defineSlots")
							break
						}
						slotsVariableName = name.Text()
					case "defineModel":
						if !c.options.Version.supportsDefineModel() {
							break
						}
						callLoc := utils.TrimNodeTextRange(c.scriptSetupEl.Ast, call.AsNode())
						c.mapText(c.lastMappedPos, innerStart+callLoc.Pos())
						c.lastMappedPos = innerStart + callLoc.Pos()
						c.processDefineModel(innerStart, call, callLoc, name.Text())
					}
				}
			case ast.KindExpressionStatement:
				expr := statement.AsExpressionStatement().Expression
				if !ast.IsCallExpression(expr) {
					break
				}
				call := expr.AsCallExpression()
				callee := call.Expression
				if !ast.IsIdentifier(callee) {
					break
				}
				calleeName := callee.Text()
				switch calleeName {
				case "withDefaults", "defineProps":
					if propsVariableName != "" {
						calleeLoc := utils.TrimNodeTextRange(c.scriptSetupEl.Ast, callee)
						c.reportDiagnostic(utils.MoveTextRange(calleeLoc, innerStart), vue_diagnostics.Duplicate_X_0_call, "defineProps")
						break
					}
					propsVariableName = "__VLS_Props"
					c.mapText(c.lastMappedPos, innerStart+statement.Pos())
					c.serviceText.WriteString("\nconst __VLS_Props = ")
					c.mapText(innerStart+statement.Pos(), innerStart+statement.End())
					c.lastMappedPos = innerStart + statement.End()
				case "defineEmits":
					if emitsVariableName != "" {
						calleeLoc := utils.TrimNodeTextRange(c.scriptSetupEl.Ast, callee)
						c.reportDiagnostic(utils.MoveTextRange(calleeLoc, innerStart), vue_diagnostics.Duplicate_X_0_call, "defineEmits")
						break
					}
					emitsVariableName = "__VLS_Emits"
					c.mapText(c.lastMappedPos, innerStart+statement.Pos())
					c.serviceText.WriteString("\nconst __VLS_Emits = ")
					c.mapText(innerStart+statement.Pos(), innerStart+statement.End())
					c.lastMappedPos = innerStart + statement.End()
				case "defineSlots":
					if !c.options.Version.supportsDefineSlots() {
						break
					}
					if slotsVariableName != "" {
						calleeLoc := utils.TrimNodeTextRange(c.scriptSetupEl.Ast, callee)
						c.reportDiagnostic(utils.MoveTextRange(calleeLoc, innerStart), vue_diagnostics.Duplicate_X_0_call, "defineSlots")
						break
					}
					slotsVariableName = "__VLS_Slots"
					c.mapText(c.lastMappedPos, innerStart+statement.Pos())
					c.serviceText.WriteString("\nconst __VLS_Slots = ")
					c.mapText(innerStart+statement.Pos(), innerStart+statement.End())
					c.lastMappedPos = innerStart + statement.End()
				case "defineExpose":
					if hasExpose {
						calleeLoc := utils.TrimNodeTextRange(c.scriptSetupEl.Ast, callee)
						c.reportDiagnostic(utils.MoveTextRange(calleeLoc, innerStart), vue_diagnostics.Duplicate_X_0_call, "defineExpose")
						break
					}
					hasExpose = true
					// Volar pattern: extract argument into variable, then call defineExpose with it
					// const __VLS_exposed = <argument>;
					// defineExpose(__VLS_exposed)
					c.mapText(c.lastMappedPos, innerStart+statement.Pos())
					c.lastMappedPos = innerStart + statement.End()
					c.serviceText.WriteString("const __VLS_exposed = ")
					if len(call.Arguments.Nodes) > 0 {
						arg := call.Arguments.Nodes[0]
						argStart := innerStart + arg.Pos()
						argEnd := innerStart + arg.End()
						c.serviceText.WriteString(c.sourceText[argStart:argEnd])
					}
					c.serviceText.WriteString(";\n")
					c.serviceText.WriteString("defineExpose(__VLS_exposed)\n")
				case "defineModel":
					callLoc := utils.TrimNodeTextRange(c.scriptSetupEl.Ast, call.AsNode())
					c.mapText(c.lastMappedPos, innerStart+callLoc.Pos())
					c.lastMappedPos = innerStart + callLoc.Pos()
					c.processDefineModel(innerStart, call, callLoc, "")
				}
			case ast.KindImportDeclaration:
				importRanges = append(importRanges, utils.MoveTextRange(statement.Loc, innerStart))
			}
		}

		// For Volar-style type extraction: transform defineProps<{...}>() into type + call
		c.emitScriptSetupContent(innerStart, hasText, importRanges, propsTypeName, emitsTypeName)

		// Generate template first (buffered) to collect used vars for binding filtering
		tmplOutput := generateTemplateBuffered(c.codegenCtx, templateEl)

		// Build set of ALL template-accessed vars for filtering bindings
		// (allAccessedVars includes vars from drained inner scopes too)
		_ = tmplOutput // used below for merging

		// Macros declaration — conditionally include macros based on Vue version
		c.serviceText.WriteString("// @ts-ignore\ndeclare const { defineProps, ")
		if c.options.Version.supportsDefineSlots() {
			c.serviceText.WriteString("defineSlots, ")
		}
		c.serviceText.WriteString("defineEmits, defineExpose, ")
		if c.options.Version.supportsDefineModel() {
			c.serviceText.WriteString("defineModel, ")
		}
		c.serviceText.WriteString("defineOptions, withDefaults, }: typeof import('vue');\n")

		// Emit consolidated model types (Volar-style: __VLS_ModelProps, __VLS_ModelEmit, __VLS_modelEmit)
		c.emitModelTypes()

		// PublicProps type (before SetupExposed, to match Volar order for medium-complex)
		hasPublicProps := false
		if propsVariableName != "" || len(c.defineModels) > 0 {
			hasPublicProps = true
			c.serviceText.WriteString("type __VLS_PublicProps = ")
			parts := []string{}
			if propsVariableName != "" {
				if propsTypeName != "" {
					parts = append(parts, propsTypeName)
				} else {
					parts = append(parts, "typeof "+propsVariableName)
				}
			}
			if len(c.defineModels) > 0 {
				parts = append(parts, "__VLS_ModelProps")
			}
			for i, p := range parts {
				if i > 0 {
					c.serviceText.WriteString(" & ")
				}
				c.serviceText.WriteString(p)
			}
			c.serviceText.WriteString(";\n")
		}

		// SetupExposed type — only include bindings that are actually used in the template
		// (Volar computes this as the intersection of script bindings and template accesses)
		c.serviceText.WriteString("type __VLS_SetupExposed = import('vue').ShallowUnwrapRef<{\n")
		// Emit bindings in template access order (first-seen order from allAccessedVars)
		var emittedBindings collections.Set[string]
		var bindingSet collections.Set[string]
		for _, name := range c.bindingNames {
			bindingSet.Add(name)
		}
		for _, v := range tmplOutput.allAccessedVars {
			if emittedBindings.Has(v) || !bindingSet.Has(v) {
				continue
			}
			emittedBindings.Add(v)
			c.serviceText.WriteString(v)
			c.serviceText.WriteString(": typeof ")
			c.serviceText.WriteString(v)
			c.serviceText.WriteString(";\n")
		}
		c.serviceText.WriteString("}>;\n")

		// EmitProps type — define when there are emits (defineEmits or defineModel).
		// Volar uses "typeof __VLS_modelEmit" for model emits and "typeof emitsVar" for defineEmits.
		hasPublicEmits := false
		if emitsVariableName != "" || len(c.defineModels) > 0 {
			hasPublicEmits = true
			// Build emit type parts
			emitTypeParts := []string{}
			if emitsVariableName != "" {
				emitTypeParts = append(emitTypeParts, "typeof "+emitsVariableName)
			}
			if len(c.defineModels) > 0 {
				emitTypeParts = append(emitTypeParts, "typeof __VLS_modelEmit")
			}
			// Only need __VLS_PublicEmits alias when there are both defineEmits and defineModel
			if len(emitTypeParts) > 1 {
				c.serviceText.WriteString("type __VLS_PublicEmits = ")
				for i, p := range emitTypeParts {
					if i > 0 {
						c.serviceText.WriteString(" & ")
					}
					c.serviceText.WriteString(p)
				}
				c.serviceText.WriteString(";\n")
			}
			c.serviceText.WriteString("type __VLS_EmitProps = __VLS_EmitsToProps<__VLS_NormalizeEmits<")
			if len(emitTypeParts) > 1 {
				c.serviceText.WriteString("__VLS_PublicEmits")
			} else {
				c.serviceText.WriteString(emitTypeParts[0])
			}
			c.serviceText.WriteString(">>;\n")
		}

		// Context object
		c.serviceText.WriteString("const __VLS_ctx = {\n")
		if selfType != "" {
			c.serviceText.WriteString("...{} as InstanceType<__VLS_PickNotAny<typeof ")
			c.serviceText.WriteString(selfType)
			c.serviceText.WriteString(" extends new () => {} ? typeof ")
			c.serviceText.WriteString(selfType)
			c.serviceText.WriteString(" : new () => {}, new () => {}>>,\n")
		} else {
			c.serviceText.WriteString("...{} as import('vue').ComponentPublicInstance,\n")
		}
		if hasPublicEmits {
			// Volar uses "typeof emitsVar" for defineEmits, "typeof __VLS_modelEmit" for defineModel
			c.serviceText.WriteString("...{} as { $emit: ")
			emitRef := c.emitTypeRef(emitsVariableName)
			c.serviceText.WriteString(emitRef)
			c.serviceText.WriteString(" },\n")
		}
		if hasPublicProps {
			// For $props and context spread, Volar uses "typeof propsVar" when available,
			// + __VLS_ModelProps when there are defineModel calls.
			// __VLS_PublicProps is used only in the export section, not here.
			writeCtxPropsType := func() {
				if propsVariableName != "" {
					c.serviceText.WriteString("typeof ")
					c.serviceText.WriteString(propsVariableName)
				}
				if len(c.defineModels) > 0 {
					if propsVariableName != "" {
						c.serviceText.WriteString(" & ")
					}
					c.serviceText.WriteString("__VLS_ModelProps")
				}
				if hasPublicEmits {
					c.serviceText.WriteString(" & __VLS_EmitProps")
				}
			}
			c.serviceText.WriteString("...{} as { $props: ")
			writeCtxPropsType()
			c.serviceText.WriteString(" },\n")
			c.serviceText.WriteString("...{} as ")
			writeCtxPropsType()
			c.serviceText.WriteString(",\n")
		}
		c.serviceText.WriteString("...{} as __VLS_SetupExposed,\n")
		c.serviceText.WriteString("};\n")

		// Component and directive type declarations
		c.serviceText.WriteString("type __VLS_LocalComponents = __VLS_SetupExposed;\n")
		c.serviceText.WriteString("type __VLS_GlobalComponents = import('vue').GlobalComponents;\n")
		c.serviceText.WriteString("let __VLS_components!: __VLS_LocalComponents & __VLS_GlobalComponents;\n")
		if c.options.Version.hasJsxRuntimeTypes() {
			c.serviceText.WriteString("let __VLS_intrinsics!: import('vue/jsx-runtime').JSX.IntrinsicElements;\n")
		} else {
			c.serviceText.WriteString("let __VLS_intrinsics!: globalThis.JSX.IntrinsicElements;\n")
		}
		c.serviceText.WriteString("type __VLS_LocalDirectives = __VLS_SetupExposed;\n")
		c.serviceText.WriteString("let __VLS_directives!: __VLS_LocalDirectives & import('vue').GlobalDirectives;\n")

		// __VLS_StyleScopedClasses type (only when <style scoped> present)
		if c.hasScopedStyle && len(c.cssClasses) > 0 {
			c.serviceText.WriteString("type __VLS_StyleScopedClasses = {}\n")
			for _, cls := range c.cssClasses {
				c.serviceText.WriteString(" & { '")
				c.serviceText.WriteString(cls)
				c.serviceText.WriteString("': boolean };\n")
			}
		}

		// Merge buffered template output
		c.mergeTemplateOutput(tmplOutput)

		// Used template vars (top-level remaining after inner scope drains)
		c.serviceText.WriteString("// @ts-ignore\n[")
		for _, v := range c.usedTemplateVars {
			c.serviceText.WriteString(v)
			c.serviceText.WriteString(",")
		}
		c.serviceText.WriteString("];\n")

		// Export
		if hasGeneric {
			// TODO: generic export pattern
			c.serviceText.WriteString("const __VLS_export = (await import('vue')).defineComponent({\n});\nexport default {} as typeof __VLS_export;\n")
		} else {
			hasSlots := slotsVariableName != "" || c.templateHasSlots
			hasDefineComponentOptions := hasPublicProps || hasPublicEmits || hasExpose

			if !hasDefineComponentOptions && !hasSlots {
				// Simple case: no props/emits/expose/slots
				c.serviceText.WriteString("const __VLS_export = (await import('vue')).defineComponent({\n});\nexport default {} as typeof __VLS_export;\n\n")
			} else if hasSlots {
				// With slots: use __VLS_base intermediate
				c.serviceText.WriteString("const __VLS_base = (await import('vue')).defineComponent({\n")
				if hasPublicEmits {
					emitType := c.emitTypeForExport(emitsVariableName, emitsTypeName)
					if c.options.Version.supportsTypeEmits() {
						c.serviceText.WriteString("__typeEmits: {} as ")
						c.serviceText.WriteString(emitType)
						c.serviceText.WriteString(",\n")
					} else {
						c.serviceText.WriteString("emits: {} as unknown as __VLS_NormalizeEmits<")
						c.serviceText.WriteString(emitType)
						c.serviceText.WriteString(">,\n")
					}
				}
				if hasPublicProps {
					if c.options.Version.supportsTypeProps() {
						c.serviceText.WriteString("__typeProps: {} as __VLS_PublicProps,\n")
					} else {
						c.serviceText.WriteString("props: {} as unknown as __VLS_TypePropsToOption<__VLS_PublicProps>,\n")
					}
				}
				if hasExpose {
					c.serviceText.WriteString("setup: () => (__VLS_exposed),\n")
				}
				c.serviceText.WriteString("});\n")
				c.serviceText.WriteString("const __VLS_export = {} as __VLS_WithSlots<typeof __VLS_base, __VLS_Slots>;\n")
				c.serviceText.WriteString("export default {} as typeof __VLS_export;\n")
				c.serviceText.WriteString("type __VLS_WithSlots<T, S> = T & {\n\tnew(): {\n\t\t$slots: S;\n\t}\n};\n\n")
			} else {
				// Without slots: directly assign to __VLS_export
				c.serviceText.WriteString("const __VLS_export = (await import('vue')).defineComponent({\n")
				if hasPublicEmits {
					emitType := c.emitTypeForExport(emitsVariableName, emitsTypeName)
					if c.options.Version.supportsTypeEmits() {
						c.serviceText.WriteString("__typeEmits: {} as ")
						c.serviceText.WriteString(emitType)
						c.serviceText.WriteString(",\n")
					} else {
						c.serviceText.WriteString("emits: {} as unknown as __VLS_NormalizeEmits<")
						c.serviceText.WriteString(emitType)
						c.serviceText.WriteString(">,\n")
					}
				}
				if hasPublicProps {
					if c.options.Version.supportsTypeProps() {
						c.serviceText.WriteString("__typeProps: {} as __VLS_PublicProps,\n")
					} else {
						c.serviceText.WriteString("props: {} as unknown as __VLS_TypePropsToOption<__VLS_PublicProps>,\n")
					}
				}
				if hasExpose {
					c.serviceText.WriteString("setup: () => (__VLS_exposed),\n")
				}
				c.serviceText.WriteString("});\n")
				c.serviceText.WriteString("export default {} as typeof __VLS_export;\n")
			}
		}

	} else {
		c.serviceText.WriteString("type __VLS_SetupExposed = {};\n")
		c.serviceText.WriteString("const __VLS_ctx = {\n")
		if selfType != "" {
			c.serviceText.WriteString("...{} as InstanceType<__VLS_PickNotAny<typeof ")
			c.serviceText.WriteString(selfType)
			c.serviceText.WriteString(" extends new () => {} ? typeof ")
			c.serviceText.WriteString(selfType)
			c.serviceText.WriteString(" : new () => {}, new () => {}>>,\n")
		} else {
			c.serviceText.WriteString("...{} as import('vue').ComponentPublicInstance,\n")
		}
		c.serviceText.WriteString("};\n")
		generateTemplate(c.codegenCtx, templateEl)
	}
}

// emitScriptSetupContent emits the script setup content inline, handling import hoisting
// and Volar-style defineProps/defineEmits type extraction.
func (c *scriptCodegenCtx) emitScriptSetupContent(innerStart int, hasText bool, importRanges []core.TextRange, propsTypeName, emitsTypeName string) {
	if !hasText {
		return
	}

	text := c.scriptSetupEl.Children[0].AsText()
	sourceContent := c.sourceText[text.Loc.Pos():text.Loc.End()]

	// For Volar-style type extraction, we need to transform:
	// const props = defineProps<{...}>() → type __VLS_Props = {...};\nconst props = defineProps<__VLS_Props>()
	// const emit = defineEmits<{...}>() → type __VLS_Emit = {...};\nconst emit = defineEmits<__VLS_Emit>()
	if propsTypeName != "" || emitsTypeName != "" {
		c.emitScriptSetupContentWithTypeExtraction(innerStart, sourceContent, text, importRanges, propsTypeName, emitsTypeName)
	} else {
		c.emitScriptSetupContentDirect(innerStart, sourceContent, text, importRanges)
	}
}

// emitScriptSetupContentDirect emits script setup content without type extraction transforms.
func (c *scriptCodegenCtx) emitScriptSetupContentDirect(innerStart int, sourceContent string, text *vue_ast.TextNode, importRanges []core.TextRange) {
	// Start from lastMappedPos to avoid re-emitting text already handled by the first pass
	// (e.g. defineProps/defineEmits/defineModel/defineExpose expression statements)
	pos := c.lastMappedPos
	for _, imp := range importRanges {
		// Skip imports that were already emitted in the first pass
		if imp.End() <= pos {
			continue
		}
		// Emit content before this import
		if pos < imp.Pos() {
			c.mapText(pos, imp.Pos())
		}
		// Emit the import inline (Volar keeps imports inline in the output)
		c.mapText(imp.Pos(), imp.End())
		pos = imp.End()
	}
	// Emit remaining content
	if pos < text.Loc.End() {
		c.mapText(pos, text.Loc.End())
	}
}

// emitScriptSetupContentWithTypeExtraction handles Volar-style type extraction from defineProps/defineEmits.
func (c *scriptCodegenCtx) emitScriptSetupContentWithTypeExtraction(innerStart int, sourceContent string, text *vue_ast.TextNode, importRanges []core.TextRange, propsTypeName, emitsTypeName string) {
	// Start from lastMappedPos to avoid re-emitting text already handled by the first pass
	// (e.g. defineProps/defineEmits/defineModel/defineExpose expression statements)
	pos := c.lastMappedPos

	var statements []*ast.Node
	if c.scriptSetupEl.Ast != nil {
		statements = c.scriptSetupEl.Ast.Statements.Nodes
	}

	for _, statement := range statements {
		stmtStart := innerStart + statement.Pos()
		stmtEnd := innerStart + statement.End()

		// Skip statements already emitted by the first pass
		if stmtEnd <= pos {
			continue
		}

		if statement.Kind == ast.KindVariableStatement {
			for _, d := range statement.AsVariableStatement().DeclarationList.AsVariableDeclarationList().Declarations.Nodes {
				decl := d.AsVariableDeclaration()
				if decl.Initializer == nil || !ast.IsCallExpression(decl.Initializer) {
					continue
				}
				call := decl.Initializer.AsCallExpression()
				callee := call.Expression
				if !ast.IsIdentifier(callee) {
					continue
				}
				calleeName := callee.Text()

				if (calleeName == "defineProps" && propsTypeName == "__VLS_Props") ||
					(calleeName == "defineEmits" && emitsTypeName == "__VLS_Emit") {
					if call.TypeArguments != nil && len(call.TypeArguments.Nodes) == 1 {
						typeArg := call.TypeArguments.Nodes[0]
						if ast.IsTypeLiteralNode(typeArg) {
							// Emit content before this statement's leading trivia
							if pos < stmtStart {
								c.mapText(pos, stmtStart)
							}

							// Extract the type literal content
							typeStart := innerStart + typeArg.Pos()
							typeEnd := innerStart + typeArg.End()
							typeLiteralContent := c.sourceText[typeStart:typeEnd]

							var extractedTypeName string
							if calleeName == "defineProps" {
								extractedTypeName = "__VLS_Props"
							} else {
								extractedTypeName = "__VLS_Emit"
							}

							// Emit leading trivia (whitespace/newlines) before the type extraction
							// stmtStart includes leading trivia; find where actual content starts
							actualContentStart := stmtStart
							for actualContentStart < stmtEnd && (c.sourceText[actualContentStart] == '\n' || c.sourceText[actualContentStart] == '\r' || c.sourceText[actualContentStart] == ' ' || c.sourceText[actualContentStart] == '\t') {
								actualContentStart++
							}
							if actualContentStart > stmtStart {
								c.mapText(stmtStart, actualContentStart)
							}

							// Emit: type __VLS_Props = {...};
							c.serviceText.WriteString("type ")
							c.serviceText.WriteString(extractedTypeName)
							c.serviceText.WriteString(" = ")
							c.serviceText.WriteString(typeLiteralContent)
							c.serviceText.WriteString(";\n")

							// Emit the variable declaration with the extracted type reference
							// "const props = defineProps<__VLS_Props>()"
							typeArgSourceStart := innerStart + typeArg.Pos()
							typeArgSourceEnd := innerStart + typeArg.End()

							// Emit "const props = defineProps<" (from actual content start)
							c.mapText(actualContentStart, typeArgSourceStart)
							// Emit the extracted type name
							c.serviceText.WriteString(extractedTypeName)
							// Skip the original type literal, emit ">()"
							pos = typeArgSourceEnd
							c.mapText(pos, stmtEnd)
							pos = stmtEnd
							goto nextStatement
						}
					}
				}
			}
		}

		// Default: emit statement as-is
		if pos < stmtStart {
			c.mapText(pos, stmtStart)
		}
		c.mapText(stmtStart, stmtEnd)
		pos = stmtEnd

	nextStatement:
	}

	// Emit remaining content after last statement
	if pos < text.Loc.End() {
		c.mapText(pos, text.Loc.End())
	}
}

// processDefineModel collects metadata from a defineModel() call for consolidated type generation.
// localName is the variable name from `const localName = defineModel(...)`, or "" for expression statements.
func (c *scriptCodegenCtx) processDefineModel(innerStart int, call *ast.CallExpression, callLoc core.TextRange, localName string) {
	// Map the defineModel call as-is (Volar emits the raw call inline)
	c.mapText(innerStart+callLoc.Pos(), innerStart+callLoc.End())
	c.lastMappedPos = innerStart + callLoc.End()

	// Parse model name from first argument
	var modelName string
	if len(call.Arguments.Nodes) >= 1 {
		if ast.IsStringLiteral(call.Arguments.Nodes[0]) {
			modelName = call.Arguments.Nodes[0].AsStringLiteral().Text
		}
	}

	camelizedName := "modelValue"
	camelizedModifierName := "model"
	if modelName != "" {
		camelizedName = camelizeStr(modelName)
		camelizedModifierName = camelizedName
	}

	// Extract type argument text
	var typeText string
	var modifierTypeText string
	if call.TypeArguments != nil && len(call.TypeArguments.Nodes) >= 1 {
		typeArg := call.TypeArguments.Nodes[0]
		typeText = c.sourceText[innerStart+typeArg.Pos() : innerStart+typeArg.End()]
		if len(call.TypeArguments.Nodes) >= 2 {
			modArg := call.TypeArguments.Nodes[1]
			modifierTypeText = c.sourceText[innerStart+modArg.Pos() : innerStart+modArg.End()]
		}
	}

	// Parse options object (second arg if first is string, first arg if it's an object)
	var required bool
	var hasDefault bool
	var hasRuntimeType bool
	var optionsArg *ast.Node
	if len(call.Arguments.Nodes) >= 2 && ast.IsObjectLiteralExpression(call.Arguments.Nodes[1]) {
		optionsArg = call.Arguments.Nodes[1]
	} else if len(call.Arguments.Nodes) >= 1 && ast.IsObjectLiteralExpression(call.Arguments.Nodes[0]) {
		optionsArg = call.Arguments.Nodes[0]
	}
	if optionsArg != nil {
		for _, prop := range optionsArg.AsObjectLiteralExpression().Properties.Nodes {
			if !ast.IsPropertyAssignment(prop) {
				continue
			}
			pa := prop.AsPropertyAssignment()
			if pa.Name() == nil {
				continue
			}
			propName := pa.Name().Text()
			switch propName {
			case "required":
				if pa.Initializer != nil && pa.Initializer.Kind == ast.KindTrueKeyword {
					required = true
				}
			case "default":
				hasDefault = true
			case "type":
				hasRuntimeType = true
			}
		}
	}

	// Determine type text: Volar's priority is:
	// 1. Explicit type arg: defineModel<T>() → "T"
	// 2. Runtime type + local name: defineModel('name', { type: X }) → "typeof localName['value']"
	// 3. Default value + prop name: defineModel({ default: v }) → "typeof __VLS_defaultModels['propName']"
	// 4. Fallback: "any"
	if typeText == "" && hasRuntimeType && localName != "" {
		typeText = "typeof " + localName + "['value']"
	}

	// Check for duplicate model names
	if c.seenDefineModels.Has(camelizedName) {
		callee := call.Expression
		calleeLoc := utils.TrimNodeTextRange(c.scriptSetupEl.Ast, callee)
		c.reportDiagnostic(utils.MoveTextRange(calleeLoc, innerStart), vue_diagnostics.Duplicate_model_name_X_0, camelizedName)
	} else {
		c.seenDefineModels.Add(camelizedName)
	}

	c.defineModels = append(c.defineModels, defineModelInfo{
		camelizedName:         camelizedName,
		camelizedModifierName: camelizedModifierName,
		typeText:              typeText,
		modifierTypeText:      modifierTypeText,
		required:              required,
		hasDefault:            hasDefault,
	})
}

// emitModelTypes generates consolidated __VLS_ModelProps and __VLS_ModelEmit types from collected defineModel info.
func (c *scriptCodegenCtx) emitModelTypes() {
	if len(c.defineModels) == 0 {
		return
	}

	// __VLS_ModelProps
	c.serviceText.WriteString("type __VLS_ModelProps = {\n")
	for _, m := range c.defineModels {
		propType := m.typeText
		if propType == "" {
			propType = "any"
		}
		if m.camelizedName == "modelValue" {
			// Default model: no quotes for modelValue
			c.serviceText.WriteString(m.camelizedName)
		} else {
			c.serviceText.WriteString("'")
			c.serviceText.WriteString(m.camelizedName)
			c.serviceText.WriteString("'")
		}
		if m.required {
			c.serviceText.WriteString(": ")
		} else {
			c.serviceText.WriteString("?: ")
		}
		c.serviceText.WriteString(propType)
		c.serviceText.WriteString(";\n")

		// Modifiers property
		if m.modifierTypeText != "" {
			c.serviceText.WriteString("'")
			c.serviceText.WriteString(m.camelizedModifierName)
			c.serviceText.WriteString("Modifiers'?: Partial<Record<")
			c.serviceText.WriteString(m.modifierTypeText)
			c.serviceText.WriteString(", true>>;\n")
		}
	}
	c.serviceText.WriteString("};\n")

	// __VLS_ModelEmit
	c.serviceText.WriteString("type __VLS_ModelEmit = {\n")
	for _, m := range c.defineModels {
		propType := m.typeText
		if propType == "" {
			propType = "any"
		}
		c.serviceText.WriteString("'update:")
		c.serviceText.WriteString(m.camelizedName)
		c.serviceText.WriteString("': [value: ")
		c.serviceText.WriteString(propType)
		if !m.required && !m.hasDefault {
			c.serviceText.WriteString(" | undefined")
		}
		c.serviceText.WriteString("];\n")
	}
	c.serviceText.WriteString("};\n")

	// __VLS_modelEmit instance
	c.serviceText.WriteString("const __VLS_modelEmit = defineEmits<__VLS_ModelEmit>();\n")
}

// emitTypeRef returns the type reference for $emit in the context.
// Volar uses "typeof emitsVar" for defineEmits, "typeof __VLS_modelEmit" for defineModel.
func (c *scriptCodegenCtx) emitTypeRef(emitsVariableName string) string {
	if emitsVariableName != "" && len(c.defineModels) > 0 {
		return "__VLS_PublicEmits"
	}
	if emitsVariableName != "" {
		return "typeof " + emitsVariableName
	}
	return "typeof __VLS_modelEmit"
}

// propsTypeRef returns the type reference for $props in the context.
func (c *scriptCodegenCtx) propsTypeRef(propsVariableName, propsTypeName string) string {
	if propsVariableName != "" && len(c.defineModels) > 0 {
		// Both defineProps and defineModel — use the combined __VLS_PublicProps
		return "__VLS_PublicProps"
	}
	if len(c.defineModels) > 0 {
		return "__VLS_ModelProps"
	}
	if propsTypeName != "" {
		return propsTypeName
	}
	if propsVariableName != "" {
		return "typeof " + propsVariableName
	}
	return "__VLS_PublicProps"
}

// emitTypeForExport returns the emit type to use in the defineComponent export.
// Volar uses literal types (e.g. __VLS_Emit, __VLS_ModelEmit) in the export.
func (c *scriptCodegenCtx) emitTypeForExport(emitsVariableName, emitsTypeName string) string {
	hasDefineEmits := emitsVariableName != ""
	hasDefineModel := len(c.defineModels) > 0

	if hasDefineEmits && hasDefineModel {
		// Both: use combined __VLS_PublicEmits
		return "__VLS_PublicEmits"
	}
	if hasDefineModel {
		// Only model: use literal type
		return "__VLS_ModelEmit"
	}
	if emitsTypeName != "" {
		// Extracted type (e.g. __VLS_Emit)
		return emitsTypeName
	}
	// defineEmits without type extraction: use typeof
	return "typeof " + emitsVariableName
}

func (c *scriptCodegenCtx) collectBindingRanges(innerStart int, node *ast.Node) {
	switch node.Kind {
	case ast.KindVariableStatement:
		for _, d := range node.AsVariableStatement().DeclarationList.AsVariableDeclarationList().Declarations.Nodes {
			decl := d.AsVariableDeclaration()
			name := decl.Name()
			var visitor ast.Visitor
			// TODO: types, etc
			// TODO: declare const?
			visitor = func(n *ast.Node) bool {
				if ast.IsIdentifier(n) {
					c.bindingNames = append(c.bindingNames, n.AsIdentifier().Text)
				} else if ast.IsBindingPattern(n) {
					for _, el := range n.AsBindingPattern().Elements.Nodes {
						if ast.IsBindingElement(el) {
							visitor(el.Name())
						}
					}
				} else {
					n.ForEachChild(visitor)
				}
				return false
			}
			visitor(name)
		}
	case ast.KindFunctionDeclaration, ast.KindClassDeclaration, ast.KindEnumDeclaration:
		if name := node.Name(); name != nil {
			c.bindingNames = append(c.bindingNames, name.AsIdentifier().Text)
		}
	case ast.KindImportDeclaration:
		importClause := node.AsImportDeclaration().ImportClause
		if importClause == nil || importClause.IsTypeOnly() {
			return
		}

		if importClause.Name() != nil {
			c.bindingNames = append(c.bindingNames, importClause.Name().AsIdentifier().Text)
		}

		namedBindings := importClause.AsImportClause().NamedBindings
		if namedBindings != nil {
			if ast.IsNamespaceImport(namedBindings) {
				c.bindingNames = append(c.bindingNames, namedBindings.Name().AsIdentifier().Text)
			} else {
				for _, element := range namedBindings.Elements() {
					if !element.IsTypeOnly() {
						c.bindingNames = append(c.bindingNames, element.Name().AsIdentifier().Text)
					}
				}
			}
		}

	}
}
