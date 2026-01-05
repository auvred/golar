package vue_codegen

import (
	vue_ast "github.com/auvred/golar/internal/vue/ast"
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/core"
)

// getAttributeValue returns the value of an attribute by name, or empty string if not found
func getAttributeValue(el *vue_ast.ElementNode, name string) string {
	for _, prop := range el.Props {
		if prop.Kind == vue_ast.KindAttribute {
			attr := prop.AsAttribute()
			if attr.Name == name && attr.Value != nil {
				return attr.Value.Content
			}
		}
	}
	return ""
}

type scriptCodegenCtx struct {
	*codegenCtx
	scriptSetupEl *vue_ast.ElementNode
	scriptEl      *vue_ast.ElementNode
	templateEl    *vue_ast.ElementNode // nil if template should be generated separately
}

// definePropsInfo holds information about a defineProps call
type definePropsInfo struct {
	// Variable name if assigned (e.g., "props" in `const props = defineProps<T>()`)
	// Empty if not assigned to a variable
	varName string
	// Type argument range if using type-only syntax: defineProps<{ msg: string }>()
	typeArgRange *core.TextRange
	// Runtime argument range if using runtime syntax: defineProps({ msg: String })
	runtimeArgRange *core.TextRange
}

// extractDefinePropsFromCall extracts defineProps info from a call expression.
// Returns nil if the call is not defineProps or withDefaults(defineProps<T>(), {...})
func extractDefinePropsFromCall(callExpr *ast.Node) *definePropsInfo {
	if callExpr == nil || !ast.IsCallExpression(callExpr) {
		return nil
	}

	call := callExpr.AsCallExpression()

	// Check for direct defineProps call
	if ast.IsIdentifier(call.Expression) {
		if call.Expression.AsIdentifier().Text == "defineProps" {
			info := &definePropsInfo{}

			// Check for type argument: defineProps<{ msg: string }>()
			if call.TypeArguments != nil && len(call.TypeArguments.Nodes) > 0 {
				typeArg := call.TypeArguments.Nodes[0]
				r := core.NewTextRange(typeArg.Pos(), typeArg.End())
				info.typeArgRange = &r
			}

			// Check for runtime argument: defineProps({ msg: String })
			if call.Arguments != nil && len(call.Arguments.Nodes) > 0 {
				arg := call.Arguments.Nodes[0]
				r := core.NewTextRange(arg.Pos(), arg.End())
				info.runtimeArgRange = &r
			}

			return info
		}

		// Check for withDefaults(defineProps<T>(), {...})
		if call.Expression.AsIdentifier().Text == "withDefaults" {
			if call.Arguments != nil && len(call.Arguments.Nodes) > 0 {
				// First argument should be defineProps call
				firstArg := call.Arguments.Nodes[0]
				if ast.IsCallExpression(firstArg) {
					innerCall := firstArg.AsCallExpression()
					if ast.IsIdentifier(innerCall.Expression) &&
						innerCall.Expression.AsIdentifier().Text == "defineProps" {
						info := &definePropsInfo{}

						// Extract type argument from inner defineProps
						if innerCall.TypeArguments != nil && len(innerCall.TypeArguments.Nodes) > 0 {
							typeArg := innerCall.TypeArguments.Nodes[0]
							r := core.NewTextRange(typeArg.Pos(), typeArg.End())
							info.typeArgRange = &r
						}

						return info
					}
				}
			}
		}
	}

	return nil
}

// findDefineProps looks for defineProps calls in the script setup AST
func findDefineProps(file *ast.SourceFile) *definePropsInfo {
	for _, stmt := range file.Statements.Nodes {
		var callExpr *ast.Node
		var varName string

		switch stmt.Kind {
		case ast.KindVariableStatement:
			// const props = defineProps<T>() or const props = withDefaults(defineProps<T>(), {...})
			decls := stmt.AsVariableStatement().DeclarationList.AsVariableDeclarationList().Declarations.Nodes
			for _, decl := range decls {
				varDecl := decl.AsVariableDeclaration()
				if varDecl.Initializer != nil && ast.IsCallExpression(varDecl.Initializer) {
					callExpr = varDecl.Initializer
					if ast.IsIdentifier(varDecl.Name()) {
						varName = varDecl.Name().AsIdentifier().Text
					}
				}
			}
		case ast.KindExpressionStatement:
			// Standalone defineProps<T>() or withDefaults(defineProps<T>(), {...})
			expr := stmt.AsExpressionStatement().Expression
			if ast.IsCallExpression(expr) {
				callExpr = expr
			}
		}

		if info := extractDefinePropsFromCall(callExpr); info != nil {
			info.varName = varName
			return info
		}
	}

	return nil
}

func generateScript(base *codegenCtx, scriptSetupEl *vue_ast.ElementNode, scriptEl *vue_ast.ElementNode, templateEl *vue_ast.ElementNode) {
	c := scriptCodegenCtx{
		codegenCtx:    base,
		scriptSetupEl: scriptSetupEl,
		scriptEl:      scriptEl,
		templateEl:    templateEl,
	}

	c.serviceText.WriteString("import { defineComponent as __VLS_DefineComponent } from 'vue'\n")

	// Handle template-only components (no <script> or <script setup>)
	if c.scriptEl == nil && c.scriptSetupEl == nil {
		c.serviceText.WriteString("const __VLS_ctx = {} as import('vue').ComponentPublicInstance;\n")
		c.serviceText.WriteString("type __VLS_LocalComponents = {};\n")
		c.serviceText.WriteString("type __VLS_GlobalComponents = import('vue').GlobalComponents;\n")
		c.serviceText.WriteString("let __VLS_components!: __VLS_LocalComponents & __VLS_GlobalComponents;\n")
		c.serviceText.WriteString("let __VLS_intrinsics!: import('vue/jsx-runtime').JSX.IntrinsicElements;\n")
		c.serviceText.WriteString("type __VLS_LocalDirectives = {};\n")
		c.serviceText.WriteString("let __VLS_directives!: __VLS_LocalDirectives & import('vue').GlobalDirectives;\n")
		return
	}

	var selfType string
	if c.scriptEl != nil {
		// Handle <script src="..."> - external script source
		if srcAttr := getAttributeValue(c.scriptEl, "src"); srcAttr != "" {
			// Generate import from external source and re-export
			c.serviceText.WriteString("import __VLS_default from '")
			c.serviceText.WriteString(srcAttr)
			c.serviceText.WriteString("';\n")
			c.serviceText.WriteString("export default __VLS_default;;\n")
			selfType = "__VLS_default"
			// External script with src doesn't have inline content, skip the rest
		} else {
			// Handle inline <script> content
			if len(c.scriptEl.Children) != 1 {
				panic("TODO: len of <script> children != 1")
			}

			innerStart := c.scriptEl.InnerLoc.Pos()
			text := c.scriptEl.Children[0].AsText()

			mapStart := text.Loc.Pos()
			hasExportDefault := false

			for _, statement := range c.scriptEl.Ast.Statements.Nodes {
				if !ast.IsExportAssignment(statement) {
					continue
				}

				hasExportDefault = true
				export := statement.AsExportAssignment()
				c.mapText(mapStart, innerStart+export.Expression.Pos())
				c.serviceText.WriteString(" {} as unknown as typeof __VLS_Export\n")
				if c.scriptSetupEl == nil {
					c.serviceText.WriteString("const __VLS_Export = ")
					selfType = "__VLS_Export"
				} else {
					c.serviceText.WriteString("const __VLS_Self = ")
					selfType = "__VLS_Self"
				}
				mapStart = innerStart + export.Expression.Pos()

				break
			}

			c.mapText(mapStart, text.Loc.End())
			c.serviceText.WriteString("\n\n")

			// Only create __VLS_Export for regular script if there's no script setup
			// When both exist, script setup will create __VLS_Export
			if !hasExportDefault && c.scriptSetupEl == nil {
				c.serviceText.WriteString("const __VLS_Export = __VLS_DefineComponent({})\nexport default __VLS_Export\n")
				selfType = "__VLS_Export"
			}

			// TODO: options wrapper - wrap export default |defineComponent(|{}|)|
		}

		// For regular <script> (no setup), generate __VLS_ctx and component types
		if c.scriptSetupEl == nil && selfType != "" {
			c.serviceText.WriteString("const __VLS_ctx = {} as InstanceType<__VLS_PickNotAny<typeof ")
			c.serviceText.WriteString(selfType)
			c.serviceText.WriteString(", new () => {}>>;\n")
			c.serviceText.WriteString("type __VLS_LocalComponents = {};\n")
			c.serviceText.WriteString("type __VLS_GlobalComponents = import('vue').GlobalComponents;\n")
			c.serviceText.WriteString("let __VLS_components!: __VLS_LocalComponents & __VLS_GlobalComponents;\n")
			c.serviceText.WriteString("let __VLS_intrinsics!: import('vue/jsx-runtime').JSX.IntrinsicElements;\n")
			c.serviceText.WriteString("type __VLS_LocalDirectives = {};\n")
			c.serviceText.WriteString("let __VLS_directives!: __VLS_LocalDirectives & import('vue').GlobalDirectives;\n")
		}
	}

	// TODO: generic support
	if c.scriptSetupEl != nil {
		// Handle empty <script setup>
		if len(c.scriptSetupEl.Children) == 0 {
			if c.scriptEl != nil && selfType != "" {
				// Empty <script setup> with regular <script>
				// Generate the async IIFE wrapper with ctx based on the regular script's export
				c.serviceText.WriteString("const __VLS_export = await (async () => {\n")
				c.serviceText.WriteString("const __VLS_ctx = {} as InstanceType<__VLS_PickNotAny<typeof ")
				c.serviceText.WriteString(selfType)
				c.serviceText.WriteString(", new () => {}>>;\n")
				c.serviceText.WriteString("type __VLS_LocalComponents = {};\n")
				c.serviceText.WriteString("type __VLS_GlobalComponents = import('vue').GlobalComponents;\n")
				c.serviceText.WriteString("let __VLS_components!: __VLS_LocalComponents & __VLS_GlobalComponents;\n")
				c.serviceText.WriteString("let __VLS_intrinsics!: import('vue/jsx-runtime').JSX.IntrinsicElements;\n")
				c.serviceText.WriteString("type __VLS_LocalDirectives = {};\n")
				c.serviceText.WriteString("let __VLS_directives!: __VLS_LocalDirectives & import('vue').GlobalDirectives;\n")
				c.serviceText.WriteString("return (await import('vue')).defineComponent({});\n")
				c.serviceText.WriteString("})();\n")
			} else {
				// Just empty <script setup> with no regular script
				c.serviceText.WriteString("const __VLS_ctx = {} as import('vue').ComponentPublicInstance;\n")
				c.serviceText.WriteString("type __VLS_LocalComponents = {};\n")
				c.serviceText.WriteString("type __VLS_GlobalComponents = import('vue').GlobalComponents;\n")
				c.serviceText.WriteString("let __VLS_components!: __VLS_LocalComponents & __VLS_GlobalComponents;\n")
				c.serviceText.WriteString("let __VLS_intrinsics!: import('vue/jsx-runtime').JSX.IntrinsicElements;\n")
				c.serviceText.WriteString("type __VLS_LocalDirectives = {};\n")
				c.serviceText.WriteString("let __VLS_directives!: __VLS_LocalDirectives & import('vue').GlobalDirectives;\n")
			}
			return
		}
		if len(c.scriptSetupEl.Children) != 1 {
			panic("TODO: len of <script setup> children != 1")
		}

		text := c.scriptSetupEl.Children[0].AsText()

		if c.scriptEl != nil {
			// When both <script> and <script setup> exist, add export default first
			c.serviceText.WriteString("export default {} as typeof __VLS_Export\n")
			c.serviceText.WriteString("const __VLS_Export = await (async () => {\n")
		} else {
			// TODO
			c.serviceText.WriteString("const __VLS_Export = __VLS_DefineComponent({})\n")
		}
		innerStart := c.scriptSetupEl.InnerLoc.Pos()

		// Find defineProps call
		propsInfo := findDefineProps(c.scriptSetupEl.Ast)

		// Collect type declarations to hoist and other statement ranges
		typeRanges := []core.TextRange{}
		bindingRanges := []core.TextRange{}
		bindingNames := []string{} // Store actual identifier names for setupConsts
		importRanges := []core.TextRange{}

		for _, statement := range c.scriptSetupEl.Ast.Statements.Nodes {
			switch statement.Kind {
			case ast.KindTypeAliasDeclaration, ast.KindInterfaceDeclaration:
				// Collect type/interface declarations for hoisting
				typeRanges = append(typeRanges, core.NewTextRange(innerStart+statement.Loc.Pos(), innerStart+statement.Loc.End()))
			case ast.KindVariableStatement:
				for _, decl := range statement.AsVariableStatement().DeclarationList.AsVariableDeclarationList().Declarations.Nodes {
					name := decl.AsVariableDeclaration().Name()
					var visitor ast.Visitor
					visitor = func(n *ast.Node) bool {
						if ast.IsIdentifier(n) {
							bindingRanges = append(bindingRanges, n.Loc)
							bindingNames = append(bindingNames, n.AsIdentifier().Text)
						}
						return n.ForEachChild(visitor)
					}
					visitor(name)
				}
			case ast.KindFunctionDeclaration, ast.KindClassDeclaration, ast.KindEnumDeclaration:
				if name := statement.Name(); name != nil {
					bindingRanges = append(bindingRanges, name.Loc)
					if ast.IsIdentifier(name) {
						bindingNames = append(bindingNames, name.AsIdentifier().Text)
					}
				}
			case ast.KindImportDeclaration:
				// Skip type-only imports entirely (import type { ... })
				if ast.IsTypeOnlyImportDeclaration(statement) {
					continue
				}
				importClause := statement.AsImportDeclaration().ImportClause
				if importClause != nil {
					// Default import (import Foo from ...)
					if importClause.Name() != nil {
						bindingRanges = append(bindingRanges, importClause.Name().Loc)
						if ast.IsIdentifier(importClause.Name()) {
							bindingNames = append(bindingNames, importClause.Name().AsIdentifier().Text)
						}
					}

					namedBindings := importClause.AsImportClause().NamedBindings
					if namedBindings != nil {
						if ast.IsNamespaceImport(namedBindings) {
							bindingRanges = append(bindingRanges, namedBindings.Name().Loc)
							if ast.IsIdentifier(namedBindings.Name()) {
								bindingNames = append(bindingNames, namedBindings.Name().AsIdentifier().Text)
							}
						} else {
							// Named imports (import { Foo, Bar } from ...)
							for _, element := range namedBindings.Elements() {
								// Skip type-only import specifiers (import { type Foo } from ...)
								if ast.IsPartOfTypeOnlyImportOrExportDeclaration(element) {
									continue
								}
								bindingRanges = append(bindingRanges, element.Name().Loc)
								if ast.IsIdentifier(element.Name()) {
									bindingNames = append(bindingNames, element.Name().AsIdentifier().Text)
								}
							}
						}
					}
				}
			}
		}

		// Populate setupConsts with binding names for template codegen
		for _, name := range bindingNames {
			c.setupConsts[name] = true
		}

		// Hoist type declarations first (emit without mappings to avoid position conflicts)
		for _, typeRange := range typeRanges {
			c.serviceText.WriteString(c.sourceText[typeRange.Pos():typeRange.End()])
			c.serviceText.WriteByte('\n')
		}

		// Now emit the rest of the script, skipping type declarations
		lastMappedPos := text.Loc.Pos()
		for _, statement := range c.scriptSetupEl.Ast.Statements.Nodes {
			stmtStart := innerStart + statement.Loc.Pos()
			stmtEnd := innerStart + statement.Loc.End()

			switch statement.Kind {
			case ast.KindTypeAliasDeclaration, ast.KindInterfaceDeclaration:
				// Skip - already hoisted
				if lastMappedPos < stmtStart {
					c.mapText(lastMappedPos, stmtStart)
				}
				lastMappedPos = stmtEnd
			case ast.KindImportDeclaration:
				if c.scriptEl != nil {
					importRanges = append(importRanges, core.NewTextRange(stmtStart, stmtEnd))
					if lastMappedPos < stmtStart {
						c.mapText(lastMappedPos, stmtStart)
					}
					lastMappedPos = stmtEnd
				}
			}
		}
		c.mapText(lastMappedPos, text.Loc.End())
		c.serviceText.WriteByte('\n')

		// Generate props type if defineProps was found with type argument
		hasPropsType := false
		hasRuntimeProps := false
		if propsInfo != nil && propsInfo.typeArgRange != nil {
			hasPropsType = true
			c.serviceText.WriteString("type __VLS_Props = ")
			// Map the type argument from source
			c.mapText(innerStart+propsInfo.typeArgRange.Pos(), innerStart+propsInfo.typeArgRange.End())
			c.serviceText.WriteString("\n")
		} else if propsInfo != nil && propsInfo.runtimeArgRange != nil {
			// For runtime defineProps, we need to create a props variable
			// This will be typed by Vue's ExtractPropTypes
			hasRuntimeProps = true
			c.serviceText.WriteString("const __VLS_props = defineProps(")
			c.mapText(innerStart+propsInfo.runtimeArgRange.Pos(), innerStart+propsInfo.runtimeArgRange.End())
			c.serviceText.WriteString(");\n")
		}

		if len(bindingRanges) > 0 {
			// Use Vue's ShallowUnwrapRef for automatic ref unwrapping in templates
			c.serviceText.WriteString("type __VLS_SetupExposed = import('vue').ShallowUnwrapRef<{\n")
			for _, binding := range bindingRanges {
				c.serviceText.WriteString(c.sourceText[innerStart+binding.Pos() : innerStart+binding.End()])
				c.serviceText.WriteString(": typeof ")
				c.serviceText.WriteString(c.sourceText[innerStart+binding.Pos() : innerStart+binding.End()])
				c.serviceText.WriteString(";\n")
			}
			c.serviceText.WriteString("}>;\n")
		}

		c.serviceText.WriteString("const __VLS_ctx = {\n")
		c.serviceText.WriteString("...{} as import('vue').ComponentPublicInstance,\n")
		if len(bindingRanges) > 0 {
			c.serviceText.WriteString("...{} as __VLS_SetupExposed,\n")
		}
		// Add props to context
		if hasPropsType {
			c.serviceText.WriteString("...{} as unknown as __VLS_Props,\n")
		} else if hasRuntimeProps {
			c.serviceText.WriteString("...{} as typeof __VLS_props,\n")
		}
		if selfType != "" {
			c.serviceText.WriteString("...{} as unknown as InstanceType<__VLS_PickNotAny<typeof ")
			c.serviceText.WriteString(selfType)
			c.serviceText.WriteString(", new () => {}>>,\n")
		} else {
			c.serviceText.WriteString("...{} as unknown as import('vue').ComponentPublicInstance,\n")
		}
		c.serviceText.WriteString("};\n")

		// Add component/directive/intrinsic type declarations
		if len(bindingRanges) > 0 {
			c.serviceText.WriteString("type __VLS_LocalComponents = __VLS_SetupExposed;\n")
		} else {
			c.serviceText.WriteString("type __VLS_LocalComponents = {};\n")
		}
		c.serviceText.WriteString("type __VLS_GlobalComponents = import('vue').GlobalComponents;\n")
		c.serviceText.WriteString("let __VLS_components!: __VLS_LocalComponents & __VLS_GlobalComponents;\n")
		c.serviceText.WriteString("let __VLS_intrinsics!: import('vue/jsx-runtime').JSX.IntrinsicElements;\n")
		if len(bindingRanges) > 0 {
			c.serviceText.WriteString("type __VLS_LocalDirectives = __VLS_SetupExposed;\n")
		} else {
			c.serviceText.WriteString("type __VLS_LocalDirectives = {};\n")
		}
		c.serviceText.WriteString("let __VLS_directives!: __VLS_LocalDirectives & import('vue').GlobalDirectives;\n")
		c.serviceText.WriteString("type __VLS_StyleScopedClasses = {};\n")

		// When both scripts exist, generate template INSIDE the async IIFE
		// so it can access __VLS_ctx, __VLS_intrinsics, etc.
		if c.scriptEl != nil && c.templateEl != nil {
			generateTemplate(c.codegenCtx, c.templateEl)
		}

		if c.scriptEl != nil {
			c.serviceText.WriteString("\n})()\n")
			for _, loc := range importRanges {
				c.mapText(loc.Pos(), loc.End())
				c.serviceText.WriteString("\n")
			}
		}

		if c.scriptEl == nil {
			c.serviceText.WriteString("export default {} as unknown as Awaited<typeof __VLS_Export>\n")
		}
	}
}
