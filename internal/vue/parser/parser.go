package vue_parser

import (
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/auvred/golar/internal/utils"
	"github.com/auvred/golar/internal/vue/ast"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/binder"
	"github.com/microsoft/typescript-go/shim/core"
	"github.com/microsoft/typescript-go/shim/parser"
	"github.com/microsoft/typescript-go/shim/tspath"
)

type ParseError struct {
	Message string
	Pos     int
}

type Parser struct {
	// let currentOptions: MergedParserOptions = defaultParserOptions
	currentRoot *vue_ast.RootNode

	// parser state
	sourceText            string
	currentOpenTag        *vue_ast.ElementNode
	currentProp           *vue_ast.Node // AttributeNode | DirectiveNode | null = null
	currentAttrValue      string
	currentAttrStartIndex int //= -1
	currentAttrEndIndex   int //= -1
	inPre                 int
	inVPre                bool
	currentVPreBoundary   *vue_ast.ElementNode
	// TODO(perf): why stack is prepended???
	stack     []*vue_ast.ElementNode
	tokenizer *Tokenizer

	errors []ParseError
}

func Parse(source string) (*vue_ast.RootNode, []ParseError) {
	p := Parser{
		tokenizer:             NewTokenizer(source),
		sourceText:            source,
		currentAttrStartIndex: -1,
		currentAttrEndIndex:   -1,
	}
	p.tokenizer.parser = &p
	p.currentRoot = vue_ast.NewRootNode()
	p.tokenizer.parse()
	return p.currentRoot, p.errors
}

func (p *Parser) emitError(msg string, pos int) {
	p.errors = append(p.errors, ParseError{msg, pos})
}

func (p *Parser) addNode(node *vue_ast.Node) {
	if len(p.stack) > 0 {
		p.stack[0].Children = append(p.stack[0].Children, node)
	} else {
		p.currentRoot.Children = append(p.currentRoot.Children, node)
	}
}

func (p *Parser) onerr(code ErrorCode, index int) {
}

func (p *Parser) ontext(start int, end int) {
	p.onText(p.sourceText[start:end], start, end)
}

func (p *Parser) ontextentity(char string, start int, end int) {
	p.onText(char, start, end)
}

func (p *Parser) oninterpolation(start int, end int) {
	if p.inVPre {
		p.onText(p.sourceText[start:end], start, end)
		return
	}
	innerStart := start + len(p.tokenizer.delimiterOpen)
	innerEnd := end - len(p.tokenizer.delimiterClose)
	// for {
	// 	r, size := utf8.DecodeRuneInString(p.sourceText[innerStart:])
	// 	if !isWhitespace(r) {
	// 		break
	// 	}
	// 	innerStart += size
	// }
	// for {
	// 	r, size := utf8.DecodeLastRuneInString(p.sourceText[:innerEnd])
	// 	if !isWhitespace(r) {
	// 		break
	// 	}
	// 	innerEnd -= size
	// }
	expContent := p.sourceText[innerStart:innerEnd]
	// decode entities for backwards compat
	// TODO:
	// if exp.includes("&") {
	// 	if __BROWSER__ {
	// 		exp = currentOptions.decodeEntities(exp, false)
	// 	} else {
	// 		exp = decodeHTML(exp)
	// 	}
	// }

	exp := vue_ast.NewSimpleExpressionNode(ParseTsAst("("+expContent+")"), core.NewTextRange(innerStart, innerEnd), 1, 1)

	p.addNode(vue_ast.NewInterpolationNode(
		exp,
		core.NewTextRange(start, end),
	).AsNode())
}

func (p *Parser) onopentagname(start int, end int) {
	name := p.sourceText[start:end]
	p.currentOpenTag = vue_ast.NewElementNode(
		// TODO: do we need to support namespaces?
		// currentOptions.getNamespace(name, stack[0], currentOptions.ns),
		vue_ast.NamespaceHTML,
		name,
		core.NewTextRange(start-1, end),
	)
}

// TODO: opts? currentOptions.isPreTag
func isPreTag(tag string) bool {
	return tag == "pre"
}

func (p *Parser) onopentagend(end int) {
	if p.tokenizer.inSFCRoot() {
		p.currentOpenTag.InnerLoc = core.NewTextRange(end+1, end+1)
	}
	p.addNode(p.currentOpenTag.AsNode())
	if p.currentOpenTag.Ns == vue_ast.NamespaceHTML && isPreTag(p.currentOpenTag.Tag) {
		p.inPre++
	}
	if _, ok := VoidTags[p.currentOpenTag.Tag]; ok {
		p.onCloseTag(p.currentOpenTag, end, false)
	} else {
		p.stack = slices.Insert(p.stack, 0, p.currentOpenTag)
		if p.currentOpenTag.Ns == vue_ast.NamespaceSVG || p.currentOpenTag.Ns == vue_ast.NamespaceMATH_ML {
			p.tokenizer.inXML = true
		}
	}
	p.currentOpenTag = nil
}

func (p *Parser) onclosetag(start int, end int) {
	name := p.sourceText[start:end]
	if _, ok := VoidTags[name]; !ok {
		found := false
		for i, e := range p.stack {
			if strings.ToLower(e.Tag) == strings.ToLower(name) {
				found = true
				if i > 0 {
					p.emitError("Missing end tag", p.stack[0].Loc.Pos())
				}
				for j := 0; j <= i; j++ {
					el := p.stack[0]
					p.stack = slices.Delete(p.stack, 0, 1)
					p.onCloseTag(el, end, j < i)
				}
				break
			}
		}
		if !found {
			p.emitError("Invalid end tag", p.backTrack(start, CharCodeLt))
		}
	}
}

func (p *Parser) onselfclosingtag(end int) {
	name := p.currentOpenTag.Tag
	p.currentOpenTag.IsSelfClosing = true
	p.onopentagend(end)
	if len(p.stack) > 0 && p.stack[0].Tag == name {
		el := p.stack[0]
		p.stack = slices.Delete(p.stack, 0, 1)
		p.onCloseTag(el, end, false)
	}
}

func (p *Parser) onattribname(start int, end int) {
	// plain attribute
	p.currentProp = vue_ast.NewAttributeNode(
		p.sourceText[start:end],
		core.NewTextRange(start, end),
		core.NewTextRange(start, -1),
	).AsNode()
}

func (p *Parser) ondirname(start int, end int) {
	raw := p.sourceText[start:end]
	var name string
	switch raw {
	case ".", ":":
		name = "bind"
	case "@":
		name = "on"
	case "#":
		name = "slot"
	default:
		name = raw[2:]
	}

	if !p.inVPre && name == "" {
		p.emitError("Missing directive name", start)
	}

	if p.inVPre || name == "" {
		p.currentProp = vue_ast.NewAttributeNode(
			raw,
			core.NewTextRange(start, end),
			core.NewTextRange(start, -1),
		).AsNode()
	} else {
		p.currentProp = vue_ast.NewDirectiveNode(
			name,
			raw,
			// TODO:
			// modifiers: raw == ".' ? [createSimpleExpression('prop")] : [],
			core.NewTextRange(start, end),
			core.NewTextRange(start, -1),
		).AsNode()
		if name == "pre" {
			p.tokenizer.inVPre = true
			p.inVPre = true
			p.currentVPreBoundary = p.currentOpenTag
			// convert dirs before this one to attributes
			// TODO:
			// props := currentOpenTag.props
			// for i := 0; i < len(props); i++ {
			// 	if props[i].type == vue_ast.KindDirective {
			// 		props[i] = dirToAttr(props[i] as DirectiveNode)
			// 	}
			// }
		}
	}
}

func isVPre(p *vue_ast.Node) bool {
	return p.Kind == vue_ast.KindDirective && p.AsDirective().Name == "pre"
}

func (p *Parser) ondirarg(start int, end int) {
	if start == end {
		return
	}
	arg := p.sourceText[start:end]
	if p.inVPre && !isVPre(p.currentProp) {
		prop := p.currentProp.AsAttribute()
		prop.Name += arg
		prop.Loc = prop.Loc.WithEnd(end)
	} else {
		prop := p.currentProp.AsDirective()
		prop.IsStatic = arg[0] != '['
		if !prop.IsStatic {
			arg = arg[1 : len(arg)-1]
		}
		prop.Arg = arg
	}
}

func (p *Parser) ondirmodifier(start int, end int) {
	mod := p.sourceText[start:end]
	if p.inVPre && !isVPre(p.currentProp) {
		prop := p.currentProp.AsAttribute()
		prop.Name += "." + mod
		prop.Loc = prop.Loc.WithEnd(end)
		return
	}
	prop := p.currentProp.AsDirective()
	if prop.Name == "slot" {
		// slot has no modifiers, special case for edge cases like
		// https://github.com/vuejs/language-tools/issues/2710
		// TODO:
		// arg := (currentProp as DirectiveNode).arg
		// if arg {
		// 	;(arg as SimpleExpressionNode).content += "." + mod
		// 	setLocEnd(arg.loc, end)
		// }
	} else {
		// TODO:
		// exp := createSimpleExpression(mod, true, getLoc(start, end))
		// ;(currentProp as DirectiveNode).modifiers.push(exp)
	}
}

func (p *Parser) onattribdata(start int, end int) {
	p.currentAttrValue += p.sourceText[start:end]
	if p.currentAttrStartIndex < 0 {
		p.currentAttrStartIndex = start
	}
	p.currentAttrEndIndex = end
}

func (p *Parser) onattribentity(char string, start int, end int) {
	p.currentAttrValue += char
	if p.currentAttrStartIndex < 0 {
		p.currentAttrStartIndex = start
	}
	p.currentAttrEndIndex = end
}

func (p *Parser) onattribnameend(end int) {
	start := p.currentProp.Loc.Pos()
	name := p.sourceText[start:end]
	if p.currentProp.Kind == vue_ast.KindDirective {
		p.currentProp.AsDirective().RawName = name
	}
	// check duplicate attrs
	// TODO:
	// if 	// 	currentOpenTag!.props.some(
	// 		p => (p.type == vue_ast.KindDirective ? p.rawName : p.name) == name,
	// 	)
	//  {
	// 	emitError(ErrorCodes.DUPLICATE_ATTRIBUTE, start)
	// }
}

func (p *Parser) onattribend(quote QuoteType, end int) {
	if p.currentOpenTag != nil && p.currentProp != nil {
		// finalize end pos
		p.currentProp.Loc = p.currentProp.Loc.WithEnd(end)

		if quote != QuoteTypeNoValue {
			if p.currentProp.Kind == vue_ast.KindAttribute {
				// assign value

				prop := p.currentProp.AsAttribute()
				// TODO: why???
				// condense whitespaces in class
				// if prop.Name == "class" {
				// 	p.currentAttrValue = condense(p.currentAttrValue).trim()
				// }

				if quote == QuoteTypeUnquoted && p.currentAttrValue == "" {
					p.emitError("Missing attribute value", end)
				}

				prop.Value = vue_ast.NewTextNode(
					p.currentAttrValue,
					core.NewTextRange(p.currentAttrStartIndex, p.currentAttrEndIndex),
				)
				if quote == QuoteTypeUnquoted {
					prop.Value.Loc = core.NewTextRange(p.currentAttrStartIndex-1, p.currentAttrEndIndex+1)
				}
				if p.tokenizer.inSFCRoot() &&
					p.currentOpenTag.Tag == "template" &&
					prop.Name == "lang" &&
					p.currentAttrValue != "" &&
					p.currentAttrValue != "html" {
					// SFC root template with preprocessor lang, force tokenizer to
					// RCDATA mode
					t := []rune(`</template`)
					p.tokenizer.enterRCDATA(&t, 0)
				}
			} else {
				// directive
				// TODO:
				// expParseMode := ExpParseMode.Normal
				// if !__BROWSER__ {
				// 	if currentProp.name == "for" {
				// 		expParseMode = ExpParseMode.Skip
				// 	} else if currentProp.name == "slot" {
				// 		expParseMode = ExpParseMode.Params
				// 	} else if currentProp.name == "on" &&
				// 		currentAttrValue.includes(";") {
				// 		expParseMode = ExpParseMode.Statements
				// 	}
				// }
				prop := p.currentProp.AsDirective()
				trimmedValue, _, _ := utils.TrimWhiteSpaceOrLineTerminator(p.currentAttrValue)
				isValueEmpty := len(trimmedValue) == 0
				isVFor := prop.Name == "for"
				if isValueEmpty || isVFor {
					prop.Expression = vue_ast.NewSimpleExpressionNode(nil, core.NewTextRange(p.currentAttrStartIndex, p.currentAttrEndIndex), 0, 0)
					if isVFor {
						prop.ForParseResult = parseForExpression(
							p.currentAttrValue,
							prop.Expression.Loc,
						)
						if prop.ForParseResult == nil {
							p.emitError("v-for has invalid expression", p.currentAttrStartIndex)
						}
					}
				} else {
					switch prop.Name {
					case "slot":
						prop.Expression = vue_ast.NewSimpleExpressionNode(
							ParseTsAst("("+p.currentAttrValue+")=>{}"),
							core.NewTextRange(p.currentAttrStartIndex, p.currentAttrEndIndex),
							1,
							5,
						)
					case "on":
						// https://github.com/vuejs/core/issues/14287
						prefixLen := 1
						suffixLen := 1
						ast := ParseTsAst("(" + p.currentAttrValue + ")")
						// TODO: report syntactic diagnostics
						diagnostics := ast.Diagnostics()
						if len(diagnostics) != 0 {
							prefixLen = 0
							suffixLen = 0
							ast = ParseTsAst(p.currentAttrValue)
							if len(ast.Diagnostics()) != 0 {
								// TODO: report
								_ = diagnostics
							}
						}
						prop.Expression = vue_ast.NewSimpleExpressionNode(ast, core.NewTextRange(p.currentAttrStartIndex, p.currentAttrEndIndex), prefixLen, suffixLen)
					default:
						prop.Expression = vue_ast.NewSimpleExpressionNode(ParseTsAst(
							"("+p.currentAttrValue+")",
						), core.NewTextRange(p.currentAttrStartIndex, p.currentAttrEndIndex), 1, 1)
					}
				}
				// if currentProp.name == "for" {
				// 	currentProp.forParseResult = parseForExpression(currentProp.exp)
				// }
				// TODO: do we need it?
				// // 2.x compat v-bind:foo.sync -> v-model:foo
				// syncIndex := -1
				// if __COMPAT__ &&
				// 	currentProp.name == "bind" &&
				// 	(syncIndex = currentProp.modifiers.findIndex(
				// 		mod => mod.content == "sync",
				// 	)) > -1 &&
				// 	checkCompatEnabled(
				// 		CompilerDeprecationTypes.COMPILER_V_BIND_SYNC,
				// 		currentOptions,
				// 		currentProp.loc,
				// 		currentProp.arg!.loc.source,
				// 	) {
				// 	currentProp.name = "model"
				// 	currentProp.modifiers.splice(syncIndex, 1)
				// }
			}
		}
		if p.currentProp.Kind != vue_ast.KindDirective ||
			p.currentProp.AsDirective().Name != "pre" {
			p.currentOpenTag.Props = append(p.currentOpenTag.Props, p.currentProp)
		}
	}
	p.currentAttrValue = ""
	p.currentAttrStartIndex = -1
	p.currentAttrEndIndex = -1
}

func (p *Parser) oncomment(start int, end int) {
	p.addNode(vue_ast.NewCommentNode(
		p.sourceText[start:end],
		core.NewTextRange(start-4, end+3),
	).AsNode())
}

func (p *Parser) onend() {
	end := len(p.sourceText)
	// EOF ERRORS
	if p.tokenizer.state != StateText {
		switch p.tokenizer.state {
		case StateBeforeTagName:
		case StateBeforeClosingTagName:
			p.emitError("EOF before tag name", end)
			break
		case StateInterpolation:
		case StateInterpolationClose:
			p.emitError(
				"Missing interpolation end",
				p.tokenizer.sectionStart,
			)
			break
		case StateInCommentLike:
			if p.tokenizer.currentSequence == &SequenceCdataEnd {
				p.emitError("EOF in cdata", end)
			} else {
				p.emitError("EOF in comment", end)
			}
			break
		case StateInTagName:
		case StateInSelfClosingTag:
		case StateInClosingTagName:
		case StateBeforeAttrName:
		case StateInAttrName:
		case StateInDirName:
		case StateInDirArg:
		case StateInDirDynamicArg:
		case StateInDirModifier:
		case StateAfterAttrName:
		case StateBeforeAttrValue:
		case StateInAttrValueDq: // "
		case StateInAttrValueSq: // '
		case StateInAttrValueNq:
			p.emitError("EOF in tag", end)
			break
		default:
			// console.log(p.tokenizer.state)
			break
		}
	}
	for _, e := range p.stack {
		p.onCloseTag(e, end-1, false)
		p.emitError("Missing end tag", e.Loc.Pos())
	}
}

func (p *Parser) oncdata(start int, end int) {
	if len(p.stack) > 0 && p.stack[0].Ns != vue_ast.NamespaceHTML {
		p.onText(p.sourceText[start:end], start, end)
	} else {
		p.emitError("CDATA in HTML content", start-9)
	}
}

func (p *Parser) onprocessinginstruction(start int, endIndex int) {
	// ignore as we do not have runtime handling for this, only check error
	ns := vue_ast.NamespaceHTML // currentOptions.ns
	if len(p.stack) > 0 {
		ns = p.stack[0].Ns
	}
	if ns == vue_ast.NamespaceHTML {
		p.emitError(
			"Unexpected question mark instead of tag name",
			start-1,
		)
	}
}

// https://github.com/vuejs/core/blob/d8a2de44859bdea7fad6a939ae3dbb651527045f/packages/compiler-core/src/utils.ts#L571
var forAliasRE = regexp.MustCompile(`([\s\S]*?)\s+(?:in|of)\s+(\S[\s\S]*)`)

// https://github.com/vuejs/core/blob/d8a2de44859bdea7fad6a939ae3dbb651527045f/packages/compiler-core/src/parser.ts#L493
// This regex doesn't cover the case if key or index aliases have destructuring,
// but those do not make sense in the first place, so this works in practice.
var forIteratorRE = regexp.MustCompile(`,([^,\}\]]*)(?:,([^,\}\]]*))?$`)

func parseForExpression(exp string, loc core.TextRange) *vue_ast.ForParseResult {
	inMatch := forAliasRE.FindStringSubmatchIndex(exp)
	if len(inMatch) < 6 {
		return nil
	}

	lhsStart := inMatch[2]
	lhsEnd := inMatch[3]
	rhsStart := inMatch[4]
	rhsEnd := inMatch[5]

	locStart := loc.Pos()
	lhs := exp[lhsStart:lhsEnd]
	rhs := exp[rhsStart:rhsEnd]

	result := &vue_ast.ForParseResult{
		Source: vue_ast.NewSimpleExpressionNode(ParseTsAst("("+rhs+")"), core.NewTextRange(loc.Pos()+rhsStart, loc.Pos()+rhsEnd), 1, 1),
		Value:  nil,
		Key:    nil,
		Index:  nil,
	}

	valueContent := lhs
	valueOffset := lhsStart
	if len(valueContent) > 0 && valueContent[0] == '(' && valueContent[len(valueContent)-1] == ')' {
		valueOffset++
		valueContent = valueContent[1:]
		valueContent = valueContent[:len(valueContent)-1]
	}

	iteratorMatch := forIteratorRE.FindStringSubmatchIndex(valueContent)
	if len(iteratorMatch) > 0 {
		if len(iteratorMatch) >= 4 && iteratorMatch[2] != -1 {
			keyContent := valueContent[iteratorMatch[2]:iteratorMatch[3]]
			trimmedKeyContent, _, _ := utils.TrimWhiteSpaceOrLineTerminator(keyContent)
			if trimmedKeyContent != "" {
				keyStart := locStart + valueOffset + iteratorMatch[2]
				keyEnd := keyStart + len(keyContent)
				keyParseContent := "(" + keyContent + ")=>{}"
				result.Key = vue_ast.NewSimpleExpressionNode(
					ParseTsAst(keyParseContent),
					core.NewTextRange(keyStart, keyEnd),
					1,
					5,
				)
			}
		}

		if len(iteratorMatch) >= 6 && iteratorMatch[4] != -1 {
			indexContent := valueContent[iteratorMatch[4]:iteratorMatch[5]]
			trimmedIndexContent, _, _ := utils.TrimWhiteSpaceOrLineTerminator(indexContent)
			if trimmedIndexContent != "" {
				indexStart := locStart + valueOffset + iteratorMatch[4]
				indexEnd := indexStart + len(indexContent)
				indexParseContent := "(" + indexContent + ")=>{}"
				result.Index = vue_ast.NewSimpleExpressionNode(
					ParseTsAst(indexParseContent),
					core.NewTextRange(indexStart, indexEnd),
					1,
					5,
				)
			}
		}

		valueContent = valueContent[:iteratorMatch[0]]
	}

	if strings.TrimSpace(valueContent) != "" {
		valueStart := locStart + valueOffset
		valueEnd := valueStart + len(valueContent)
		valueParseContent := "(" + valueContent + ")=>{}"
		result.Value = vue_ast.NewSimpleExpressionNode(
			ParseTsAst(valueParseContent),
			core.NewTextRange(valueStart, valueEnd),
			1,
			5,
		)
	}

	return result
}

func (p *Parser) onText(content string, start, end int) {
	children := p.currentRoot.Children
	if len(p.stack) > 0 {
		children = p.stack[0].Children
	}
	if len(children) > 0 && children[len(children)-1].Kind == vue_ast.KindText {
		lastNode := children[len(children)-1].AsText()
		// merge
		lastNode.Content += content
		lastNode.Loc = lastNode.Loc.WithEnd(end)
	} else {
		n := vue_ast.NewTextNode(
			content,
			core.NewTextRange(start, end),
		).AsNode()
		if len(p.stack) > 0 {
			p.stack[0].Children = append(p.stack[0].Children, n)
		} else {
			p.currentRoot.Children = append(p.currentRoot.Children, n)
		}
	}
}

func (p *Parser) onCloseTag(el *vue_ast.ElementNode, end int, isImplied bool) {
	// attach end position
	if isImplied {
		// implied close, end should be backtracked to close
		el.Loc = el.Loc.WithEnd(p.backTrack(end, CharCodeLt))
	} else {
		el.Loc = el.Loc.WithEnd(p.lookAhead(end, CharCodeGt) + 1)
	}

	if p.tokenizer.inSFCRoot() {
		if len(el.Children) > 0 {
			el.InnerLoc = el.InnerLoc.WithEnd(el.Children[len(el.Children)-1].Loc.End())
			if el.Tag == "script" {
				if len(el.Children) != 1 {
					panic("assertion failed: <script> has more than 1 child")
				}
				content := p.sourceText[el.Children[0].Loc.Pos():el.Children[0].Loc.End()]
				el.Ast = ParseTsAst(content)
			}
		}
	}

	// refine element type
	// { tag, ns, children } := el
	if !p.inVPre {
		// TODO: do we need it?
		// if el.Tag == "slot" {
		// 	el.TagType = vue_ast.ElementTypeSLOT
		// } else if isFragmentTemplate(el) {
		// 	el.TagType = vue_ast.ElementTypeTEMPLATE
		// } else if isComponent(el) {
		// 	el.TagType = vue_ast.ElementTypeCOMPONENT
		// }
	}

	// whitespace management
	if !p.tokenizer.inRCDATA {
		// TODO:
		// el.Children = condenseWhitespace(children)
	}

	// TODO:
	// if ns == vue_ast.NamespaceHTML && currentOptions.isIgnoreNewlineTag(tag) {
	// 	// remove leading newline for <textarea> and <pre> per html spec
	// 	// https://html.spec.whatwg.org/multipage/parsing.html#parsing-main-inbody
	// 	first := children[0]
	// 	if first && first.Type == vue_ast.KindText {
	// 		// TODO:
	// 		// first.content = first.content.replace(/^\r?\n/, "")
	// 	}
	// }

	if isPreTag(el.Tag) {
		p.inPre--
	}
	if p.currentVPreBoundary == el {
		p.tokenizer.inVPre = false
		p.inVPre = false
		p.currentVPreBoundary = nil
	}
	// TODO:
	// ns := currentOptions.ns
	// if len(p.stack) > 0 {
	// 	ns = stack[0].Ns
	// }
	// if tokenizer.inXML && ns == vue_ast.NamespaceHTML {
	// 	tokenizer.inXML = false
	// }
}

func (p *Parser) lookAhead(index int, c rune) int {
	for off, r := range p.sourceText[index:] {
		if r == c {
			return index + off
		}
	}
	return len(p.sourceText) - 1
}

func (p *Parser) backTrack(index int, c rune) int {
	for i := index; i >= 0; {
		r, size := utf8.DecodeLastRuneInString(p.sourceText[:i+1])
		if r == c {
			return i + 1 - size
		}
		i -= size
	}
	return 0
}

func isUpperCase(c rune) bool {
	return c >= 'A' && c <= 'Z'
}

// windowsNewlineRE := /\r\n/g
// function condenseWhitespace(nodes: TemplateChildNode[]): TemplateChildNode[] {
// 	shouldCondense := currentOptions.whitespace != "preserve"
// 	removedWhitespace := false
// 	for i := 0; i < len(nodes); i++ {
// 		node := nodes[i]
// 		if node.type == vue_ast.KindText {
// 			if !inPre {
// 				if isAllWhitespace(node.content) {
// 					prev := nodes[i - 1] && nodes[i - 1].type
// 					next := nodes[i + 1] && nodes[i + 1].type
// 					// Remove if:
// 					// - the whitespace is the first or last node, or:
// 					// - (condense mode) the whitespace is between two comments, or:
// 					// - (condense mode) the whitespace is between comment and element, or:
// 					// - (condense mode) the whitespace is between two elements AND contains newline
// 					if !prev ||
// 						!next ||
// 						(shouldCondense &&
// 							((prev == vue_ast.KindComment &&
// 								(next == vue_ast.KindComment || next == vue_ast.NodeTypeELEMENT)) ||
// 								(prev == vue_ast.KindElement &&
// 									(next == vue_ast.KindComment ||
// 										(next == vue_ast.KindElement &&
// 											hasNewlineChar(node.content)))))) {
// 						removedWhitespace = true
// 						nodes[i] = null as any
// 					} else {
// 						// Otherwise, the whitespace is condensed into a single space
// 						node.content = " "
// 					}
// 				} else if shouldCondense {
// 					// in condense mode, consecutive whitespaces in text are condensed
// 					// down to a single space.
// 					node.content = condense(node.content)
// 				}
// 			} else {
// 				// #6410 normalize windows newlines in <pre>:
// 				// in SSR, browsers normalize server-rendered \r\n into a single \n
// 				// in the DOM
// 				node.content = node.content.replace(windowsNewlineRE, "\n")
// 			}
// 		}
// 	}
// 	return removedWhitespace ? nodes.filter(Boolean) : nodes
// }

// function hasNewlineChar(str: string) {
// 	for i := 0; i < len(str); i++ {
// 		c := str.charCodeAt(i)
// 		if c == CharCodes.NewLine || c == CharCodes.CarriageReturn {
// 			return true
// 		}
// 	}
// 	return false
// }

func condense(s string) string {
	var result []rune
	prevIsWhitespace := false

	for _, r := range s {
		if unicode.IsSpace(r) {
			if !prevIsWhitespace {
				result = append(result, ' ')
				prevIsWhitespace = true
			}
		} else {
			result = append(result, r)
			prevIsWhitespace = false
		}
	}

	return string(result)
}

// function dirToAttr(dir: DirectiveNode): AttributeNode {
// 	attr: AttributeNode := {
// 		type: vue_ast.KindAttribute,
// 		name: dir.rawName!,
// 		nameLoc: getLoc(
// 			dir.loc.start.offset,
// 			dir.loc.start.offset + len(dir.rawName!),
// 		),
// 		value: undefined,
// 		loc: dir.loc,
// 	}
// 	if dir.exp {
// 		// account for quotes
// 		loc := dir.exp.loc
// 		if loc.end.offset < dir.loc.end.offset {
// 			loc.start.offset--
// 			loc.start.column--
// 			loc.end.offset++
// 			loc.end.column++
// 		}
// 		attr.value = {
// 			type: vue_ast.KindText,
// 			content: (dir.exp as SimpleExpressionNode).content,
// 			loc,
// 		}
// 	}
// 	return attr
// }

// enum ExpParseMode {
// 	Normal,
// 	Params,
// 	Statements,
// 	Skip,
// }
//
// function createExp(
// 	content: SimpleExpressionNode["content"],
// 	isStatic: SimpleExpressionNode["isStatic"] = false,
// 	loc: SourceLocation,
// 	constType: ConstantTypes = ConstantTypes.NOT_CONSTANT,
// 	parseMode = ExpParseMode.Normal,
// ) {
// 	exp := createSimpleExpression(content, isStatic, loc, constType)
// 	if !__BROWSER__ &&
// 		!isStatic &&
// 		currentOptions.prefixIdentifiers &&
// 		parseMode != ExpParseMode.Skip &&
// 		content.trim() {
// 		if isSimpleIdentifier(content) {
// 			exp.vue_ast = null // fast path
// 			return exp
// 		}
// 		try {
// 			plugins := currentOptions.expressionPlugins
// 			options: BabelOptions := {
// 				plugins: plugins ? [...plugins, "typescript'] : ['typescript"],
// 			}
// 			if parseMode == ExpParseMode.Statements {
// 				// v-on with multi-inline-statements, pad 1 char
// 				exp.vue_ast = parse(` ${content} `, options).program
// 			} else if parseMode == ExpParseMode.Params {
// 				exp.vue_ast = parseExpression(`(${content})=>{}`, options)
// 			} else {
// 				// normal exp, wrap with parens
// 				exp.vue_ast = parseExpression(`(${content})`, options)
// 			}
// 		} catch (e: any) {
// 			exp.vue_ast = false // indicate an error
// 			emitError(ErrorCodes.X_INVALID_EXPRESSION, loc.start.offset, e.message)
// 		}
// 	}
// 	return exp
// }

// function emitError(code: ErrorCodes, index: number, message?: string) {
// 	currentOptions.onError(
// 		createCompilerError(code, getLoc(index, index), undefined, message),
// 	)
// }

// function reset() {
// 	tokenizer.reset()
// 	currentOpenTag = null
// 	currentProp = null
// 	currentAttrValue = ""
// 	currentAttrStartIndex = -1
// 	currentAttrEndIndex = -1
// len(	stack) = 0
// }

// export function baseParse(input: string, options?: ParserOptions): RootNode {
// 	reset()
// 	sourceText = input
// 	currentOptions = extend({}, defaultParserOptions)
//
// 	if options {
// 		let key: keyof ParserOptions
// 		for (key in options) {
// 			if options[key] != null {
// 				// @ts-expect-error
// 				currentOptions[key] = options[key]
// 			}
// 		}
// 	}
//
// 	if __DEV__ {
// 		if !__BROWSER__ && currentOptions.decodeEntities {
// 			console.warn(
// 				`[@vue/compiler-core] decodeEntities option is passed but will be ` +
// 					`ignored in non-browser builds.`,
// 			)
// 		} else if __BROWSER__ && !__TEST__ && !currentOptions.decodeEntities {
// 			throw new Error(
// 				`[@vue/compiler-core] decodeEntities option is required in browser builds.`,
// 			)
// 		}
// 	}
//
// 	tokenizer.mode =
// 		currentOptions.parseMode == "html"
// 			? ParseMode.HTML
// 			: currentOptions.parseMode == "sfc"
// 				? ParseMode.SFC
// 				: ParseMode.BASE
//
//
// 		delimiters := options && options.delimiters
// 	if delimiters {
// 		tokenizer.delimiterOpen = toCharCodes(delimiters[0])
// 		tokenizer.delimiterClose = toCharCodes(delimiters[1])
// 	}
//
// 	root := (currentRoot = createRoot([], input))
// 	tokenizer.parse(sourceText)
// 	root.loc = getLoc(0, len(input))
// 	root.children = condenseWhitespace(root.children)
// 	currentRoot = null
// 	return root
// }

// copied from https://github.com/vuejs/core/blob/44ee43848fe8563c914be6cf731157e360d4e801/packages/shared/src/domTagConfig.ts
var (
	NativeTags = map[string]struct{}{
		// html tags
		"html":       struct{}{},
		"body":       struct{}{},
		"base":       struct{}{},
		"head":       struct{}{},
		"link":       struct{}{},
		"meta":       struct{}{},
		"style":      struct{}{},
		"title":      struct{}{},
		"address":    struct{}{},
		"article":    struct{}{},
		"aside":      struct{}{},
		"footer":     struct{}{},
		"header":     struct{}{},
		"hgroup":     struct{}{},
		"h1":         struct{}{},
		"h2":         struct{}{},
		"h3":         struct{}{},
		"h4":         struct{}{},
		"h5":         struct{}{},
		"h6":         struct{}{},
		"nav":        struct{}{},
		"section":    struct{}{},
		"div":        struct{}{},
		"dd":         struct{}{},
		"dl":         struct{}{},
		"dt":         struct{}{},
		"figcaption": struct{}{},
		"figure":     struct{}{},
		"picture":    struct{}{},
		"hr":         struct{}{},
		"img":        struct{}{},
		"li":         struct{}{},
		"main":       struct{}{},
		"ol":         struct{}{},
		"p":          struct{}{},
		"pre":        struct{}{},
		"ul":         struct{}{},
		"a":          struct{}{},
		"b":          struct{}{},
		"abbr":       struct{}{},
		"bdi":        struct{}{},
		"bdo":        struct{}{},
		"br":         struct{}{},
		"cite":       struct{}{},
		"code":       struct{}{},
		"data":       struct{}{},
		"dfn":        struct{}{},
		"em":         struct{}{},
		"i":          struct{}{},
		"kbd":        struct{}{},
		"mark":       struct{}{},
		"q":          struct{}{},
		"rp":         struct{}{},
		"rt":         struct{}{},
		"ruby":       struct{}{},
		"s":          struct{}{},
		"samp":       struct{}{},
		"small":      struct{}{},
		"span":       struct{}{},
		"strong":     struct{}{},
		"sub":        struct{}{},
		"sup":        struct{}{},
		"time":       struct{}{},
		"u":          struct{}{},
		"var":        struct{}{},
		"wbr":        struct{}{},
		"area":       struct{}{},
		"audio":      struct{}{},
		"map":        struct{}{},
		"track":      struct{}{},
		"video":      struct{}{},
		"embed":      struct{}{},
		"object":     struct{}{},
		"param":      struct{}{},
		"source":     struct{}{},
		"canvas":     struct{}{},
		"script":     struct{}{},
		"noscript":   struct{}{},
		"del":        struct{}{},
		"ins":        struct{}{},
		"caption":    struct{}{},
		"col":        struct{}{},
		"colgroup":   struct{}{},
		"table":      struct{}{},
		"thead":      struct{}{},
		"tbody":      struct{}{},
		"td":         struct{}{},
		"th":         struct{}{},
		"tr":         struct{}{},
		"button":     struct{}{},
		"datalist":   struct{}{},
		"fieldset":   struct{}{},
		"form":       struct{}{},
		"input":      struct{}{},
		"label":      struct{}{},
		"legend":     struct{}{},
		"meter":      struct{}{},
		"optgroup":   struct{}{},
		"option":     struct{}{},
		"output":     struct{}{},
		"progress":   struct{}{},
		"select":     struct{}{},
		"textarea":   struct{}{},
		"details":    struct{}{},
		"dialog":     struct{}{},
		"menu":       struct{}{},
		"summary":    struct{}{},
		"template":   struct{}{},
		"blockquote": struct{}{},
		"iframe":     struct{}{},
		"tfoot":      struct{}{},

		// svg tags
		"svg":                 struct{}{},
		"animate":             struct{}{},
		"animateMotion":       struct{}{},
		"animateTransform":    struct{}{},
		"circle":              struct{}{},
		"clipPath":            struct{}{},
		"color-profile":       struct{}{},
		"defs":                struct{}{},
		"desc":                struct{}{},
		"discard":             struct{}{},
		"ellipse":             struct{}{},
		"feBlend":             struct{}{},
		"feColorMatrix":       struct{}{},
		"feComponentTransfer": struct{}{},
		"feComposite":         struct{}{},
		"feConvolveMatrix":    struct{}{},
		"feDiffuseLighting":   struct{}{},
		"feDisplacementMap":   struct{}{},
		"feDistantLight":      struct{}{},
		"feDropShadow":        struct{}{},
		"feFlood":             struct{}{},
		"feFuncA":             struct{}{},
		"feFuncB":             struct{}{},
		"feFuncG":             struct{}{},
		"feFuncR":             struct{}{},
		"feGaussianBlur":      struct{}{},
		"feImage":             struct{}{},
		"feMerge":             struct{}{},
		"feMergeNode":         struct{}{},
		"feMorphology":        struct{}{},
		"feOffset":            struct{}{},
		"fePointLight":        struct{}{},
		"feSpecularLighting":  struct{}{},
		"feSpotLight":         struct{}{},
		"feTile":              struct{}{},
		"feTurbulence":        struct{}{},
		"filter":              struct{}{},
		"foreignObject":       struct{}{},
		"g":                   struct{}{},
		"hatch":               struct{}{},
		"hatchpath":           struct{}{},
		"image":               struct{}{},
		"line":                struct{}{},
		"linearGradient":      struct{}{},
		"marker":              struct{}{},
		"mask":                struct{}{},
		"mesh":                struct{}{},
		"meshgradient":        struct{}{},
		"meshpatch":           struct{}{},
		"meshrow":             struct{}{},
		"metadata":            struct{}{},
		"mpath":               struct{}{},
		"path":                struct{}{},
		"pattern":             struct{}{},
		"polygon":             struct{}{},
		"polyline":            struct{}{},
		"radialGradient":      struct{}{},
		"rect":                struct{}{},
		"set":                 struct{}{},
		"solidcolor":          struct{}{},
		"stop":                struct{}{},
		"switch":              struct{}{},
		"symbol":              struct{}{},
		"text":                struct{}{},
		"textPath":            struct{}{},
		// "title":               struct{}{},
		"tspan":   struct{}{},
		"unknown": struct{}{},
		"use":     struct{}{},
		"view":    struct{}{},

		// math tags
		"annotation":     struct{}{},
		"annotation-xml": struct{}{},
		"maction":        struct{}{},
		"maligngroup":    struct{}{},
		"malignmark":     struct{}{},
		"math":           struct{}{},
		"menclose":       struct{}{},
		"merror":         struct{}{},
		"mfenced":        struct{}{},
		"mfrac":          struct{}{},
		"mfraction":      struct{}{},
		"mglyph":         struct{}{},
		"mi":             struct{}{},
		"mlabeledtr":     struct{}{},
		"mlongdiv":       struct{}{},
		"mmultiscripts":  struct{}{},
		"mn":             struct{}{},
		"mo":             struct{}{},
		"mover":          struct{}{},
		"mpadded":        struct{}{},
		"mphantom":       struct{}{},
		"mprescripts":    struct{}{},
		"mroot":          struct{}{},
		"mrow":           struct{}{},
		"ms":             struct{}{},
		"mscarries":      struct{}{},
		"mscarry":        struct{}{},
		"msgroup":        struct{}{},
		"msline":         struct{}{},
		"mspace":         struct{}{},
		"msqrt":          struct{}{},
		"msrow":          struct{}{},
		"mstack":         struct{}{},
		"mstyle":         struct{}{},
		"msub":           struct{}{},
		"msubsup":        struct{}{},
		"msup":           struct{}{},
		"mtable":         struct{}{},
		"mtd":            struct{}{},
		"mtext":          struct{}{},
		"mtr":            struct{}{},
		"munder":         struct{}{},
		"munderover":     struct{}{},
		"none":           struct{}{},
		"semantics":      struct{}{},
	}

	VoidTags = map[string]struct{}{
		"area":   struct{}{},
		"base":   struct{}{},
		"br":     struct{}{},
		"col":    struct{}{},
		"embed":  struct{}{},
		"hr":     struct{}{},
		"img":    struct{}{},
		"input":  struct{}{},
		"link":   struct{}{},
		"meta":   struct{}{},
		"param":  struct{}{},
		"source": struct{}{},
		"track":  struct{}{},
		"wbr":    struct{}{},
	}
)

func ParseTsAst(source string) *ast.SourceFile {
	file := parser.ParseSourceFile(ast.SourceFileParseOptions{
		FileName: "/virtual.tsx",
		Path:     tspath.Path("/virtual.tsx"),
		CompilerOptions: core.SourceFileAffectingCompilerOptions{
			BindInStrictMode: true,
		},
		ExternalModuleIndicatorOptions: ast.ExternalModuleIndicatorOptions{
			Force: true,
		},
		JSDocParsingMode: ast.JSDocParsingModeParseAll,
	}, source, core.ScriptKindTSX) // TODO: script kind
	// TODO: JSDiagnostics?
	if len(file.Diagnostics()) == 0 {
		binder.BindSourceFile(file)
	}
	return file
}
