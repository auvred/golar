package vue_codegen

import (
	"slices"
	"strings"

	"github.com/auvred/golar/internal/collections"
	"github.com/auvred/golar/internal/vue/ast"
	"github.com/auvred/golar/internal/vue/diagnostics"
	"github.com/microsoft/typescript-go/shim/ast"
)

type templateCodegenCtx struct {
	*codegenCtx
	scopes []collections.Set[string]
}

func newTemplateCodegenCtx(base *codegenCtx) templateCodegenCtx {
	return templateCodegenCtx{
		codegenCtx: base,
	}
}

func generateTemplate(base *codegenCtx, el *vue_ast.ElementNode) {
	c := newTemplateCodegenCtx(base)
	if el != nil {
		c.visit(el)
	}
}

func (c *templateCodegenCtx) enterScope() {
	c.scopes = append(c.scopes, collections.Set[string]{})
}
func (c *templateCodegenCtx) exitScope() {
	if len(c.scopes) > 0 {
		c.scopes = c.scopes[:len(c.scopes)-1]
	}
}
func (c *templateCodegenCtx) declareScopeVar(name string) {
	if len(c.scopes) > 0 {
		c.scopes[len(c.scopes)-1].Add(name)
	}
}

// globalIdentifiers contains JavaScript global names that should NOT be prefixed with __VLS_ctx.
// This matches Vue's GLOBALS_ALLOWED from @vue/shared:
// https://github.com/vuejs/core/blob/main/packages/shared/src/globalsAllowed.ts
var globalIdentifiers = makeGlobalIdentifiers()

// literalWhitelist matches Vue's isLiteralWhitelisted
// https://github.com/vuejs/core/blob/main/packages/compiler-core/src/transforms/transformExpression.ts
var literalWhitelist = makeLiteralWhitelist()

func makeGlobalIdentifiers() collections.Set[string] {
	// Exact match of Vue's GLOBALS_ALLOWED
	globals := strings.Split(
		"Infinity,undefined,NaN,isFinite,isNaN,parseFloat,parseInt,decodeURI,decodeURIComponent,encodeURI,encodeURIComponent,Math,Number,Date,Array,Object,Boolean,String,RegExp,Map,Set,JSON,Intl,BigInt,console,Error,Symbol",
		",",
	)
	set := collections.Set[string]{}
	for _, g := range globals {
		set.Add(g)
	}
	return set
}

func makeLiteralWhitelist() collections.Set[string] {
	// Matches Vue's isLiteralWhitelisted: 'true,false,null,this'
	literals := []string{"true", "false", "null", "this"}
	set := collections.Set[string]{}
	for _, l := range literals {
		set.Add(l)
	}
	return set
}

func (c *templateCodegenCtx) shouldPrefixIdentifier(identifier *ast.Node) bool {
	name := identifier.Text()

	// Check template scopes first (v-for variables, slot props, etc.)
	for _, scope := range c.scopes {
		if scope.Has(name) {
			return false
		}
	}

	// Check local scope from TypeScript AST (function params, const declarations, etc.)
	for location := identifier; location != nil; location = location.Parent {
		locals := location.Locals()
		if _, ok := locals[name]; ok {
			return false
		}
	}

	// Check if it's a globally allowed identifier (Vue's GLOBALS_ALLOWED)
	if globalIdentifiers.Has(name) {
		return false
	}

	// Check literal whitelist (true, false, null, this)
	if literalWhitelist.Has(name) {
		return false
	}

	// Check for require (special case in Volar)
	if name == "require" {
		return false
	}

	// Check for __VLS_ internal variables
	if strings.HasPrefix(name, "__VLS_") {
		return false
	}

	return true
}

type conditionalChain uint8

const (
	conditionalChainNone conditionalChain = iota
	conditionalChainValid
	conditionalChainBroken
)

func (c *templateCodegenCtx) visit(el *vue_ast.ElementNode) {
	condChain := conditionalChainNone
	for _, child := range el.Children {
		switch child.Kind {
		case vue_ast.KindElement:
			elem := child.AsElement()

			var conditionalDirective *vue_ast.DirectiveNode
			var forDirective *vue_ast.ForParseResult
			var eventDirectives []*vue_ast.DirectiveNode
			var slotDirective *vue_ast.DirectiveNode
			var seenProps collections.Set[string]
			hasSeenConditionalDirective := false

			for _, p := range elem.Props {
				if p.Kind != vue_ast.KindDirective {
					attr := p.AsAttribute()
					if seenProps.Has(attr.Name) {
						c.reportDiagnostic(attr.NameLoc, vue_diagnostics.Elements_cannot_have_multiple_X_0_with_the_same_name, "attributes")
					} else {
						seenProps.Add(attr.Name)
					}
					continue
				}
				dir := p.AsDirective()
				if seenProps.Has(dir.RawName) {
					c.reportDiagnostic(dir.NameLoc, vue_diagnostics.Elements_cannot_have_multiple_X_0_with_the_same_name, "directives")
					continue
				} else {
					seenProps.Add(dir.RawName)
				}
				switch dir.Name {
				case "if":
					if hasSeenConditionalDirective {
						c.reportDiagnostic(dir.NameLoc, vue_diagnostics.Multiple_conditional_directives_cannot_coexist_on_the_same_element)
						break
					}
					hasSeenConditionalDirective = true
					condChain = conditionalChainValid
					conditionalDirective = dir
				case "else-if":
					if hasSeenConditionalDirective {
						c.reportDiagnostic(dir.NameLoc, vue_diagnostics.Multiple_conditional_directives_cannot_coexist_on_the_same_element)
						break
					}
					hasSeenConditionalDirective = true
					switch condChain {
					case conditionalChainNone:
						c.reportDiagnostic(dir.NameLoc, vue_diagnostics.X_0_has_no_adjacent_v_if_or_v_else_if, "v-else-if")
						condChain = conditionalChainBroken
					case conditionalChainValid:
						conditionalDirective = dir
					}
				case "else":
					if hasSeenConditionalDirective {
						c.reportDiagnostic(dir.NameLoc, vue_diagnostics.Multiple_conditional_directives_cannot_coexist_on_the_same_element)
						break
					}
					hasSeenConditionalDirective = true
					switch condChain {
					case conditionalChainNone:
						c.reportDiagnostic(dir.NameLoc, vue_diagnostics.X_0_has_no_adjacent_v_if_or_v_else_if, "v-else")
					case conditionalChainValid:
						condChain = conditionalChainNone
						conditionalDirective = dir
					}
				case "for":
					forDirective = dir.ForParseResult
				case "on":
					eventDirectives = append(eventDirectives, dir)
				case "slot":
					slotDirective = dir
				}
			}
			if conditionalDirective != nil {
				switch conditionalDirective.Name {
				case "else-if":
					c.serviceText.WriteString("else ")
					fallthrough
				case "if":
					c.serviceText.WriteString("if (")
					if conditionalDirective.Expression != nil && conditionalDirective.Expression.Ast != nil {
						c.mapExpressionInNonBindingPosition(conditionalDirective.Expression)
					} else {
						c.reportDiagnostic(conditionalDirective.Loc, vue_diagnostics.X_0_is_missing_expression, conditionalDirective.RawName)
						c.serviceText.WriteString("1 as number")
					}
					c.serviceText.WriteString(") {\n")
				case "else":
					c.serviceText.WriteString("else {\n")
				}
			} else if !hasSeenConditionalDirective {
				condChain = conditionalChainNone
			}
			if forDirective != nil {
				c.enterScope()
				// Use for...of loop to iterate over __VLS_vFor results
				// e.g., for (const [item, key, index] of __VLS_vFor(source)) { ... }
				c.serviceText.WriteString("for (const [")
				if forDirective.Value != nil {
					c.mapExpressionInBindingPosition(forDirective.Value)
				}
				c.serviceText.WriteString(",")
				if forDirective.Key != nil {
					c.mapExpressionInBindingPosition(forDirective.Key)
				}
				c.serviceText.WriteString(",")
				if forDirective.Index != nil {
					c.mapExpressionInBindingPosition(forDirective.Index)
				}
				c.serviceText.WriteString("] of __VLS_vFor(")
				c.mapExpressionInNonBindingPosition(forDirective.Source)
				c.serviceText.WriteString(")) {\n")
			}
			// Generate element type checking call
			// Skip template elements (used for v-if/v-for grouping)
			if elem.Tag != "template" {
				c.generateElementCall(elem)
			}

			// Generate event handlers
			for _, eventDir := range eventDirectives {
				c.generateEventHandler(eventDir)
			}
			// Handle slot directives (v-slot, #slotName)
			if slotDirective != nil && slotDirective.Expression != nil {
				// Slot with props binding (e.g., #default="{ field }")
				c.generateSlot(slotDirective, elem)
			} else {
				c.visit(elem)
			}
			if forDirective != nil {
				c.exitScope()
				c.serviceText.WriteString("}\n") // Close the for loop
			}
			if conditionalDirective != nil {
				c.serviceText.WriteString("}\n")
			}
		case vue_ast.KindInterpolation:
			interpolation := child.AsInterpolation()
			c.serviceText.WriteString(";( ")
			c.mapExpressionInNonBindingPosition(interpolation.Content)
			// Use ";\n" instead of just "\n" to prevent ASI issues when followed by a block
			// e.g., ";(expr)" followed by "{" would be parsed as arrow function without "=>"
			c.serviceText.WriteString(" );\n")
		}
	}
}

type expressionMapper struct {
	*templateCodegenCtx
	expr          *vue_ast.SimpleExpressionNode
	innerStart    int
	lastMappedPos int
	typeOnly      bool
}

func newExpressionMapper(c *templateCodegenCtx, expr *vue_ast.SimpleExpressionNode) expressionMapper {
	return expressionMapper{
		templateCodegenCtx: c,
		expr:               expr,
		innerStart:         expr.Loc.Pos() - expr.PrefixLen,
		lastMappedPos:      expr.Loc.Pos(),
	}
}

func (m *expressionMapper) mapTextToNodePos(pos int) {
	pos += m.innerStart
	m.mapText(m.lastMappedPos, pos)
	m.lastMappedPos = pos
}

func (m *expressionMapper) shouldPrefixIdentifier(identifier *ast.Node) bool {
	if m.typeOnly {
		return false
	}
	return m.templateCodegenCtx.shouldPrefixIdentifier(identifier)
}

func (c *templateCodegenCtx) mapExpressionInNonBindingPosition(expr *vue_ast.SimpleExpressionNode) {
	m := newExpressionMapper(c, expr)
	if len(expr.Ast.Statements.Nodes) > 0 {
		firstStmt := expr.Ast.Statements.Nodes[0]
		// TODO: report non-binding cases
		if ast.IsExpressionStatement(firstStmt) {
			expr := firstStmt.AsExpressionStatement().Expression
			if ast.IsParenthesizedExpression(expr) {
				m.mapInNonBindingPosition(expr.AsParenthesizedExpression().Expression)
			}
		}
	}
	m.mapTextToNodePos(expr.Ast.End() - expr.SuffixLen)
}
func (c *templateCodegenCtx) mapExpressionInBindingPosition(expr *vue_ast.SimpleExpressionNode) {
	m := newExpressionMapper(c, expr)
	if len(expr.Ast.Statements.Nodes) > 0 {
		firstStmt := expr.Ast.Statements.Nodes[0]
		// TODO: report non-binding cases
		if ast.IsExpressionStatement(firstStmt) {
			expr := firstStmt.AsExpressionStatement().Expression
			if ast.IsArrowFunction(expr) {
				fn := expr.AsArrowFunction()
				if len(fn.Parameters.Nodes) == 1 && ast.IsParameter(fn.Parameters.Nodes[0]) {
					m.mapInBindingPosition(fn.Parameters.Nodes[0].AsParameterDeclaration().Name())
				}
			}
		}
	}
	m.mapTextToNodePos(expr.Ast.End() - expr.SuffixLen)
}

func (m *expressionMapper) mapInBindingPosition(node *ast.BindingName) bool {
	switch node.Kind {
	case ast.KindIdentifier:
		m.declareScopeVar(node.AsIdentifier().Text)
	case ast.KindArrayBindingPattern, ast.KindObjectBindingPattern:
		for _, elem := range node.AsBindingPattern().Elements.Nodes {
			bindingElem := elem.AsBindingElement()
			if visit(m.mapInNonBindingPositionIfNotIdentifier, bindingElem.PropertyName) ||
				visit(m.mapInBindingPosition, bindingElem.Name()) ||
				visit(m.mapInNonBindingPosition, bindingElem.Initializer) {
				return true
			}
		}
	}
	return false
}

func visit(v ast.Visitor, node *ast.Node) bool {
	if node != nil {
		return v(node)
	}
	return false
}
func visitNodeList(v ast.Visitor, nodeList *ast.NodeList) bool {
	if nodeList == nil {
		return false
	}
	return slices.ContainsFunc(nodeList.Nodes, v)
}

func (m *expressionMapper) withTypeOnlyVisit(fn func() bool) bool {
	before := m.typeOnly
	m.typeOnly = true
	res := fn()
	m.typeOnly = before
	return res
}
func (m *expressionMapper) typeOnlyVisit(node *ast.Node) bool {
	return m.withTypeOnlyVisit(func() bool {
		return visit(m.mapInNonBindingPosition, node)
	})
}
func (m *expressionMapper) valueOnlyVisit(node *ast.Node) bool {
	before := m.typeOnly
	m.typeOnly = false
	res := visit(m.mapInNonBindingPosition, node)
	m.typeOnly = before
	return res
}
func (m *expressionMapper) typeOnlyNodeListVisit(nodeList *ast.NodeList) bool {
	if nodeList == nil {
		return false
	}
	return m.withTypeOnlyVisit(func() bool {
		for _, n := range nodeList.Nodes {
			if visit(m.mapInNonBindingPosition, n) {
				return true
			}
		}
		return false
	})
}

func (m *expressionMapper) mapInNonBindingPositionIfNotIdentifier(node *ast.Node) bool {
	return !ast.IsIdentifier(node) && m.mapInNonBindingPosition(node)
}

func (m *expressionMapper) mapInNonBindingPosition(node *ast.Node) bool {
	switch node.Kind {
	case ast.KindIdentifier:
		if m.shouldPrefixIdentifier(node) {
			m.mapTextToNodePos(node.Pos())
			m.serviceText.WriteString(" __VLS_ctx.")
			m.mapTextToNodePos(node.End())
		}
		return false
	case ast.KindShorthandPropertyAssignment:
		name := node.Name()
		if m.shouldPrefixIdentifier(name) {
			m.mapTextToNodePos(node.Pos())
			m.serviceText.WriteString(name.Text())
			m.serviceText.WriteString(": __VLS_ctx.")
			m.mapTextToNodePos(node.End())
		}
		return false
	case ast.KindPropertyAccessExpression:
		n := node.AsPropertyAccessExpression()
		return visit(m.mapInNonBindingPosition, n.Expression) || visit(m.mapInNonBindingPositionIfNotIdentifier, n.Name())
	case ast.KindQualifiedName:
		n := node.AsQualifiedName()
		return visit(m.mapInNonBindingPosition, n.Left) || visit(m.mapInNonBindingPositionIfNotIdentifier, n.Right)
	case ast.KindEnumMember:
		n := node.AsEnumMember()
		return visit(m.mapInNonBindingPositionIfNotIdentifier, n.Name()) || visit(m.mapInNonBindingPosition, n.Initializer)
	case ast.KindPropertyDeclaration:
		n := node.AsPropertyDeclaration()
		return visit(m.mapInNonBindingPositionIfNotIdentifier, n.Name()) || m.typeOnlyVisit(n.Type) || visit(m.mapInNonBindingPosition, n.Initializer)
	case ast.KindPropertyAssignment:
		n := node.AsPropertyAssignment()
		return visit(m.mapInNonBindingPositionIfNotIdentifier, n.Name()) || visit(m.mapInNonBindingPosition, n.Initializer)
	case ast.KindGetAccessor:
		n := node.AsGetAccessorDeclaration()
		return visit(m.mapInNonBindingPositionIfNotIdentifier, n.Name()) || m.typeOnlyNodeListVisit(n.TypeParameters) || visitNodeList(m.mapInNonBindingPosition, n.Parameters) || m.typeOnlyVisit(n.Type) || m.typeOnlyVisit(n.FullSignature) || visit(m.mapInNonBindingPosition, n.Body)
	case ast.KindSetAccessor:
		n := node.AsSetAccessorDeclaration()
		return visit(m.mapInNonBindingPositionIfNotIdentifier, n.Name()) || m.typeOnlyNodeListVisit(n.TypeParameters) || visitNodeList(m.mapInNonBindingPosition, n.Parameters) || m.typeOnlyVisit(n.Type) || m.typeOnlyVisit(n.FullSignature) || visit(m.mapInNonBindingPosition, n.Body)
	case ast.KindVariableDeclaration:
		decl := node.AsVariableDeclaration()
		return visit(m.mapInBindingPosition, decl.Name()) || m.typeOnlyVisit(decl.Type) || visit(m.mapInNonBindingPosition, decl.Initializer)
	case ast.KindBreakStatement,
		ast.KindContinueStatement,
		ast.KindLabeledStatement,
		ast.KindModuleDeclaration:
		return false
	case ast.KindFunctionDeclaration:
		n := node.AsFunctionDeclaration()
		return m.typeOnlyNodeListVisit(n.TypeParameters) || visitNodeList(m.mapInNonBindingPosition, n.Parameters) || m.typeOnlyVisit(n.Type) || m.typeOnlyVisit(n.FullSignature) || visit(m.mapInNonBindingPosition, n.Body)
	case ast.KindArrowFunction:
		n := node.AsArrowFunction()
		return m.typeOnlyNodeListVisit(n.TypeParameters) || visitNodeList(m.mapInNonBindingPosition, n.Parameters) || m.typeOnlyVisit(n.Type) || m.typeOnlyVisit(n.FullSignature) || visit(m.mapInNonBindingPosition, n.Body)
	case ast.KindFunctionExpression:
		n := node.AsFunctionExpression()
		return m.typeOnlyNodeListVisit(n.TypeParameters) || visitNodeList(m.mapInNonBindingPosition, n.Parameters) || m.typeOnlyVisit(n.Type) || m.typeOnlyVisit(n.FullSignature) || visit(m.mapInNonBindingPosition, n.Body)
	case ast.KindClassDeclaration:
		n := node.ClassLikeData()
		return m.typeOnlyNodeListVisit(n.TypeParameters) || visitNodeList(m.mapInNonBindingPosition, n.HeritageClauses) || visitNodeList(m.mapInNonBindingPosition, n.Members)
	case ast.KindConstructor:
		n := node.AsConstructorDeclaration()
		return m.typeOnlyNodeListVisit(n.TypeParameters) || visitNodeList(m.mapInNonBindingPosition, n.Parameters) || m.typeOnlyVisit(n.Type) || m.typeOnlyVisit(n.FullSignature) || visit(m.mapInNonBindingPosition, n.Body)
	case ast.KindMethodDeclaration:
		n := node.AsMethodDeclaration()
		return visit(m.mapInNonBindingPositionIfNotIdentifier, n.Name()) || m.typeOnlyNodeListVisit(n.TypeParameters) || visitNodeList(m.mapInNonBindingPosition, n.Parameters) || m.typeOnlyVisit(n.Type) || m.typeOnlyVisit(n.FullSignature) || visit(m.mapInNonBindingPosition, n.Body)
	case ast.KindHeritageClause:
		n := node.AsHeritageClause()
		if n.Token == ast.KindImplementsKeyword {
			return m.withTypeOnlyVisit(func() bool {
				return node.ForEachChild(m.mapInNonBindingPosition)
			})
		}
	case ast.KindExpressionWithTypeArguments:
		n := node.AsExpressionWithTypeArguments()
		return visit(m.mapInNonBindingPosition, n.Expression) || m.typeOnlyNodeListVisit(n.TypeArguments)
	case ast.KindParameter:
		n := node.AsParameterDeclaration()
		return visit(m.mapInNonBindingPosition, n.Name()) || m.typeOnlyVisit(n.Type) || visit(m.mapInNonBindingPosition, n.Initializer)
	case ast.KindAsExpression:
		n := node.AsAsExpression()
		return visit(m.mapInNonBindingPosition, n.Expression) || m.typeOnlyVisit(n.Type)
	case ast.KindCallExpression:
		n := node.AsCallExpression()
		return visit(m.mapInNonBindingPosition, n.Expression) || m.typeOnlyNodeListVisit(n.TypeArguments) || visitNodeList(m.mapInNonBindingPosition, n.Arguments)
	case ast.KindTypeQuery:
		n := node.AsTypeQueryNode()
		return m.valueOnlyVisit(n.ExprName) || m.typeOnlyNodeListVisit(n.TypeArguments)
	case ast.KindTypeAliasDeclaration, ast.KindInterfaceDeclaration:
		return m.withTypeOnlyVisit(func() bool {
			return node.ForEachChild(m.mapInNonBindingPosition)
		})
	}
	// TODO: JSX

	return node.ForEachChild(m.mapInNonBindingPosition)
}

// generateEventHandler generates TypeScript code for v-on directives.
// Based on Volar's approach:
// - Compound expressions (inline statements like "count++") are wrapped in arrow functions with $event
// - Simple expressions (function references like "handleClick") are used directly
func (c *templateCodegenCtx) generateEventHandler(dir *vue_ast.DirectiveNode) {
	if dir.Expression == nil || dir.Expression.Ast == nil {
		// No expression, emit empty handler
		c.serviceText.WriteString(";(() => {})\n")
		return
	}

	// Determine if the expression is compound (inline statement) or simple (function reference)
	isCompound := c.isCompoundExpression(dir.Expression.Ast)

	if isCompound {
		// Compound expression: wrap in arrow function with $event
		// e.g., @click="count++" -> ;((...[$event]: [any]) => { count++ })
		// Using [any] to allow $event to be passed to any event handler
		// TODO: Infer proper event type from element/component event type
		c.serviceText.WriteString(";((...[$event]: [any]) => {\n")
		c.enterScope()
		c.declareScopeVar("$event")
		c.mapExpressionInNonBindingPosition(dir.Expression)
		c.exitScope()
		c.serviceText.WriteString("\n})\n")
	} else {
		// Simple expression: use directly
		// e.g., @click="handleClick" -> ;(handleClick)
		c.serviceText.WriteString(";(")
		c.mapExpressionInNonBindingPosition(dir.Expression)
		c.serviceText.WriteString(")\n")
	}
}

// isCompoundExpression determines if an expression is a compound statement
// (requires wrapping in arrow function) or a simple function reference.
// Based on Volar's isCompoundExpression logic.
func (c *templateCodegenCtx) isCompoundExpression(file *ast.SourceFile) bool {
	if len(file.Statements.Nodes) == 0 {
		return false
	}
	if len(file.Statements.Nodes) > 1 {
		return true
	}

	// Check the single statement
	stmt := file.Statements.Nodes[0]
	if !ast.IsExpressionStatement(stmt) {
		return true
	}

	expr := stmt.AsExpressionStatement().Expression
	// Unwrap parentheses (our parsed expression is wrapped in parens)
	if ast.IsParenthesizedExpression(expr) {
		expr = expr.AsParenthesizedExpression().Expression
	}

	// Arrow function or function expression is not compound
	if ast.IsArrowFunction(expr) || ast.IsFunctionExpression(expr) {
		return false
	}

	// Simple identifier or property access chain is not compound
	if c.isPropertyAccessOrIdentifier(expr) {
		return false
	}

	return true
}

// isPropertyAccessOrIdentifier checks if expression is an identifier
// or a chain of property accesses (e.g., obj.method, a.b.c)
func (c *templateCodegenCtx) isPropertyAccessOrIdentifier(node *ast.Node) bool {
	if ast.IsIdentifier(node) {
		return true
	}
	if ast.IsPropertyAccessExpression(node) {
		return c.isPropertyAccessOrIdentifier(node.AsPropertyAccessExpression().Expression)
	}
	return false
}

// generateSlot generates TypeScript code for v-slot directives.
// Based on Volar's approach:
// - Extracts slot from __VLS_Ctx.slots
// - Uses __VLS_vSlot helper to extract typed slot props
// - Creates scope for slot props bindings
//
// For example: <template #default="{ item }">
// Generates:
//
//	{
//	const { default: __VLS_slot } = __VLS_Ctx.slots!
//	const [{ item }] = __VLS_vSlot(__VLS_slot!)
//	// ... children
//	}
func (c *templateCodegenCtx) generateSlot(dir *vue_ast.DirectiveNode, elem *vue_ast.ElementNode) {
	c.serviceText.WriteString("{\n")

	// Determine slot name from directive argument
	slotName := "default"
	if dir.Arg != nil && dir.ArgIsStatic {
		// Static slot name from argument
		slotName = c.sourceText[dir.Arg.Loc.Pos():dir.Arg.Loc.End()]
	}

	// Extract slot from context: const { slotName: __VLS_slot } = __VLS_ctx.slots!
	// Quote slot name if it contains special characters like hyphens
	c.serviceText.WriteString("const { ")
	if needsQuotes(slotName) {
		c.serviceText.WriteString("\"")
		c.serviceText.WriteString(slotName)
		c.serviceText.WriteString("\"")
	} else {
		c.serviceText.WriteString(slotName)
	}
	c.serviceText.WriteString(": __VLS_slot } = __VLS_ctx.slots!\n")

	// Generate slot props binding if expression exists
	c.enterScope()
	if dir.Expression != nil && dir.Expression.Ast != nil {
		// Parse slot props expression and declare bindings
		// Following Volar's approach for typed slots:
		// - Extract binding pattern without type annotation
		// - If type exists, pass it as callback: __VLS_vSlot(slot!, (_: Type) => [] as any)
		c.generateSlotParameters(dir.Expression)
	}

	// Visit children within slot scope
	c.visit(elem)

	c.exitScope()
	c.serviceText.WriteString("}\n")
}

// generateSlotParameters generates slot props binding following Volar's approach.
// For typed slots like `v-slot="{ item }: { item: Type }"`:
// - Output: const [{ item }] = __VLS_vSlot(__VLS_slot!, (_: { item: Type }, ) => [] as any)
// For untyped slots like `v-slot="{ item }"`:
// - Output: const [{ item }] = __VLS_vSlot(__VLS_slot!)
func (c *templateCodegenCtx) generateSlotParameters(expr *vue_ast.SimpleExpressionNode) {
	// Try to extract typed slot parameters following Volar's approach
	if len(expr.Ast.Statements.Nodes) > 0 {
		firstStmt := expr.Ast.Statements.Nodes[0]
		if ast.IsExpressionStatement(firstStmt) {
			exprNode := firstStmt.AsExpressionStatement().Expression
			if ast.IsArrowFunction(exprNode) {
				fn := exprNode.AsArrowFunction()
				if len(fn.Parameters.Nodes) > 0 && ast.IsParameter(fn.Parameters.Nodes[0]) {
					param := fn.Parameters.Nodes[0].AsParameterDeclaration()
					bindingName := param.Name()
					paramType := param.Type // Type annotation, if present

					// Output: const [binding] = __VLS_vSlot(__VLS_slot!, optionalTypeCallback)
					c.serviceText.WriteString("const [")

					// Map the binding pattern and declare scope variables
					m := newExpressionMapper(c, expr)
					m.mapInBindingPosition(bindingName)
					// End at the binding name, not including type annotation
					m.mapTextToNodePos(bindingName.End())

					c.serviceText.WriteString("] = __VLS_vSlot(__VLS_slot!")

					// If there's a type annotation, add the callback pattern
					if paramType != nil {
						c.serviceText.WriteString(", (_")
						// Extract type annotation text from source
						// AST positions are relative to the parsed "(expr) => {}" string
						// We need to map back to source positions
						typeStartInAst := bindingName.End()
						typeEndInAst := paramType.End()
						// The AST text starts at position expr.PrefixLen (which is 1 for the opening paren)
						// So source position = expr.Loc.Pos() + (astPos - expr.PrefixLen)
						sourceStart := expr.Loc.Pos() + (typeStartInAst - expr.PrefixLen)
						sourceEnd := expr.Loc.Pos() + (typeEndInAst - expr.PrefixLen)
						if sourceStart >= 0 && sourceEnd <= len(c.sourceText) && sourceStart < sourceEnd {
							typeText := c.sourceText[sourceStart:sourceEnd]
							c.serviceText.WriteString(typeText)
						}
						c.serviceText.WriteString(", ) => [] as any")
					}

					c.serviceText.WriteString(")\n")
					return
				}
			}
		}
	}

	// Fallback: use the original approach for non-standard cases
	c.serviceText.WriteString("const [")
	c.mapExpressionInBindingPosition(expr)
	c.serviceText.WriteString("] = __VLS_vSlot(__VLS_slot!)\n")
}

// writeIntrinsicAccess writes access to __VLS_intrinsics for a tag name.
// Uses dot notation for valid identifiers, bracket notation for tags with hyphens.
// e.g., "div" -> __VLS_intrinsics.div
// e.g., "f-switch" -> __VLS_intrinsics["f-switch"]
func (c *templateCodegenCtx) writeIntrinsicAccess(tag string) {
	c.serviceText.WriteString("__VLS_intrinsics")
	if needsQuotes(tag) {
		c.serviceText.WriteString("[\"")
		c.serviceText.WriteString(tag)
		c.serviceText.WriteString("\"]")
	} else {
		c.serviceText.WriteString(".")
		c.serviceText.WriteString(tag)
	}
}

// generateElementCall generates a __VLS_asFunctionalElement1 call for intrinsic HTML elements.
// This provides type checking for element props and enables proper TypeScript type inference.
// Based on Volar's element.ts generateElement function.
func (c *templateCodegenCtx) generateElementCall(elem *vue_ast.ElementNode) {
	tag := elem.Tag

	// Generate: __VLS_asFunctionalElement1(__VLS_intrinsics.TAG, __VLS_intrinsics.TAG)({...props...});
	// For tags with hyphens (like "f-switch"), use bracket notation: __VLS_intrinsics["f-switch"]
	c.serviceText.WriteString("__VLS_asFunctionalElement1(")
	c.writeIntrinsicAccess(tag)
	c.serviceText.WriteString(", ")
	c.writeIntrinsicAccess(tag)
	c.serviceText.WriteString(")({\n")

	// Generate props
	for _, p := range elem.Props {
		if p.Kind == vue_ast.KindAttribute {
			attr := p.AsAttribute()
			c.generateElementAttribute(attr)
		} else if p.Kind == vue_ast.KindDirective {
			dir := p.AsDirective()
			c.generateElementDirectiveProp(dir)
		}
	}

	c.serviceText.WriteString("});\n")
}

// toCamelCase converts a kebab-case string to camelCase
// e.g., "overlay-click" -> "overlayClick"
// toCamelCase converts a kebab-case string to camelCase.
// Only hyphens trigger capitalization - this matches Vue's camelize() from @vue/shared.
// Colons are preserved (e.g., "update:open" stays as "update:open", not "updateOpen").
// e.g., "overlay-click" -> "overlayClick"
// e.g., "on-overlay-click" -> "onOverlayClick"
// e.g., "on-update:open" -> "onUpdate:open" (colon preserved)
func toCamelCase(s string) string {
	if !strings.Contains(s, "-") {
		return s
	}
	var result strings.Builder
	capitalizeNext := false
	for _, ch := range s {
		if ch == '-' {
			capitalizeNext = true
		} else if capitalizeNext {
			if ch >= 'a' && ch <= 'z' {
				result.WriteRune(ch - 32) // Convert to uppercase
			} else {
				result.WriteRune(ch)
			}
			capitalizeNext = false
		} else {
			result.WriteRune(ch)
		}
	}
	return result.String()
}

// needsQuotes returns true if the property name needs to be quoted in JS
// (contains hyphens, starts with number, has special chars, etc.)
func needsQuotes(name string) bool {
	if len(name) == 0 {
		return true
	}
	// Check first character - must be letter, underscore, or $
	first := name[0]
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_' || first == '$') {
		return true
	}
	// Check rest - must be alphanumeric, underscore, or $
	for i := 1; i < len(name); i++ {
		ch := name[i]
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '$') {
			return true
		}
	}
	return false
}

// writePropertyName writes a property name, quoting it if necessary
func (c *templateCodegenCtx) writePropertyName(name string) {
	if needsQuotes(name) {
		c.serviceText.WriteString("\"")
		c.serviceText.WriteString(name)
		c.serviceText.WriteString("\"")
	} else {
		c.serviceText.WriteString(name)
	}
}

// escapeStringLiteral escapes a string for use in a JS string literal.
// Handles newlines, quotes, and backslashes.
func escapeStringLiteral(s string) string {
	var result strings.Builder
	for _, ch := range s {
		switch ch {
		case '\n':
			result.WriteString("\\n")
		case '\r':
			result.WriteString("\\r")
		case '\t':
			result.WriteString("\\t")
		case '"':
			result.WriteString("\\\"")
		case '\\':
			result.WriteString("\\\\")
		default:
			result.WriteRune(ch)
		}
	}
	return result.String()
}

// generateElementAttribute generates a prop for a static HTML attribute.
// e.g., class="foo" -> ...{ class: "foo" },
// e.g., aria-label="text" -> ...{ "aria-label": "text" },
func (c *templateCodegenCtx) generateElementAttribute(attr *vue_ast.AttributeNode) {
	c.serviceText.WriteString("...{ ")
	c.writePropertyName(attr.Name)
	c.serviceText.WriteString(": ")
	if attr.Value != nil {
		c.serviceText.WriteString("\"")
		c.serviceText.WriteString(escapeStringLiteral(attr.Value.Content))
		c.serviceText.WriteString("\"")
	} else {
		c.serviceText.WriteString("true")
	}
	c.serviceText.WriteString(" },\n")
}

// generateElementDirectiveProp generates a prop for a v-bind or v-on directive.
// e.g., :disabled="isDisabled" -> disabled: (__VLS_ctx.isDisabled),
// e.g., @click="handler" -> ...{ onClick: (handler) },
func (c *templateCodegenCtx) generateElementDirectiveProp(dir *vue_ast.DirectiveNode) {
	switch dir.Name {
	case "bind":
		// v-bind / :attr
		if dir.Arg != nil && dir.ArgIsStatic && dir.Expression != nil {
			argName := c.sourceText[dir.Arg.Loc.Pos():dir.Arg.Loc.End()]
			// Special handling for class/style - use spread
			if argName == "class" || argName == "style" {
				c.serviceText.WriteString("...{ ")
				c.writePropertyName(argName)
				c.serviceText.WriteString(": (")
				c.mapExpressionInNonBindingPosition(dir.Expression)
				c.serviceText.WriteString(") },\n")
			} else {
				c.writePropertyName(argName)
				c.serviceText.WriteString(": (")
				c.mapExpressionInNonBindingPosition(dir.Expression)
				c.serviceText.WriteString("),\n")
			}
		}
	case "on":
		// v-on / @event - generate prop for element type checking
		if dir.Arg != nil && dir.ArgIsStatic && dir.Expression != nil {
			eventName := c.sourceText[dir.Arg.Loc.Pos():dir.Arg.Loc.End()]
			// Convert to onEventName format using camelize (matches Vue's @vue/shared camelize)
			// e.g., "overlay-click" -> "onOverlayClick"
			// e.g., "update:open" -> "onUpdate:open" (colon preserved, will be quoted)
			propName := toCamelCase("on-" + eventName)
			c.serviceText.WriteString("...{ ")
			c.writePropertyName(propName)
			c.serviceText.WriteString(": ")

			// Check if compound expression - needs arrow function wrapper
			isCompound := c.isCompoundExpression(dir.Expression.Ast)
			if isCompound {
				// Compound expression: wrap in arrow function with $event in scope
				// e.g., "count++; foo = bar" -> "(...[$event]: [any]) => { count++; foo = bar }"
				// Using [any] type to avoid implicit any errors
				c.serviceText.WriteString("(...[$event]: [any]) => {\n")
				c.enterScope()
				c.declareScopeVar("$event")
				c.mapExpressionInNonBindingPosition(dir.Expression)
				c.exitScope()
				c.serviceText.WriteString("\n}")
			} else {
				// Simple expression: use directly
				// e.g., "handleClick" -> "(handleClick)"
				c.serviceText.WriteString("(")
				c.mapExpressionInNonBindingPosition(dir.Expression)
				c.serviceText.WriteString(")")
			}
			c.serviceText.WriteString(" },\n")
		}
	}
}
