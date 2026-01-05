package vue_codegen

import (
	"github.com/auvred/golar/internal/collections"
	"github.com/auvred/golar/internal/vue/ast"
	"github.com/auvred/golar/internal/vue/diagnostics"
	"github.com/microsoft/typescript-go/shim/ast"
	"slices"
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

func (c *templateCodegenCtx) shouldPrefixIdentifier(identifier *ast.Node) bool {
	name := identifier.Text()

	for location := identifier; location != nil; location = location.Parent {
		locals := location.Locals()
		if _, ok := locals[name]; ok {
			return false
		}
	}

	for _, scope := range c.scopes {
		if scope.Has(name) {
			return false
		}
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
				c.serviceText.WriteString("{\nconst [")
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
				c.serviceText.WriteString("] = __VLS_vFor(")
				c.mapExpressionInNonBindingPosition(forDirective.Source)
				c.serviceText.WriteString(")\n")
			}
			// Generate event handlers
			for _, eventDir := range eventDirectives {
				c.generateEventHandler(eventDir)
			}
			// TODO: Implement proper slot handling
			// For now, skip slot directive processing - just visit children normally
			// Slot content providers (<template #slotName> inside components) don't need special codegen
			// Slot prop receivers need more complex handling with component type inference
			_ = slotDirective
			c.visit(elem)
			if forDirective != nil {
				c.exitScope()
				c.serviceText.WriteString("}\n")
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
			m.serviceText.WriteString(" __VLS_Ctx.")
			m.mapTextToNodePos(node.End())
		}
		return false
	case ast.KindShorthandPropertyAssignment:
		name := node.Name()
		if m.shouldPrefixIdentifier(name) {
			m.mapTextToNodePos(node.Pos())
			m.serviceText.WriteString(name.Text())
			m.serviceText.WriteString(": __VLS_Ctx.")
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

	// Extract slot from context: const { slotName: __VLS_slot } = __VLS_Ctx.slots!
	c.serviceText.WriteString("const { ")
	c.serviceText.WriteString(slotName)
	c.serviceText.WriteString(": __VLS_slot } = __VLS_Ctx.slots!\n")

	// Generate slot props binding if expression exists
	c.enterScope()
	if dir.Expression != nil && dir.Expression.Ast != nil {
		// Parse slot props expression and declare bindings
		// The expression is parsed as (props) => {} so we use mapExpressionInBindingPosition
		// which already handles extracting the parameter from an arrow function
		c.serviceText.WriteString("const [")
		c.mapExpressionInBindingPosition(dir.Expression)
		c.serviceText.WriteString("] = __VLS_vSlot(__VLS_slot!)\n")
	}

	// Visit children within slot scope
	c.visit(elem)

	c.exitScope()
	c.serviceText.WriteString("}\n")
}
