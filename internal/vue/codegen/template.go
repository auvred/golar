package vue_codegen

import (
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/auvred/golar/internal/collections"
	"github.com/auvred/golar/internal/utils"
	"github.com/auvred/golar/internal/vue/ast"
	"github.com/auvred/golar/internal/vue/diagnostics"
	"github.com/auvred/golar/internal/vue/parser"
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/core"
)

var commentDirectiveRe = regexp.MustCompile(`^\s*@vue-([-a-z]+)(.*)$`)

// jsGlobals is the set of JavaScript global identifiers that Vue allows in template
// expressions without prefixing with the component context.
// Matches Vue core's globalsAllowList: https://github.com/vuejs/core/blob/main/packages/shared/src/globalsAllowList.ts
var jsGlobals = collections.NewSetFromItems(
	"Infinity", "undefined", "NaN",
	"isFinite", "isNaN", "parseFloat", "parseInt",
	"decodeURI", "decodeURIComponent", "encodeURI", "encodeURIComponent",
	"Math", "Number", "Date", "Array", "Object", "Boolean", "String",
	"RegExp", "Map", "Set", "JSON", "Intl", "BigInt",
	"console", "Error", "Symbol",
	"globalThis",
)

type templateCodegenCtx struct {
	*codegenCtx
	scopes             []collections.Set[string]
	parentComponentVar string
	condChain                   conditionalChain
	blockConditions             []string // stack of condition expressions for compound event guards
	blockConditionsChainSaveLen int      // saved blockConditions length at start of current v-if chain

	ignoreError    bool
	expectError    bool
	expectErrorLoc core.TextRange

	// Track identifiers accessed via __VLS_ctx for // @ts-ignore [var,...]; blocks.
	// Volar uses a global componentAccessMap with scope-based drain:
	// - Keys are ordered by first insertion
	// - Each key maps to a count of accesses since last drain
	// - When any scope ends, ALL keys with count > 0 are emitted and counts reset
	// - Keys remain in the map (preserving insertion order for future drains)
	accessMapKeys   []string       // ordered unique keys
	accessMapCounts map[string]int // count of accesses since last drain
	// allAccessedVars tracks ALL vars ever accessed via __VLS_ctx (never cleared),
	// used for filtering __VLS_SetupExposed to only include template-used bindings.
	allAccessedVars []string
	// scopeDepth tracks nested scope count for scope drain behavior
	usedVarScopeDepth int
	// skipUsedVarTracking suppresses trackUsedVar calls (e.g., during constructor prop generation)
	skipUsedVarTracking bool
	// Track slot prop variables for slot type generation
	slotProps []slotPropInfo
	// Track root element tags for __VLS_RootEl type (only for single-root detection)
	rootElementTags []string
	// Number of non-comment root children in template
	rootChildCount int
	depth          int
}

type slotPropInfo struct {
	name     string // slot name (empty = default)
	propsVar string // variable name for slot props
}

func newTemplateCodegenCtx(base *codegenCtx) templateCodegenCtx {
	return templateCodegenCtx{
		codegenCtx: base,
	}
}

func generateTemplate(base *codegenCtx, el *vue_ast.ElementNode) {
	c := newTemplateCodegenCtx(base)
	// Push a top-level scope for tracking used vars
	c.usedVarScopeDepth = 1
	if el != nil {
		// Count non-comment, non-whitespace root children for single-root detection
		for _, child := range el.Children {
			switch child.Kind {
			case vue_ast.KindComment:
				// Skip comments
			case vue_ast.KindText:
				if strings.TrimSpace(child.AsText().Content) != "" {
					c.rootChildCount++
				}
			default:
				c.rootChildCount++
			}
		}
		// Visit children directly, skip <template> wrapper
		for _, child := range el.Children {
			c.visit(child)
		}
		// If root children ended with an unterminated conditional chain, restore blockConditions
		if c.condChain == conditionalChainValid {
			c.blockConditions = c.blockConditions[:c.blockConditionsChainSaveLen]
		}
	}
	// Collect slot props for type generation
	if len(c.slotProps) > 0 {
		c.templateHasSlots = true
		c.generateSlotTypes()
	}
	// Emit __VLS_TemplateRefs type (between Slots and RootEl to match Volar order)
	if len(c.templateRefs) > 0 {
		c.serviceText.WriteString("type __VLS_TemplateRefs = {}\n")
		for _, ref := range c.templateRefs {
			c.serviceText.WriteString("& { ")
			c.serviceText.WriteString(ref.name)
			c.serviceText.WriteString(": __VLS_Elements['")
			c.serviceText.WriteString(ref.elemTag)
			c.serviceText.WriteString("'] };\n")
		}
	}
	// Emit __VLS_RootEl type only for single-root components (Volar skips when multiple root children)
	if len(c.rootElementTags) > 0 && c.rootChildCount == 1 {
		c.serviceText.WriteString("type __VLS_RootEl = \n")
		for _, tag := range c.rootElementTags {
			c.serviceText.WriteString("| __VLS_Elements['")
			c.serviceText.WriteString(tag)
			c.serviceText.WriteString("'];\n")
		}
	}
	// Collect remaining used vars from top-level scope (keys with count > 0)
	for _, key := range c.accessMapKeys {
		count := c.accessMapCounts[key]
		for range count {
			c.usedTemplateVars = append(c.usedTemplateVars, key)
		}
	}
	// Propagate allAccessedVars to codegenCtx
	c.codegenCtx.allAccessedVars = append(c.codegenCtx.allAccessedVars, c.allAccessedVars...)
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

func (c *templateCodegenCtx) trackUsedVar(name string) {
	if c.skipUsedVarTracking {
		return
	}
	// Track in the ordered access map.
	// If this key is new, add it to the key order list.
	if c.accessMapCounts == nil {
		c.accessMapCounts = make(map[string]int)
	}
	if _, exists := c.accessMapCounts[name]; !exists {
		c.accessMapKeys = append(c.accessMapKeys, name)
	}
	c.accessMapCounts[name]++
	// Also track in the never-cleared list for SetupExposed filtering
	c.allAccessedVars = append(c.allAccessedVars, name)
}

func (c *templateCodegenCtx) pushUsedVarScope() {
	c.usedVarScopeDepth++
}

// drainUsedVarScope drains all accumulated vars and emits the // @ts-ignore [var,...]; block.
// Keys are emitted in their original insertion order, once per access since last drain.
func (c *templateCodegenCtx) drainUsedVarScope() {
	c.usedVarScopeDepth--
	// Check if any keys have been accessed since last drain
	hasEntries := false
	for _, key := range c.accessMapKeys {
		if c.accessMapCounts[key] > 0 {
			hasEntries = true
			break
		}
	}
	if !hasEntries {
		return
	}
	c.serviceText.WriteString("// @ts-ignore\n[")
	for _, key := range c.accessMapKeys {
		count := c.accessMapCounts[key]
		for range count {
			c.serviceText.WriteString(key)
			c.serviceText.WriteString(",")
		}
		c.accessMapCounts[key] = 0 // Reset count, but keep key in order
	}
	c.serviceText.WriteString("];\n")
}

func (c *templateCodegenCtx) shouldPrefixIdentifier(identifier *ast.Node) bool {
	name := identifier.Text()

	if jsGlobals.Has(name) {
		return false
	}

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

func (c *templateCodegenCtx) visit(el *vue_ast.Node) {
	switch el.Kind {
	case vue_ast.KindComment:
		if c.expectError {
			c.mapExpectErrorDirective(c.expectErrorLoc.Pos(), c.expectErrorLoc.End(), 0, 0)
		}
		c.ignoreError = false
		c.expectError = false
		comm := el.AsComment()
		m := commentDirectiveRe.FindStringSubmatch(comm.Content)
		if m == nil {
			return
		}
		switch m[1] {
		case "ignore":
			c.ignoreError = true
		case "expect-error":
			c.expectError = true
			c.expectErrorLoc = comm.Loc
		}
		return
	case vue_ast.KindText:
		text := el.AsText()
		if strings.TrimSpace(text.Content) == "" {
			return
		}
		if c.expectError {
			c.mapExpectErrorDirective(c.expectErrorLoc.Pos(), c.expectErrorLoc.End(), 0, 0)
		}
	case vue_ast.KindElement:
		elementServiceTextStart := c.serviceText.Len()
		elem := el.AsElement()

		var conditionalDirective *vue_ast.DirectiveNode
		var forDirective *vue_ast.ForParseResult
		var slotDirective *vue_ast.DirectiveNode
		var seenProps collections.Set[string]
		hasSeenConditionalDirective := false
		prevCondChain := c.condChain

		// TODO: unexpected props and directives, for example on <template>
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
				c.condChain = conditionalChainValid
				conditionalDirective = dir
			case "else-if":
				if hasSeenConditionalDirective {
					c.reportDiagnostic(dir.NameLoc, vue_diagnostics.Multiple_conditional_directives_cannot_coexist_on_the_same_element)
					break
				}
				hasSeenConditionalDirective = true
				switch c.condChain {
				case conditionalChainNone:
					c.reportDiagnostic(dir.NameLoc, vue_diagnostics.X_0_has_no_adjacent_v_if_or_v_else_if, "v-else-if")
					c.condChain = conditionalChainBroken
				case conditionalChainValid:
					conditionalDirective = dir
				}
			case "else":
				if hasSeenConditionalDirective {
					c.reportDiagnostic(dir.NameLoc, vue_diagnostics.Multiple_conditional_directives_cannot_coexist_on_the_same_element)
					break
				}
				hasSeenConditionalDirective = true
				switch c.condChain {
				case conditionalChainNone:
					c.reportDiagnostic(dir.NameLoc, vue_diagnostics.X_0_has_no_adjacent_v_if_or_v_else_if, "v-else")
				case conditionalChainValid:
					c.condChain = conditionalChainNone
					conditionalDirective = dir
				}
			case "for":
				forDirective = dir.ForParseResult
			// TODO: #slot
			case "slot":
				slotDirective = dir
			}
		}
		// Track blockConditions for condition guards in compound event handlers.
		// Volar saves blockConditions.length at the start of a v-if chain and restores
		// after all branches are processed.
		if conditionalDirective != nil {
			switch conditionalDirective.Name {
			case "else-if":
				// Negate the previous condition for else-if guards
				if len(c.blockConditions) > 0 {
					c.blockConditions[len(c.blockConditions)-1] = "!" + c.blockConditions[len(c.blockConditions)-1]
				}
				c.serviceText.WriteString("else ")
				fallthrough
			case "if":
				if conditionalDirective.Name == "if" {
					// If a previous conditional chain wasn't terminated (no v-else),
					// restore blockConditions before starting a new chain.
					if prevCondChain == conditionalChainValid {
						c.blockConditions = c.blockConditions[:c.blockConditionsChainSaveLen]
					}
					// Save the blockConditions length at chain start
					c.blockConditionsChainSaveLen = len(c.blockConditions)
				}
				c.serviceText.WriteString("if (")
				condStart := c.serviceText.Len()
				if conditionalDirective.Expression != nil && conditionalDirective.Expression.Ast != nil {
					c.mapExpressionInNonBindingPosition(conditionalDirective.Expression)
				} else {
					c.reportDiagnostic(conditionalDirective.Loc, vue_diagnostics.X_0_is_missing_expression, conditionalDirective.RawName)
					c.serviceText.WriteString("1 as number")
				}
				condText := c.serviceText.String()[condStart:]
				c.blockConditions = append(c.blockConditions, "("+condText+")")
				c.serviceText.WriteString(") {\n")
			case "else":
				// Negate the previous condition for else guards
				if len(c.blockConditions) > 0 {
					c.blockConditions[len(c.blockConditions)-1] = "!" + c.blockConditions[len(c.blockConditions)-1]
				}
				c.serviceText.WriteString("else {\n")
			}
		} else if !hasSeenConditionalDirective {
			// A non-conditional element ends any active conditional chain.
			// Restore blockConditions to the saved chain start length.
			if prevCondChain == conditionalChainValid {
				c.blockConditions = c.blockConditions[:c.blockConditionsChainSaveLen]
			}
			c.condChain = conditionalChainNone
		}
		if forDirective != nil {
			c.enterScope()
			c.pushUsedVarScope()
			// Volar uses for...of: for (const [...] of __VLS_vFor((...expr)!)) {
			c.serviceText.WriteString("for (const [")
			if forDirective.Value != nil {
				c.mapExpressionInBindingPosition(forDirective.Value)
			}
			if forDirective.Key != nil {
				c.serviceText.WriteString(",")
				c.mapExpressionInBindingPosition(forDirective.Key)
			}
			if forDirective.Index != nil {
				c.serviceText.WriteString(",")
				c.mapExpressionInBindingPosition(forDirective.Index)
			}
			c.serviceText.WriteString("] of __VLS_vFor((")
			c.mapExpressionInNonBindingPosition(forDirective.Source)
			c.serviceText.WriteString(")!)) {\n")
		}

		// Handle dynamic <component :is="expr">
		var dynamicComponentExpr *vue_ast.DirectiveNode
		if elem.Tag == "component" {
			// Look for :is directive
			for _, prop := range elem.Props {
				if prop.Kind == vue_ast.KindDirective {
					dir := prop.AsDirective()
					if dir.Name == "bind" && dir.Arg == "is" {
						dynamicComponentExpr = dir
						break
					}
				}
			}
		}

		// TODO: handle template
		isComponent := (elem.Tag == "component" && dynamicComponentExpr != nil) ||
			(elem.Tag != "template" && elem.Tag != "component" && elem.Tag != "slot" && (isBuiltInComponent(elem.Tag) || !isNativeElement(elem.Tag)))
		isTemplate := elem.Tag == "template"
		var ctxVar string
		// TODO: don't generate unused vars
		// TODO: expression components like foo.bar
		// TODO: global components
		// TODO: self component
		// TODO: component name casing
		// TODO: native elements
		if isNativeElement(elem.Tag) && !isTemplate {
			// Track root elements for __VLS_RootEl type
			if c.depth == 0 {
				c.rootElementTags = append(c.rootElementTags, elem.Tag)
			}
			// Track ref attributes for __VLS_TemplateRefs type
			c.trackRefAttribute(elem)
			c.serviceText.WriteString("__VLS_asFunctionalElement1(__VLS_intrinsics.")
			c.serviceText.WriteString(elem.Tag)
			c.serviceText.WriteString(", __VLS_intrinsics.")
			c.serviceText.WriteString(elem.Tag)
			c.serviceText.WriteString(")({\n")
			propsStart := c.serviceText.Len() - 2
			c.generateElementProps(elem, false)
			propsEnd := c.serviceText.Len() + 1
			c.serviceText.WriteString("});\n")
			// Generate standalone v-model getter for default v-model (no argument)
			c.generateVModelGetter(elem)
			// Emit scoped CSS class assertion after element
			c.emitScopedClassAssertion(elem)
			// TODO: is this valid?
			tagStart := elem.Loc.Pos() + 1
			c.mapRange(tagStart, tagStart+len(elem.Tag), propsStart, propsEnd)
		} else if isComponent {
			// Volar allocation order: componentVar, functionalVar, vnodeVar, ctxVar, propsVar
			componentVar := c.newInternalVariable()
			functionalVar := c.newInternalVariable()
			vnodeVar := c.newInternalVariable()
			ctxVarName := c.newInternalVariable()
			propsVarName := c.newInternalVariable()
			emitsVar := ""
			isCtxVarUsed := false
			isPropsVarUsed := false

			// Volar: const __VLS_0 = ComponentName; or const __VLS_0 = (expression);
			c.serviceText.WriteString("const ")
			c.serviceText.WriteString(componentVar)
			c.serviceText.WriteString(" = ")
			if dynamicComponentExpr != nil {
				// Dynamic component: wrap expression in parens
				c.serviceText.WriteString("(")
				c.mapExpressionInNonBindingPosition(dynamicComponentExpr.Expression)
				c.serviceText.WriteString(")")
			} else {
				// Static component: use the tag name directly (for imported components)
				c.serviceText.WriteString(elem.Tag)
			}
			c.serviceText.WriteString(";\n")

			// Volar: // @ts-ignore\nconst __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({...}));
			c.serviceText.WriteString("// @ts-ignore\nconst ")
			c.serviceText.WriteString(functionalVar)
			c.serviceText.WriteString(" = __VLS_asFunctionalComponent1(")
			c.serviceText.WriteString(componentVar)
			c.serviceText.WriteString(", new ")
			c.serviceText.WriteString(componentVar)
			c.serviceText.WriteString("({\n")
			// Skip used var tracking for constructor props (Volar generates propCodes once, yields twice)
			c.skipUsedVarTracking = true
			c.generateElementPropsFiltered(elem, true, dynamicComponentExpr)
			c.skipUsedVarTracking = false
			c.serviceText.WriteString("}));\n")

			// Volar: const __VLS_2 = __VLS_1({...}, ...__VLS_functionalComponentArgsRest(__VLS_1));
			c.serviceText.WriteString("const ")
			c.serviceText.WriteString(vnodeVar)
			c.serviceText.WriteString(" = ")
			c.serviceText.WriteString(functionalVar)
			propsStart := c.serviceText.Len() + 1
			c.serviceText.WriteString("({\n")
			c.generateElementPropsFiltered(elem, true, dynamicComponentExpr)
			c.serviceText.WriteString("}, ...__VLS_functionalComponentArgsRest(")
			c.serviceText.WriteString(functionalVar)
			c.serviceText.WriteString("));\n")
			propsEnd := c.serviceText.Len() - 2
			_ = propsStart
			_ = propsEnd
			// TODO: tag mapping
			// tagStart := elem.Loc.Pos() + 1
			// c.mapRange(tagStart, tagStart+len(elem.Tag), propsStart, propsEnd)

			// TODO: emits type mismatch mapping locations
			for _, prop := range elem.Props {
				if prop.Kind != vue_ast.KindDirective {
					continue
				}

				dir := prop.AsDirective()
				// TODO: dynamic event name
				if dir.Name != "on" || !dir.IsStatic {
					continue
				}
				isCtxVarUsed = true
				isPropsVarUsed = true
				if emitsVar == "" {
					emitsVar = c.newInternalVariable()
					c.serviceText.WriteString("var ")
					c.serviceText.WriteString(emitsVar)
					c.serviceText.WriteString("!: __VLS_ResolveEmits<typeof ")
					c.serviceText.WriteString(componentVar)
					c.serviceText.WriteString(", typeof ")
					c.serviceText.WriteString(ctxVarName)
					c.serviceText.WriteString(".emit>\n")
				}

				// TODO: model & vue:
				c.serviceText.WriteString("// @ts-ignore\nconst ")
				c.serviceText.WriteString(c.newInternalVariable())
				c.serviceText.WriteString(": __VLS_NormalizeComponentEvent<typeof ")
				c.serviceText.WriteString(propsVarName)
				c.serviceText.WriteString(", typeof ")
				c.serviceText.WriteString(emitsVar)
				c.serviceText.WriteString(", '")
				camelize("on-"+dir.Arg, &c.serviceText)
				c.serviceText.WriteString("', '")
				emitName := dir.Arg
				c.serviceText.WriteString(emitName)
				c.serviceText.WriteString("', '")
				camelize(emitName, &c.serviceText) // camelizedEmitName
				c.serviceText.WriteString("'> = \n{\n")
				c.generateEventName(dir)
				c.serviceText.WriteString(": ")
				c.generateEventExpression(dir)
				c.serviceText.WriteString("\n}\n")
			}

			// Conditionally emit ctxVar and propsVar (only if used for events/slots)
			if isCtxVarUsed {
				c.serviceText.WriteString("var ")
				c.serviceText.WriteString(ctxVarName)
				c.serviceText.WriteString("!: __VLS_FunctionalComponentCtx<typeof ")
				c.serviceText.WriteString(componentVar)
				c.serviceText.WriteString(", typeof ")
				c.serviceText.WriteString(vnodeVar)
				c.serviceText.WriteString(">\n")
			}

			if isPropsVarUsed {
				c.serviceText.WriteString("var ")
				c.serviceText.WriteString(propsVarName)
				c.serviceText.WriteString("!: __VLS_FunctionalComponentProps<typeof ")
				c.serviceText.WriteString(componentVar)
				c.serviceText.WriteString(", typeof ")
				c.serviceText.WriteString(vnodeVar)
				c.serviceText.WriteString(">\n")
			}

			ctxVar = ctxVarName
		} else if isTemplate {
			// For <template> elements with directives (v-if, v-for, etc.),
			// we don't emit element function call — just process children
		}

		// Slot directive handling
		if slotDirective != nil {
			parentComponentCtx := ctxVar
			if parentComponentCtx == "" {
				parentComponentCtx = c.parentComponentVar
			}
			if parentComponentCtx == "" {
				c.reportDiagnostic(slotDirective.Loc.WithEnd(slotDirective.Loc.Pos()+len(slotDirective.RawName)), vue_diagnostics.Slot_does_not_belong_to_the_parent_component)
			} else if slotDirective.Expression != nil {
				c.enterScope()
				slotVar := c.newInternalVariable()
				c.serviceText.WriteString("{\nconst { ")
				if slotDirective.Arg == "" {
					c.serviceText.WriteString("default: ")
				} else {
					// TODO: dynamic name
					c.serviceText.WriteByte('\'')
					c.serviceText.WriteString(slotDirective.Arg)
					c.serviceText.WriteString("': ")
				}
				c.serviceText.WriteString(slotVar)
				c.serviceText.WriteString("} = ")

				c.serviceText.WriteString(parentComponentCtx)
				c.serviceText.WriteString(".slots!\nconst [")
				c.mapExpressionInBindingPosition(slotDirective.Expression)
				c.serviceText.WriteString("] = __VLS_vSlot(")
				c.serviceText.WriteString(slotVar)
				c.serviceText.WriteString(")\n")
			}
		}

		// Handle <slot> elements for slot type generation
		if elem.Tag == "slot" {
			c.generateSlotPropsCapture(elem)
		}

		currCondChain := c.condChain
		c.condChain = conditionalChainNone
		currParentComponentVar := c.parentComponentVar
		c.parentComponentVar = ctxVar
		if c.ignoreError {
			c.mapIgnoreDirective(elementServiceTextStart, c.serviceText.Len())
		} else if c.expectError {
			c.mapExpectErrorDirective(c.expectErrorLoc.Pos(), c.expectErrorLoc.End(), elementServiceTextStart, c.serviceText.Len())
		}
		c.depth++
		for _, child := range elem.Children {
			c.visit(child)
		}
		c.depth--
		// If children ended with an unterminated conditional chain (v-if without v-else),
		// restore blockConditions now since no subsequent sibling will trigger the restore.
		if c.condChain == conditionalChainValid {
			c.blockConditions = c.blockConditions[:c.blockConditionsChainSaveLen]
		}
		c.parentComponentVar = currParentComponentVar
		c.condChain = currCondChain

		if slotDirective != nil && slotDirective.Expression != nil {
			c.exitScope()
			c.serviceText.WriteString("}\n")
		}
		if forDirective != nil {
			c.drainUsedVarScope()
			c.exitScope()
			c.serviceText.WriteString("}\n")
		}
		if conditionalDirective != nil {
			c.serviceText.WriteString("}\n")
			// v-else terminates the chain: restore blockConditions
			if conditionalDirective.Name == "else" {
				c.blockConditions = c.blockConditions[:c.blockConditionsChainSaveLen]
			}
		}
	case vue_ast.KindInterpolation:
		interpolation := el.AsInterpolation()
		interpolationServiceTextStart := c.serviceText.Len()
		c.serviceText.WriteString("( ")
		c.mapExpressionInNonBindingPositionTrimmed(interpolation.Content)
		c.serviceText.WriteString(" );\n")
		if c.ignoreError {
			c.mapIgnoreDirective(interpolationServiceTextStart, c.serviceText.Len())
		} else if c.expectError {
			c.mapExpectErrorDirective(c.expectErrorLoc.Pos(), c.expectErrorLoc.End(), interpolationServiceTextStart, c.serviceText.Len())
		}
	}
	c.ignoreError = false
	c.expectError = false
}

// generateSlotPropsCapture generates slot prop variable capture for <slot> elements.
func (c *templateCodegenCtx) generateSlotPropsCapture(elem *vue_ast.ElementNode) {
	slotName := ""
	propsVar := c.newInternalVariable()

	c.serviceText.WriteString("var ")
	c.serviceText.WriteString(propsVar)
	c.serviceText.WriteString(" = {\n")

	for _, prop := range elem.Props {
		switch prop.Kind {
		case vue_ast.KindAttribute:
			attr := prop.AsAttribute()
			if attr.Name == "name" {
				if attr.Value != nil {
					slotName = attr.Value.Content
				}
				continue
			}
			// Static attribute on <slot>
			c.serviceText.WriteString(attr.Name)
			c.serviceText.WriteString(": ")
			if attr.Value != nil {
				c.serviceText.WriteString("\"")
				c.escapeString(attr.Value.Content)
				c.serviceText.WriteString("\"")
			} else {
				c.serviceText.WriteString("true")
			}
			c.serviceText.WriteString(",\n")
		case vue_ast.KindDirective:
			dir := prop.AsDirective()
			if dir.Name == "bind" && dir.Arg != "" {
				c.serviceText.WriteString(dir.Arg)
				c.serviceText.WriteString(": (")
				if dir.Expression != nil {
					c.mapExpressionInNonBindingPosition(dir.Expression)
				}
				c.serviceText.WriteString("),\n")
			}
		}
	}
	c.serviceText.WriteString("};\n")

	if slotName == "" {
		slotName = "default"
	}

	c.slotProps = append(c.slotProps, slotPropInfo{
		name:     slotName,
		propsVar: propsVar,
	})
}

// generateSlotTypes generates the __VLS_Slots type from collected slot props.
func (c *templateCodegenCtx) generateSlotTypes() {
	// Emit: // @ts-ignore\nvar __VLS_N = __VLS_M, ;
	// Then: type __VLS_Slots = {} & { name?: (props: typeof __VLS_N) => any };
	for i := range c.slotProps {
		nextVar := c.newInternalVariable()
		c.serviceText.WriteString("// @ts-ignore\nvar ")
		c.serviceText.WriteString(nextVar)
		c.serviceText.WriteString(" = ")
		c.serviceText.WriteString(c.slotProps[i].propsVar)
		c.serviceText.WriteString(", ;\n")
		c.slotProps[i].propsVar = nextVar
	}

	c.serviceText.WriteString("type __VLS_Slots = {}\n")
	for _, sp := range c.slotProps {
		c.serviceText.WriteString("& { ")
		if needsQuoting(sp.name) {
			c.serviceText.WriteString("'")
			c.serviceText.WriteString(sp.name)
			c.serviceText.WriteString("'")
		} else {
			c.serviceText.WriteString(sp.name)
		}
		c.serviceText.WriteString("?: (props: typeof ")
		c.serviceText.WriteString(sp.propsVar)
		c.serviceText.WriteString(") => any };\n")
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

// mapExpressionInNonBindingPositionTrimmed maps the expression, skipping leading/trailing whitespace.
// Used for interpolations where Volar trims the expression whitespace.
func (c *templateCodegenCtx) mapExpressionInNonBindingPositionTrimmed(expr *vue_ast.SimpleExpressionNode) {
	// Skip leading whitespace
	sourceContent := c.sourceText[expr.Loc.Pos():expr.Loc.End()]
	leadingSpaces := 0
	for _, ch := range sourceContent {
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			leadingSpaces++
		} else {
			break
		}
	}
	trailingSpaces := 0
	for i := len(sourceContent) - 1; i >= leadingSpaces; i-- {
		if sourceContent[i] == ' ' || sourceContent[i] == '\t' || sourceContent[i] == '\n' || sourceContent[i] == '\r' {
			trailingSpaces++
		} else {
			break
		}
	}

	// Create a modified expression mapper that starts past the leading whitespace
	m := newExpressionMapper(c, expr)
	m.lastMappedPos = expr.Loc.Pos() + leadingSpaces

	if len(expr.Ast.Statements.Nodes) > 0 {
		var parenInner *ast.Node
		nonEmptyCount := 0
		for _, stmt := range expr.Ast.Statements.Nodes {
			if ast.IsEmptyStatement(stmt) {
				continue
			}
			nonEmptyCount++
			if nonEmptyCount == 1 && ast.IsExpressionStatement(stmt) {
				inner := stmt.AsExpressionStatement().Expression
				if ast.IsParenthesizedExpression(inner) {
					parenInner = inner.AsParenthesizedExpression().Expression
				}
			}
		}
		if nonEmptyCount == 1 && parenInner != nil {
			m.mapInNonBindingPosition(parenInner)
			// Map up to end minus trailing spaces and suffix
			endPos := expr.Ast.End() - expr.SuffixLen - trailingSpaces
			m.mapTextToNodePos(endPos)
			return
		}

		for _, stmt := range expr.Ast.Statements.Nodes {
			m.mapInNonBindingPosition(stmt)
		}
	}
	endPos := expr.Ast.End() - expr.SuffixLen - trailingSpaces
	m.mapTextToNodePos(endPos)
}

func (c *templateCodegenCtx) mapExpressionInNonBindingPosition(expr *vue_ast.SimpleExpressionNode) {
	m := newExpressionMapper(c, expr)
	if len(expr.Ast.Statements.Nodes) > 0 {
		// Check for single parenthesized expression, possibly preceded by EmptyStatements.
		// Interpolations are wrapped as ";( expr )" which parses as EmptyStatement + ExpressionStatement(ParenthesizedExpression).
		var parenInner *ast.Node
		nonEmptyCount := 0
		for _, stmt := range expr.Ast.Statements.Nodes {
			if ast.IsEmptyStatement(stmt) {
				continue
			}
			nonEmptyCount++
			if nonEmptyCount == 1 && ast.IsExpressionStatement(stmt) {
				inner := stmt.AsExpressionStatement().Expression
				if ast.IsParenthesizedExpression(inner) {
					parenInner = inner.AsParenthesizedExpression().Expression
				}
			}
		}
		if nonEmptyCount == 1 && parenInner != nil {
			m.mapInNonBindingPosition(parenInner)
			goto FinalizeMapping
		}

		for _, stmt := range expr.Ast.Statements.Nodes {
			m.mapInNonBindingPosition(stmt)
		}
	}
FinalizeMapping:
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
			// TODO: perf
			p := utils.TrimNodeTextRange(m.expr.Ast, node)
			m.mapTextToNodePos(p.Pos())
			m.serviceText.WriteString("__VLS_ctx.")
			m.mapTextToNodePos(p.End())
			// Track this identifier as used
			m.trackUsedVar(node.AsIdentifier().Text)
		}
		return false
	case ast.KindShorthandPropertyAssignment:
		name := node.Name()
		if m.shouldPrefixIdentifier(name) {
			m.mapTextToNodePos(node.Pos())
			m.serviceText.WriteString(name.Text())
			m.serviceText.WriteString(": __VLS_ctx.")
			m.mapTextToNodePos(node.End())
			m.trackUsedVar(name.Text())
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

func camelizeStr(str string) string {
	var buf strings.Builder
	return camelize(str, &buf)
}

func camelize(str string, buf *strings.Builder) string {
	oldLen := buf.Len()
	hadDash := false
	lastWritten := 0
	for i, r := range str {
		if r == '-' {
			hadDash = true
			// TODO: what if double dash, like foo--bar
			continue
		}

		if hadDash {
			hadDash = false
			buf.WriteString(str[lastWritten : i-1])
			// TODO(perf): fast path for ascii, also ToUpper allocates internally
			buf.WriteString(strings.ToUpper(string(r)))
			lastWritten = i + utf8.RuneLen(r)
		}
	}
	buf.WriteString(str[lastWritten:])
	// TODO(perf): double check that this doesn't allocate
	return buf.String()[oldLen:buf.Len()]
}

func isBuiltInComponent(name string) bool {
	switch name {
	case "Teleport",
		"teleport",
		"Suspense",
		"suspense",
		"KeepAlive",
		"keep-alive",
		"BaseTransition",
		"base-transition",
		"Transition",
		"transition",
		"TransitionGroup",
		"transition-group":
		return true
	default:
		return false
	}
}

func isNativeElement(name string) bool {
	_, ok := vue_parser.NativeTags[name]
	return ok
}

func (c *templateCodegenCtx) escapeString(str string) {
	for _, r := range str {
		switch r {
		case '\\':
			c.serviceText.WriteString("\\x5c")
		case '\n':
			c.serviceText.WriteString("\\x0a")
		case '\'':
			c.serviceText.WriteString("\\x27")
		default:
			c.serviceText.WriteRune(r)
		}
	}
}

// getStaticClassName extracts the static class value from an element's attributes.
func getStaticClassName(elem *vue_ast.ElementNode) string {
	for _, prop := range elem.Props {
		if prop.Kind == vue_ast.KindAttribute {
			attr := prop.AsAttribute()
			if attr.Name == "class" && attr.Value != nil {
				return attr.Value.Content
			}
		}
	}
	return ""
}

// emitScopedClassAssertion emits /** @type {__VLS_StyleScopedClasses['class']} */; after an element
// with a class attribute, when a scoped style block exists.
func (c *templateCodegenCtx) emitScopedClassAssertion(elem *vue_ast.ElementNode) {
	if !c.hasScopedStyle {
		return
	}
	className := getStaticClassName(elem)
	if className == "" {
		return
	}
	// Track class for __VLS_StyleScopedClasses type definition
	if c.scopedClassesSet == nil {
		c.scopedClassesSet = map[string]bool{}
	}
	if !c.scopedClassesSet[className] {
		c.scopedClassesSet[className] = true
		c.scopedClasses = append(c.scopedClasses, className)
	}
	c.serviceText.WriteString("/** @type {__VLS_StyleScopedClasses['")
	c.serviceText.WriteString(className)
	c.serviceText.WriteString("']} */;\n")
}

// trackRefAttribute tracks ref attributes on elements for __VLS_TemplateRefs type.
func (c *templateCodegenCtx) trackRefAttribute(elem *vue_ast.ElementNode) {
	for _, prop := range elem.Props {
		if prop.Kind == vue_ast.KindAttribute {
			attr := prop.AsAttribute()
			if attr.Name == "ref" && attr.Value != nil {
				c.templateRefs = append(c.templateRefs, templateRefInfo{
					name:    attr.Value.Content,
					elemTag: elem.Tag,
				})
				return
			}
		}
	}
}

func (c *templateCodegenCtx) generateElementProps(elem *vue_ast.ElementNode, isComponent bool) {
	c.generateElementPropsFiltered(elem, isComponent, nil)
}

func (c *templateCodegenCtx) generateElementPropsFiltered(elem *vue_ast.ElementNode, isComponent bool, skipDirective *vue_ast.DirectiveNode) {
	// Volar generates event props first, then other props (two passes).
	for pass := 0; pass < 2; pass++ {
		for _, prop := range elem.Props {
			// Skip the directive if specified (e.g., :is for dynamic components)
			if skipDirective != nil && prop.Kind == vue_ast.KindDirective {
				dir := prop.AsDirective()
				if dir == skipDirective {
					continue
				}
			}

			isEventProp := prop.Kind == vue_ast.KindDirective && prop.AsDirective().Name == "on"
			if pass == 0 && !isEventProp {
				continue // First pass: events only
			}
			if pass == 1 && isEventProp {
				continue // Second pass: non-events only
			}
			c.generateSingleProp(prop, elem, isComponent)
		}
	}
}

// needsQuoting returns true if a prop name contains characters invalid in JS identifiers.
func needsQuoting(name string) bool {
	return strings.ContainsAny(name, "-:")
}

func (c *templateCodegenCtx) generateSingleProp(prop *vue_ast.Node, elem *vue_ast.ElementNode, isComponent bool) {
	switch prop.Kind {
	case vue_ast.KindAttribute:
		attr := prop.AsAttribute()
		// Volar uses spread format for class: ...{ class: "value" }
		if attr.Name == "class" {
			c.serviceText.WriteString("...{ class: ")
			if attr.Value == nil {
				c.serviceText.WriteString("true")
			} else {
				c.serviceText.WriteString("\"")
				c.escapeString(attr.Value.Content)
				c.serviceText.WriteString("\"")
			}
			c.serviceText.WriteString(" },\n")
			return
		}
		if attr.Name == "ref" {
			c.serviceText.WriteString("ref: ")
			if attr.Value == nil {
				c.serviceText.WriteString("true")
			} else {
				c.serviceText.WriteString("\"")
				c.escapeString(attr.Value.Content)
				c.serviceText.WriteString("\"")
			}
			c.serviceText.WriteString(",\n")
			return
		}
		propNameStart := c.serviceText.Len()
		if isComponent {
			// Component props: always camelize (model-value → modelValue)
			camelize(attr.Name, &c.serviceText)
		} else if needsQuoting(attr.Name) {
			c.serviceText.WriteString("'")
			c.serviceText.WriteString(attr.Name)
			c.serviceText.WriteString("'")
		} else {
			camelize(attr.Name, &c.serviceText)
		}
		propNameEnd := c.serviceText.Len()
		c.mapRange(attr.Loc.Pos(), attr.Loc.Pos()+len(attr.Name), propNameStart, propNameEnd)
		if attr.Value == nil {
			c.serviceText.WriteString(": true,\n")
		} else {
			c.serviceText.WriteString(": \"")
			c.escapeString(attr.Value.Content)
			c.serviceText.WriteString("\",\n")
		}
	case vue_ast.KindDirective:
		dir := prop.AsDirective()
		if dir.Name == "on" {
			// TODO: dynamic event handling
			if !dir.IsStatic {
				break
			}

			// Volar format: ...{ onClick: expr },
			c.serviceText.WriteString("...{ ")
			c.generateEventNameBare(dir)
			c.serviceText.WriteString(": ")
			if isComponent {
				c.serviceText.WriteString("{} as any")
			} else {
				c.generateEventExpression(dir)
			}
			c.serviceText.WriteString("},\n")
			break
		}
		isBind := dir.Name == "bind"
		isModel := dir.Name == "model"
		if !isBind && !isModel {
			break
		}
		// Skip v-model without argument - it's handled as standalone getter in generateVModelGetter
		if isModel && dir.Arg == "" {
			break
		}
		if isBind && dir.Arg == "" {
			c.serviceText.WriteString("...(")
			c.mapExpressionInNonBindingPosition(dir.Expression)
			c.serviceText.WriteString("),\n")
			break
		}

		// Volar uses spread for style: ...{ style: (expr) }
		if isBind && dir.Arg == "style" {
			c.serviceText.WriteString("...{ style: (")
			if dir.Expression != nil {
				c.mapExpressionInNonBindingPosition(dir.Expression)
			}
			c.serviceText.WriteString(") },\n")
			break
		}

		propNameStart := c.serviceText.Len()
		if isBind {
			if isComponent {
				// Component props: always camelize (model-value → modelValue)
				camelize(dir.Arg, &c.serviceText)
			} else if needsQuoting(dir.Arg) {
				c.serviceText.WriteString("'")
				c.serviceText.WriteString(dir.Arg)
				c.serviceText.WriteString("'")
			} else {
				camelize(dir.Arg, &c.serviceText)
			}
		} else {
			if (dir.Arg == "" || dir.Arg == "value") && (elem.Tag == "input" || elem.Tag == "textarea" || elem.Tag == "select") {
				c.serviceText.WriteString("value")
			} else if dir.Arg == "" {
				c.serviceText.WriteString("modelValue")
			} else {
				c.serviceText.WriteString(dir.Arg)
			}
		}
		propNameEnd := c.serviceText.Len()
		// TODO: more accurate range
		c.mapRange(dir.Loc.Pos(), dir.Loc.End(), propNameStart, propNameEnd)

		c.serviceText.WriteString(": (")
		if dir.Expression == nil {
			c.serviceText.WriteString("__VLS_ctx.")
			accessStart := c.serviceText.Len()
			c.serviceText.WriteString(c.serviceText.String()[propNameStart:propNameEnd])
			c.mapRange(dir.Loc.Pos(), dir.Loc.End(), accessStart, c.serviceText.Len())
		} else {
			c.mapExpressionInNonBindingPosition(dir.Expression)
		}
		c.serviceText.WriteString("),\n")
	}
}

func (c *templateCodegenCtx) generateEventName(dir *vue_ast.DirectiveNode) {
	emitNameStart := c.serviceText.Len()
	c.serviceText.WriteByte('\'')
	camelize("on-"+dir.Arg, &c.serviceText)
	c.mapRange(dir.Loc.Pos(), dir.Loc.Pos()+len(dir.RawName), emitNameStart, c.serviceText.Len()+1)
	c.serviceText.WriteByte('\'')
}

// generateEventNameBare generates event name without quotes for spread format.
func (c *templateCodegenCtx) generateEventNameBare(dir *vue_ast.DirectiveNode) {
	emitNameStart := c.serviceText.Len()
	camelize("on-"+dir.Arg, &c.serviceText)
	_ = emitNameStart
	// TODO: mapping
}

func (c *templateCodegenCtx) generateEventExpression(dir *vue_ast.DirectiveNode) {
	if dir.Expression == nil || dir.Expression.Ast == nil {
		c.serviceText.WriteString("() => {}")
		return
	}
	isCompound := true
	if len(dir.Expression.Ast.Statements.Nodes) == 0 {
		panic("Expected event listener AST to have at least one statement")
	}
	if len(dir.Expression.Ast.Statements.Nodes) == 1 {
		if ast.IsExpressionStatement(dir.Expression.Ast.Statements.Nodes[0]) {
			expr := ast.SkipParentheses(dir.Expression.Ast.Statements.Nodes[0].AsExpressionStatement().Expression)
			if ast.IsArrowFunction(expr) || ast.IsIdentifier(expr) || ast.IsPropertyAccessExpression(expr) || ast.IsFunctionExpression(expr) {
				isCompound = false
			}
		}
	}

	if isCompound {
		c.serviceText.WriteString("(...[$event]) => {\n")
		c.enterScope()
		c.declareScopeVar("$event")
		c.pushUsedVarScope()
		// Emit condition guards for compound event handlers inside v-if/v-else blocks.
		// Volar stores conditions wrapped in parens: (expr). Negation prepends !: !(expr).
		// The guard format is: if (!COND) return; — so for negated conditions it becomes
		// if (!!(expr)) return; and for direct conditions if (!(expr)) return;
		for _, cond := range c.blockConditions {
			c.serviceText.WriteString("if (!")
			c.serviceText.WriteString(cond)
			c.serviceText.WriteString(") return;\n")
		}
		c.mapExpressionInNonBindingPosition(dir.Expression)
		c.serviceText.WriteString(";\n") // endOfLine after compound expression
		c.drainUsedVarScope()
		c.exitScope()
		c.serviceText.WriteString("}")
	} else {
		// Simple expression: wrap in parens like Volar: (expr)
		c.serviceText.WriteString("(")
		c.mapExpressionInNonBindingPosition(dir.Expression)
		c.serviceText.WriteString(")")
	}

}

// generateVModelGetter generates a standalone getter expression for v-model without an argument.
// Volar's pattern for default v-model: element props don't include the binding, instead
// a standalone expression `(__VLS_ctx.value);` is generated after the element.
func (c *templateCodegenCtx) generateVModelGetter(elem *vue_ast.ElementNode) {
	for _, prop := range elem.Props {
		if prop.Kind != vue_ast.KindDirective {
			continue
		}
		dir := prop.AsDirective()
		if dir.Name != "model" {
			continue
		}
		// Only generate standalone getter for v-model without argument (default v-model)
		if dir.Arg != "" {
			continue
		}
		// Generate: (__VLS_ctx.expression);
		c.serviceText.WriteString("(")
		if dir.Expression != nil {
			c.mapExpressionInNonBindingPosition(dir.Expression)
		}
		c.serviceText.WriteString(");\n")
		return // Only one v-model per element
	}
}
