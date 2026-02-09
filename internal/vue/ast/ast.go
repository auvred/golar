package vue_ast

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/core"
)

type Namespace uint8

const (
	NamespaceHTML Namespace = iota
	NamespaceSVG
	NamespaceMATH_ML
)

type NodeType uint16

const (
	KindRoot NodeType = iota
	KindElement
	KindText
	KindComment
	KindSimple_expression
	KindInterpolation
	KindAttribute
	KindDirective
)

type Node struct {
	Kind NodeType
	Loc  core.TextRange
	data nodeData
}

type nodeData interface {
	AsNode() *Node
}

func (n *Node) AsNode() *Node {
	return n
}

type ElementType uint8

const (
	ElementTypeELEMENT ElementType = iota
	ElementTypeCOMPONENT
	ElementTypeSLOT
	ElementTypeTEMPLATE
)

type RootNode struct {
	Node
	Children []*Node // TemplateChildNode[]
}

func NewRootNode() *RootNode {
	data := RootNode{Node: Node{Kind: KindRoot}}
	data.Node.data = &data
	return &data
}

type ElementNode struct {
	Node
	Ns  Namespace
	Tag string
	// TagType       ElementType
	Props         []*Node // Array<AttributeNode | DirectiveNode>
	Children      []*Node // TemplateChildNode[]
	IsSelfClosing bool
	// Only for <script>
	Ast *ast.SourceFile
	// Only for SFC root level elements
	InnerLoc core.TextRange
}

func NewElementNode(ns Namespace, tag string, loc core.TextRange) *ElementNode {
	data := ElementNode{Node: Node{Kind: KindElement, Loc: loc}, Ns: ns, Tag: tag}
	data.Node.data = &data
	return &data
}

type ScriptElementNode struct {
	ElementNode
}

type TextNode struct {
	Node
	Content string
}

func NewTextNode(content string, loc core.TextRange) *TextNode {
	data := TextNode{Node: Node{Kind: KindText, Loc: loc}, Content: content}
	data.Node.data = &data
	return &data
}

type SimpleExpressionNode struct {
	Node
	// TODO: right now we don't have simple identifier path
	// nil when expression is a simple identifier (static) or when empty
	Ast *ast.SourceFile
	// TODO: modify TS parser to parse expressions instead?
	PrefixLen int
	SuffixLen int
	// TODO
	// isHandlerKey?: boolean
}

func NewSimpleExpressionNode(ast *ast.SourceFile, loc core.TextRange, prefixLen, suffixLen int) *SimpleExpressionNode {
	data := SimpleExpressionNode{Node: Node{Kind: KindSimple_expression, Loc: loc}, Ast: ast, PrefixLen: prefixLen, SuffixLen: suffixLen}
	data.Node.data = &data
	return &data
}

type ForParseResult struct {
	Source *SimpleExpressionNode
	Value  *SimpleExpressionNode
	Key    *SimpleExpressionNode
	Index  *SimpleExpressionNode
}

type CommentNode struct {
	Node
	Content string
}

func NewCommentNode(content string, loc core.TextRange) *CommentNode {
	data := CommentNode{Node: Node{Kind: KindComment, Loc: loc}, Content: content}
	data.Node.data = &data
	return &data
}

type AttributeNode struct {
	Node
	Name    string
	NameLoc core.TextRange
	Value   *TextNode // | undefined
}

func NewAttributeNode(name string, nameLoc, loc core.TextRange) *AttributeNode {
	data := AttributeNode{Node: Node{Kind: KindAttribute, Loc: loc}, Name: name, NameLoc: nameLoc}
	data.Node.data = &data
	return &data
}

type DirectiveNode struct {
	Node
	// The normalized name without prefix or shorthands, e.g. "bind", "on"
	Name string
	// The raw attribute name, preserving shorthand, and including arg & modifiers
	// this is only used during parse.
	RawName string
	NameLoc core.TextRange
	// Nil when directive doesn't have expression
	Expression     *SimpleExpressionNode
	ForParseResult *ForParseResult
	IsStatic       bool
	Arg            string // TODO: support dynamic event names like @[event]="" *SimpleExpressionNode
	// modifiers: SimpleExpressionNode[]
}

func NewDirectiveNode(name, rawName string, nameLoc, loc core.TextRange) *DirectiveNode {
	data := DirectiveNode{Node: Node{Kind: KindDirective, Loc: loc}, Name: name, RawName: rawName, NameLoc: nameLoc}
	data.Node.data = &data
	return &data
}

type InterpolationNode struct {
	Node
	Content *SimpleExpressionNode
}

func NewInterpolationNode(content *SimpleExpressionNode, loc core.TextRange) *InterpolationNode {
	data := InterpolationNode{Node: Node{Kind: KindInterpolation, Loc: loc}, Content: content}
	data.Node.data = &data
	return &data
}

func (n *Node) AsElement() *ElementNode {
	return n.data.(*ElementNode)
}
func (n *Node) AsText() *TextNode {
	return n.data.(*TextNode)
}
func (n *Node) AsComment() *CommentNode {
	return n.data.(*CommentNode)
}
func (n *Node) AsSimpleExpression() *SimpleExpressionNode {
	return n.data.(*SimpleExpressionNode)
}
func (n *Node) AsInterpolation() *InterpolationNode {
	return n.data.(*InterpolationNode)
}
func (n *Node) AsAttribute() *AttributeNode {
	return n.data.(*AttributeNode)
}
func (n *Node) AsDirective() *DirectiveNode {
	return n.data.(*DirectiveNode)
}
