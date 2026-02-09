package vue_codegen

import (
	"strings"

	"github.com/auvred/golar/internal/collections"
	"github.com/auvred/golar/internal/utils"
	"github.com/auvred/golar/internal/vue/ast"
	"github.com/auvred/golar/internal/vue/diagnostics"
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/core"
)

// TODO: <script src="">

type scriptCodegenCtx struct {
	*codegenCtx
	scriptSetupEl *vue_ast.ElementNode
	scriptEl      *vue_ast.ElementNode
	lastMappedPos int

	bindingRanges []core.TextRange

	seenDefineModels        collections.Set[string]
	modelPropsVariableNames []string
	modelEmitsVariableNames []string
}

func generateScript(base *codegenCtx, scriptSetupEl *vue_ast.ElementNode, scriptEl *vue_ast.ElementNode, templateEl *vue_ast.ElementNode) {
	c := scriptCodegenCtx{
		codegenCtx:    base,
		scriptSetupEl: scriptSetupEl,
		scriptEl:      scriptEl,
	}

	// TODO: without "ts" lang
	// TODO: tsx

	// we don't import define* macros because they're globally available
	// https://github.com/vuejs/core/blob/aac7e1898907445c8f89b22047a9bfcf0a6e91b8/packages/runtime-core/types/scriptSetupHelpers.d.ts
	c.serviceText.WriteString("import { defineComponent as __VLS_DefineComponent } from 'vue'\n")

	var selfType string
	if c.scriptEl != nil {
		if len(c.scriptEl.Children) != 1 {
			panic("TODO: len of <script> children != 1")
		}

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
					c.mapRange(innerStart+exportLoc.Pos(), innerStart+export.Expression.Pos(), c.serviceText.Len(), c.serviceText.Len()+len("__VLS_DefineComponent"))
					c.serviceText.WriteString("__VLS_DefineComponent(")
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
		if len(c.scriptSetupEl.Children) != 1 {
			panic("TODO: len of <script setup> children != 1")
		}

		c.serviceText.WriteString("const __VLS_Export = ")

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

				c.serviceText.WriteByte('<')
				c.mapText(attr.Value.Loc.Pos(), attr.Value.Loc.End())
				if !strings.HasSuffix(attr.Value.Content, ",") {
					c.serviceText.WriteByte(',')
				}
				c.serviceText.WriteString(`>(
__VLS_GenericProps: NonNullable<Awaited<typeof __VLS_GenericSetup>>['props'],
__VLS_GenericCtx?: __VLS_PrettifyGlobal<Pick<NonNullable<Awaited<typeof __VLS_GenericSetup>>, 'attrs' | 'emit' | 'slots'>>,
__VLS_GenericExposed?: NonNullable<Awaited<typeof __VLS_GenericSetup>>['expose'],
__VLS_GenericSetup = `)
			}
		}

		text := c.scriptSetupEl.Children[0].AsText()

		c.serviceText.WriteString("(async () => {\n")
		innerStart := c.scriptSetupEl.InnerLoc.Pos()

		c.lastMappedPos = text.Loc.Pos()

		var (
			propsVariableName string
			emitsVariableName string
			slotsVariableName string
			hasExpose bool
		)

		// TODO: report nested compiler macros (vue compiler errors on them)
		// TODO: report incorrect compiler macros arguments
		// TODO: $emits, $props, emitstoprops

		importRanges := []core.TextRange{}
		for _, statement := range c.scriptSetupEl.Ast.Statements.Nodes {
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
						modelVariableName := c.newInternalVariable()
						callLoc := utils.TrimNodeTextRange(c.scriptSetupEl.Ast, call.AsNode())
						c.mapText(c.lastMappedPos, innerStart+callLoc.Pos())
						c.lastMappedPos = innerStart + callLoc.Pos()
						c.serviceText.WriteString("{} as unknown as typeof ")
						c.serviceText.WriteString(modelVariableName)
						c.processDefineModel(innerStart, call, callLoc, modelVariableName)
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
					c.mapText(c.lastMappedPos, innerStart+statement.End())
					c.lastMappedPos = innerStart + statement.End()
					c.serviceText.WriteString("\nconst __VLS_Expose = __VLS_DefineExpose")
					c.serviceText.WriteString(c.sourceText[innerStart + callee.End():innerStart+statement.End()])
					c.serviceText.WriteString("\n")
				case "defineModel":
					modelVariableName := c.newInternalVariable()
					callLoc := utils.TrimNodeTextRange(c.scriptSetupEl.Ast, call.AsNode())
					c.mapText(c.lastMappedPos, innerStart+callLoc.Pos())
					c.lastMappedPos = innerStart + callLoc.Pos()
					c.processDefineModel(innerStart, call, callLoc, modelVariableName)
				}
			case ast.KindImportDeclaration:
				importRanges = append(importRanges, utils.MoveTextRange(statement.Loc, innerStart))
				if c.lastMappedPos != statement.Pos() {
					c.mapText(c.lastMappedPos, innerStart+statement.Pos())
				}
				c.lastMappedPos = innerStart + statement.End()
			}
		}
		c.mapText(c.lastMappedPos, text.Loc.End())
		c.serviceText.WriteByte('\n')

		c.serviceText.WriteString("type __VLS_SetupExposed = import('vue').ShallowUnwrapRef<{\n")
		for _, binding := range c.bindingRanges {
			c.serviceText.WriteString(c.sourceText[binding.Pos():binding.End()])
			c.serviceText.WriteString(": typeof ")
			c.serviceText.WriteString(c.sourceText[binding.Pos():binding.End()])
			c.serviceText.WriteRune('\n')
		}
		c.serviceText.WriteString("}>\n")

		hasPublicProps := false
		startPublicProps := func() bool {
			if hasPublicProps {
				return false
			}
			c.serviceText.WriteString("\ntype __VLS_PublicProps = ")
			hasPublicProps = true
			return true
		}
		if propsVariableName != "" {
			startPublicProps()
			c.serviceText.WriteString("typeof ")
			c.serviceText.WriteString(propsVariableName)
		}
		for i, varName := range c.modelPropsVariableNames {
			if i > 0 || !startPublicProps() {
				c.serviceText.WriteString(" & ")
			}
			c.serviceText.WriteString(varName)
		}

		hasPublicEmits := false
		startPublicEmits := func() bool {
			if hasPublicEmits {
				return false
			}
			c.serviceText.WriteString("\ntype __VLS_PublicEmits = ")
			hasPublicEmits = true
			return true
		}
		if emitsVariableName != "" {
			startPublicEmits()
			c.serviceText.WriteString("typeof ")
			c.serviceText.WriteString(emitsVariableName)
		}
		for i, varName := range c.modelEmitsVariableNames {
			if i > 0 || !startPublicEmits() {
				c.serviceText.WriteString(" & ")
			}
			c.serviceText.WriteString("typeof ")
			c.serviceText.WriteString(varName)
		}
		if hasPublicEmits {
			c.serviceText.WriteString("\ntype __VLS_EmitProps = __VLS_EmitsToProps<__VLS_NormalizeEmits<__VLS_PublicEmits>>")
		}

		c.serviceText.WriteString("\nconst __VLS_Ctx = {\n")
		if selfType != "" {
			c.serviceText.WriteString("...{} as unknown as InstanceType<__VLS_PickNotAny<typeof ")
			c.serviceText.WriteString(selfType)
			c.serviceText.WriteString(" extends new () => {} ? typeof")
			c.serviceText.WriteString(selfType)
			c.serviceText.WriteString(" : new () => {}, new () => {}>>,\n")
		} else {
			c.serviceText.WriteString("...{} as unknown as import('vue').ComponentPublicInstance,\n")
		}
		c.serviceText.WriteString("...{} as unknown as __VLS_SetupExposed,\n")
		if hasPublicProps {
			c.serviceText.WriteString("...{} as unknown as __VLS_PublicProps,\n")
			// TODO: other $s
			c.serviceText.WriteString("...{} as unknown as { $props: __VLS_PublicProps },\n")
		}
		if hasPublicEmits {
			c.serviceText.WriteString("...{} as unknown as { $emit: __VLS_PublicEmits },\n")
		}
		c.serviceText.WriteString("}\n")

		generateTemplate(c.codegenCtx, templateEl)

		if hasGeneric {
			c.serviceText.WriteString("return {} as unknown as {\nprops: ")
			// TODO: defineProps arg
			if c.options.Version.hasPublicPropsType() {
				c.serviceText.WriteString("import('vue').PublicProps")
			} else {
				c.serviceText.WriteString("import('vue').VNodeProps & import('vue').AllowedComponentProps & import('vue').ComponentCustomProps")
			}
			if hasPublicProps {
				c.serviceText.WriteString(" & __VLS_PublicProps")
			}
			if hasPublicEmits {
				c.serviceText.WriteString(" & __VLS_EmitProps")
			}
			// TODO: defineExpose
			c.serviceText.WriteString("\nexpose: (exposed: {}) => void")
			c.serviceText.WriteString("\nattrs: any")
			c.serviceText.WriteString("\nslots: ")
			if slotsVariableName == "" {
				c.serviceText.WriteString("{}")
			} else {
				c.serviceText.WriteString(slotsVariableName)
			}
			c.serviceText.WriteString("\nemit: ")
			if hasPublicEmits {
				c.serviceText.WriteString("__VLS_PublicEmits")
			} else {
				c.serviceText.WriteString("{}")
			}
			c.serviceText.WriteString("\n}\n})()\n) => ({} as unknown as import('vue').VNode & { __ctx?: Awaited<typeof __VLS_GenericSetup> })\n")
		} else {
			c.serviceText.WriteString("\nconst __VLS_Base = __VLS_DefineComponent({\n")
			// TODO: withDefaults
			// TODO: defineProps(arg)
			if hasPublicProps {
				if c.options.Version.supportsTypeProps() {
					c.serviceText.WriteString("__typeProps: {} as unknown as __VLS_PublicProps,\n")
				} else {
					c.serviceText.WriteString("props: {} as unknown as __VLS_TypePropsToOption<__VLS_PublicProps>,\n")
				}
			}
			if hasPublicEmits {
				if c.options.Version.supportsTypeEmits() {
					c.serviceText.WriteString("__typeEmits: {} as unknown as __VLS_PublicEmits,\n")
				} else {
					c.serviceText.WriteString("emits: {} as unknown as __VLS_NormalizeEmits<__VLS_PublicEmits>,\n")
				}
			}
			if hasExpose {
				c.serviceText.WriteString("setup: () => __VLS_Expose,\n")
			}
			c.serviceText.WriteString("})\n")

			if slotsVariableName == "" {
				c.serviceText.WriteString("return __VLS_Base\n")
			} else {
				c.serviceText.WriteString("return {} as unknown as __VLS_WithSlots<typeof __VLS_Base, typeof ")
				c.serviceText.WriteString(slotsVariableName)
				c.serviceText.WriteString(">\n")
			}

			c.serviceText.WriteString("\n})()\n")
		}

		for _, loc := range importRanges {
			c.mapText(loc.Pos(), loc.End())
			c.serviceText.WriteString("\n")
		}

		c.serviceText.WriteString("\nexport default {} as unknown as Awaited<typeof __VLS_Export>\n")
	} else {
		c.serviceText.WriteString("\nconst __VLS_Ctx = {\n")
		if selfType != "" {
			c.serviceText.WriteString("...{} as unknown as InstanceType<__VLS_PickNotAny<typeof ")
			c.serviceText.WriteString(selfType)
			c.serviceText.WriteString(" extends new () => {} ? typeof ")
			c.serviceText.WriteString(selfType)
			c.serviceText.WriteString(" : new () => {}, new () => {}>>,\n")
		} else {
			c.serviceText.WriteString("...{} as unknown as import('vue').ComponentPublicInstance,\n")
		}
		c.serviceText.WriteString("}\n")
		generateTemplate(c.codegenCtx, templateEl)
	}
}

// TODO: reconcile with vuejs/core/packages/compiler-sfc/src/script/defineModel.ts
func (c *scriptCodegenCtx) parseDefineModel(expr *ast.CallExpression) string {
	var name string
	if len(expr.Arguments.Nodes) >= 1 {
		if ast.IsStringLiteral(expr.Arguments.Nodes[0]) {
			name = expr.Arguments.Nodes[0].AsStringLiteral().Text
		}
	}

	return name
}

func (c *scriptCodegenCtx) processDefineModel(innerStart int, call *ast.CallExpression, callLoc core.TextRange, modelVariableName string) {
	modelName := c.parseDefineModel(call)
	c.serviceText.WriteString("\nconst ")
	c.serviceText.WriteString(modelVariableName)
	c.serviceText.WriteString(" = ")
	c.mapText(innerStart+callLoc.Pos(), innerStart+callLoc.End())
	c.lastMappedPos = innerStart + callLoc.End()
	modelTypesVariableName := c.newInternalVariable()
	c.serviceText.WriteString("\ntype ")
	c.serviceText.WriteString(modelTypesVariableName)
	c.serviceText.WriteString(" = typeof ")
	c.serviceText.WriteString(modelVariableName)
	c.serviceText.WriteString(" extends import('vue').ModelRef<infer T, infer M extends string | number | symbol")
	if c.options.Version.modelRefHasGetterAndSetter() {
		c.serviceText.WriteString(", any, any")
	}
	c.serviceText.WriteString("> ? [T, M] : never\n")
	modelPropTypeVariableName := c.newInternalVariable()
	c.modelPropsVariableNames = append(c.modelPropsVariableNames, modelPropTypeVariableName)
	c.serviceText.WriteString("type ")
	c.serviceText.WriteString(modelPropTypeVariableName)
	c.serviceText.WriteString(" = (undefined extends ")
	c.serviceText.WriteString(modelTypesVariableName)
	c.serviceText.WriteString("[0] ? { '")
	// TODO: don't use quotes when not needed
	// TODO: escape
	camelizedModelName := "modelValue"
	camelizedModelNameForModifiers := "model"
	if modelName == "" {
		c.serviceText.WriteString(camelizedModelName)
	} else {
		camelizedModelName = camelize(modelName, &c.serviceText)
		camelizedModelNameForModifiers = camelizedModelName
	}
	c.serviceText.WriteString("'?: ")
	c.serviceText.WriteString(modelTypesVariableName)
	c.serviceText.WriteString("[0] } : { '")
	c.serviceText.WriteString(camelizedModelName)
	c.serviceText.WriteString("': ")
	c.serviceText.WriteString(modelTypesVariableName)
	c.serviceText.WriteString("[0] }) & { '")
	c.serviceText.WriteString(camelizedModelNameForModifiers)
	c.serviceText.WriteString("Modifiers'?: Partial<Record<")
	c.serviceText.WriteString(modelTypesVariableName)
	c.serviceText.WriteString("[1], true>> }\n")

	modelEmitTypeVariableName := c.newInternalVariable()
	c.modelEmitsVariableNames = append(c.modelEmitsVariableNames, modelEmitTypeVariableName)
	c.serviceText.WriteString("const ")
	c.serviceText.WriteString(modelEmitTypeVariableName)
	// TODO: escape?
	c.serviceText.WriteString(" = defineEmits<{ 'update:")
	c.serviceText.WriteString(camelizedModelName)
	c.serviceText.WriteString("': [value: ")
	c.serviceText.WriteString(modelTypesVariableName)
	c.serviceText.WriteString("[0]] }>()\n")

	if c.seenDefineModels.Has(camelizedModelName) {
		callee := call.Expression
		calleeLoc := utils.TrimNodeTextRange(c.scriptSetupEl.Ast, callee)
		c.reportDiagnostic(utils.MoveTextRange(calleeLoc, innerStart), vue_diagnostics.Duplicate_model_name_X_0, camelizedModelName)
	} else {
		c.seenDefineModels.Add(camelizedModelName)
	}
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
					c.bindingRanges = append(c.bindingRanges, utils.MoveTextRange(n.Loc, innerStart))
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
			c.bindingRanges = append(c.bindingRanges, utils.MoveTextRange(name.Loc, innerStart))
		}
	case ast.KindImportDeclaration:
		importClause := node.AsImportDeclaration().ImportClause
		if importClause == nil || importClause.IsTypeOnly() {
			return
		}

		if importClause.Name() != nil {
			c.bindingRanges = append(c.bindingRanges, utils.MoveTextRange(importClause.Name().Loc, innerStart))
		}

		namedBindings := importClause.AsImportClause().NamedBindings
		if namedBindings != nil {
			if ast.IsNamespaceImport(namedBindings) {
				c.bindingRanges = append(c.bindingRanges, utils.MoveTextRange(namedBindings.Name().Loc, innerStart))
			} else {
				for _, element := range namedBindings.Elements() {
					if !element.IsTypeOnly() {
						c.bindingRanges = append(c.bindingRanges, utils.MoveTextRange(element.Name().Loc, innerStart))
					}
				}
			}
		}

	}
}
