package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"slices"
	"strings"
	"unsafe"

	"golang.org/x/tools/go/packages"
)

const tsgoInternalPrefix = "github.com/microsoft/typescript-go/pkg/"
const tsgoAstSchemaPath = "thirdparty/typescript-go/_scripts/ast.json"
const rustGeneratedDir = "crates/golar/src"

// TODO
var sizes = types.SizesFor("gc", "amd64")

type childVisit struct {
	list bool
	call *ast.CallExpr
}

type fieldInfo struct {
	rawName    string
	getterName string
	rawType    string
	getterType string
	getterBody string
	skip       bool
}

type flagTypeSpec struct {
	typeName string
	file     *ast.File
	scope    *types.Scope
	stop     func(name string) bool
	enum     bool
}

type astSchema struct {
	Kinds schemaKinds           `json:"kinds"`
	Bases map[string]schemaBase `json:"bases"`
	Nodes schemaNodes           `json:"nodes"`
}

type schemaKinds struct {
	Elements []schemaKindElement        `json:"elements"`
	Markers  []schemaKindMarker         `json:"markers"`
	Aliases  map[string]json.RawMessage `json:"aliases"`
}

type schemaKindElement struct {
	Name    string `json:"name"`
	Comment string `json:"comment"`
}

func (e *schemaKindElement) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err == nil {
		e.Name = name
		return nil
	}

	type object schemaKindElement
	return json.Unmarshal(data, (*object)(e))
}

type schemaKindMarker struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type schemaNodes struct {
	Definitions map[string]schemaNodeDef   `json:"definitions"`
	Aliases     map[string]json.RawMessage `json:"aliases"`
	ListAliases map[string]string          `json:"listAliases"`
}

type schemaNodeDef struct {
	Kind                 schemaTypeNames       `json:"kind"`
	Extends              []string              `json:"extends"`
	Members              []schemaMember        `json:"members"`
	HandWritten          bool                  `json:"handWritten"`
	HandWrittenVisitor   bool                  `json:"handWrittenVisitor"`
	TypeParameters       []schemaTypeParameter `json:"typeParameters"`
	InstantiationAliases map[string]string     `json:"instantiationAliases"`
}

type schemaBase struct {
	Extends []string                   `json:"extends"`
	Fields  map[string]schemaBaseField `json:"fields"`
}

type schemaMember struct {
	Name      string          `json:"name"`
	Type      schemaTypeNames `json:"type"`
	List      string          `json:"list"`
	Inherited bool            `json:"inherited"`
	GoOnly    bool            `json:"goOnly"`
	NoFactory bool            `json:"noFactory"`
}

type schemaBaseField struct {
	Type      schemaTypeNames `json:"type"`
	List      string          `json:"list"`
	GoOnly    bool            `json:"goOnly"`
	NoFactory bool            `json:"noFactory"`
}

type schemaTypeParameter struct {
	Name       string `json:"name"`
	Constraint string `json:"constraint"`
	Default    string `json:"default"`
}

type schemaTypeNames []string

func (t *schemaTypeNames) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}

	var name string
	if err := json.Unmarshal(data, &name); err == nil {
		*t = []string{name}
		return nil
	}

	var names []string
	if err := json.Unmarshal(data, &names); err != nil {
		return err
	}
	*t = names
	return nil
}

type resolvedSchemaMember struct {
	name      string
	types     []string
	list      string
	goOnly    bool
	noFactory bool
}

func main() {
	schemaJSON, err := os.ReadFile(tsgoAstSchemaPath)
	if err != nil {
		panic(err)
	}
	var schema astSchema
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		panic(err)
	}

	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.LoadSyntax,
	}, tsgoInternalPrefix+"ast", tsgoInternalPrefix+"checker")
	if err != nil {
		panic(fmt.Sprintf("Error loading package: %v", err))
	}

	var astPkg *packages.Package
	var checkerPkg *packages.Package
	for _, pkg := range pkgs {
		switch pkg.PkgPath {
		case tsgoInternalPrefix + "ast":
			astPkg = pkg
		case tsgoInternalPrefix + "checker":
			checkerPkg = pkg
		}
	}
	if astPkg == nil {
		panic("could not load pkg/ast")
	}
	if checkerPkg == nil {
		panic("could not load pkg/checker")
	}

	astScope := astPkg.Types.Scope()
	nodeType := astScope.Lookup("Node").Type().(*types.Named)
	checkerScope := checkerPkg.Types.Scope()
	typeType := checkerScope.Lookup("Type").Type().(*types.Named)

	var symbolFlagsFile *ast.File
	var checkFlagsFile *ast.File
	var modifierFlagsFile *ast.File
	for i, file := range astPkg.Syntax {
		if strings.HasSuffix(astPkg.CompiledGoFiles[i], "pkg/ast/symbolflags.go") {
			symbolFlagsFile = file
		}
		if strings.HasSuffix(astPkg.CompiledGoFiles[i], "pkg/ast/checkflags.go") {
			checkFlagsFile = file
		}
		if strings.HasSuffix(astPkg.CompiledGoFiles[i], "pkg/ast/modifierflags.go") {
			modifierFlagsFile = file
		}
	}
	if symbolFlagsFile == nil {
		panic("could not locate pkg/ast/symbolflags.go")
	}
	if checkFlagsFile == nil {
		panic("could not locate pkg/ast/checkflags.go")
	}
	if modifierFlagsFile == nil {
		panic("could not locate pkg/ast/modifierflags.go")
	}

	var checkerTypesFile *ast.File
	for i, file := range checkerPkg.Syntax {
		if strings.HasSuffix(checkerPkg.CompiledGoFiles[i], "pkg/checker/types.go") {
			checkerTypesFile = file
			break
		}
	}
	if checkerTypesFile == nil {
		panic("could not locate pkg/checker/types.go")
	}

	var astOut bytes.Buffer
	var flagsOut bytes.Buffer
	var visitorCases bytes.Buffer
	var nodes bytes.Buffer
	var typesOut bytes.Buffer
	var checkerTypes bytes.Buffer

	astOut.WriteString(`// Code generated by tools/gen-rust-ast; DO NOT EDIT.
#![allow(non_snake_case)]

use std::marker;

use crate::common::*;
use crate::flags_generated::*;

`)
	typesOut.WriteString(`// Code generated by tools/gen-rust-ast; DO NOT EDIT.
#![allow(non_snake_case)]

use std::marker;

use crate::common::*;
use crate::flags_generated::*;

`)
	flagsOut.WriteString(`// Code generated by tools/gen-rust-ast; DO NOT EDIT.

`)

	writeFlagTypes(&flagsOut,
		flagTypeSpec{typeName: "SymbolFlags", file: symbolFlagsFile, scope: astScope},
		flagTypeSpec{typeName: "CheckFlags", file: checkFlagsFile, scope: astScope},
		flagTypeSpec{typeName: "ModifierFlags", file: modifierFlagsFile, scope: astScope},
		flagTypeSpec{typeName: "AccessFlags", file: checkerTypesFile, scope: checkerScope},
		flagTypeSpec{typeName: "TypeFlags", file: checkerTypesFile, scope: checkerScope},
		flagTypeSpec{typeName: "ObjectFlags", file: checkerTypesFile, scope: checkerScope},
		flagTypeSpec{typeName: "ElementFlags", file: checkerTypesFile, scope: checkerScope},
		flagTypeSpec{typeName: "IndexFlags", file: checkerTypesFile, scope: checkerScope},
		flagTypeSpec{typeName: "SignatureFlags", file: checkerTypesFile, scope: checkerScope},
		flagTypeSpec{typeName: "TypePredicateKind", file: checkerTypesFile, scope: checkerScope, enum: true},
	)
	fmt.Fprintf(&flagsOut, "#[repr(i16)]\n#[derive(Debug, Copy, Clone, PartialEq, Eq)]\npub enum Kind {\n")
	for _, entry := range schema.kindElements() {
		fmt.Fprintf(&flagsOut, "\t%s = %s,\n", rustKeyword(entry.name), entry.value)
	}
	flagsOut.WriteString("}\n\n")

	writeSchemaVisitorCases(&visitorCases, &schema)

	writeVisitor(&astOut, visitorCases.Bytes())

	for method := range nodeType.Methods() {
		methodName := method.Name()
		if !strings.HasPrefix(methodName, "As") || methodName == "AsNode" || methodName == "AsMutable" || methodName == "AsSyntheticExpression" {
			continue
		}

		returnTypePtr, ok := method.Signature().Results().At(0).Type().(*types.Pointer)
		if !ok {
			panic(methodName + " method can't be translated")
		}

		returnType := returnTypePtr.Elem().(*types.Named)
		nodeName := returnType.Obj().Name()
		nodeStruct := returnType.Underlying().(*types.Struct)

		kinds := schema.kindValuesForNode(nodeName)
		if len(kinds) == 0 {
			continue
		}

		var fields []fieldInfo
		nodeOffset, ok := embeddedNodeOffset(nodeStruct, 0)
		if !ok {
			panic("could not locate embedded Node for " + nodeName)
		}
		lastOffset := int64(0)
		padIndex := 0
		nodeEmitted := false
		seenRawFields := map[string]bool{"node": true}

		fmt.Fprintf(&nodes, "#[repr(C)]\n#[derive(Copy, Clone)]\npub struct Raw%v {\n", nodeName)

		emitPadding := func(offset int64, size types.Type) {
			padding := offset - lastOffset
			if padding < 0 {
				panic(fmt.Sprintf("negative padding for %v at offset %v after %v", nodeName, offset, lastOffset))
			}
			if padding != 0 {
				fmt.Fprintf(&nodes, "\tpub _pad%v: [u8; %v],\n", padIndex, padding)
				padIndex++
			}
			lastOffset = offset + sizes.Sizeof(size)
		}
		emitOpaqueField := func(offset int64, size types.Type) {
			padding := offset - lastOffset
			if padding < 0 {
				panic(fmt.Sprintf("negative opaque padding for %v at offset %v after %v", nodeName, offset, lastOffset))
			}
			if padding != 0 {
				fmt.Fprintf(&nodes, "\tpub _pad%v: [u8; %v],\n", padIndex, padding)
				padIndex++
			}
			fieldSize := sizes.Sizeof(size)
			if fieldSize != 0 {
				fmt.Fprintf(&nodes, "\tpub _pad%v: [u8; %v],\n", padIndex, fieldSize)
				padIndex++
			}
			lastOffset = offset + fieldSize
		}
		emitNode := func() {
			if nodeEmitted {
				return
			}
			padding := nodeOffset - lastOffset
			if padding < 0 {
				panic(fmt.Sprintf("embedded Node overlaps previous fields for %v", nodeName))
			}
			if padding != 0 {
				fmt.Fprintf(&nodes, "\tpub _pad%v: [u8; %v],\n", padIndex, padding)
				padIndex++
			}
			nodes.WriteString("\tpub node: RawNode,\n")
			lastOffset = nodeOffset + sizes.Sizeof(nodeType.Underlying())
			nodeEmitted = true
		}

		iterStructFields(nodeStruct, 0, func(field *types.Var) bool {
			if nodeName == "SourceFile" {
				if field.Embedded() {
					return true
				}
				switch field.Name() {
				case "text", "Statements", "EndOfFileToken":
				default:
					return false
				}
			}
			fieldName := field.Name()
			if fieldName == "Kind" || fieldName == "NextContainer" || fieldName == "compositeNodeBase" {
				return false
			}
			if field.Embedded() {
				fieldType := field.Type()
				name := fieldType.(*types.Named).Obj().Name()
				if name == "Node" {
					return false
				}
			}
			return true
		}, func(field *types.Var, offset int64) {
			if !nodeEmitted && offset >= nodeOffset {
				emitNode()
			}

			mapped, ok := mapField(field)
			if !ok {
				panic(fmt.Sprintf("ERROR: unknown field type for %v.%v: %v\n", nodeName, field.Name(), field.Type()))
			}
			if mapped.skip {
				emitOpaqueField(offset, field.Type())
				return
			}
			if seenRawFields[mapped.rawName] {
				emitOpaqueField(offset, field.Type())
				return
			}
			seenRawFields[mapped.rawName] = true

			emitPadding(offset, field.Type())
			fmt.Fprintf(&nodes, "\tpub %v: %v,\n", mapped.rawName, mapped.rawType)
			fields = append(fields, mapped)
		})
		emitNode()

		nodes.WriteString("}\n\n")

		fmt.Fprintf(&nodes, `#[repr(transparent)]
#[derive(Copy, Clone)]
pub struct %v<'a> {
	pub(crate) raw: *const Raw%v,
	_marker: marker::PhantomData<&'a ()>,
}

impl<'a> %v<'a> {
	#[inline(always)]
	pub fn as_node(self) -> Node<'a> {
		Node::from_raw(unsafe { std::ptr::addr_of!((*self.raw).node) })
	}
`, nodeName, nodeName, nodeName)
		for _, field := range fields {
			fmt.Fprintf(&nodes, `
pub fn %v(self) -> %v {
	%v
}
`, field.getterName, field.getterType, field.getterBody)
		}
		nodes.WriteString("}\n\n")

		var kindPattern strings.Builder
		for i, kind := range kinds {
			if i > 0 {
				kindPattern.WriteString(" | ")
			}
			fmt.Fprintf(&kindPattern, "Kind::%v", kind.name)
		}
		rawCastExpr := "node.raw.cast()"
		if nodeOffset != 0 {
			rawCastExpr = fmt.Sprintf("unsafe { node.raw.cast::<u8>().sub(%d).cast() }", nodeOffset)
		}
		fmt.Fprintf(&nodes, `impl<'a> FromNode<'a> for %v<'a> {
	#[inline(always)]
	fn matches(kind: Kind) -> bool {
		matches!(kind, %s)
	}

	#[inline(always)]
	unsafe fn from_node_unchecked(node: Node<'a>) -> Self {
		Self {
			raw: %s,
			_marker: marker::PhantomData,
		}
	}
}

		`, nodeName, kindPattern.String(), rawCastExpr)
	}

	writeCheckerTypes(&checkerTypes, checkerScope, typeType)

	astOut.Write(nodes.Bytes())
	typesOut.Write(checkerTypes.Bytes())

	const flagsGeneratedPath = rustGeneratedDir + "/flags_generated.rs"
	const astGeneratedPath = rustGeneratedDir + "/ast_generated.rs"
	const typesGeneratedPath = rustGeneratedDir + "/types_generated.rs"

	err = os.WriteFile(flagsGeneratedPath, flagsOut.Bytes(), 0o666)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(astGeneratedPath, astOut.Bytes(), 0o666)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(typesGeneratedPath, typesOut.Bytes(), 0o666)
	if err != nil {
		panic(err)
	}

	for _, path := range []string{flagsGeneratedPath, astGeneratedPath, typesGeneratedPath} {
		cmd := exec.Command("rustfmt", path)
		output, err := cmd.CombinedOutput()
		if err != nil {
			panic(fmt.Sprintf("rustfmt failed for %s: %v\n%s", path, err, output))
		}
	}
}

func writeVisitor(out *bytes.Buffer, cases []byte) {
	fmt.Fprintf(out, `pub fn visit_node_slice<'a, F>(f: &mut F, nodes: GoSlice<*const RawNode>) -> bool
where
	F: FnMut(Node<'a>) -> bool,
{
	for child in slice_iter::<RawNode, Node<'a>>(nodes) {
		if f(child) {
			return true;
		}
	}

	false
}

pub fn for_each_child<'a, F>(f: &mut F, node: Node<'a>) -> bool
where
	F: FnMut(Node<'a>) -> bool,
{
	match node.kind() {
%s		_ => false,
	}
}

`, string(cases))
}

func (schema *astSchema) kindElements() []kindValue {
	values := make([]kindValue, 0, len(schema.Kinds.Elements))
	for _, element := range schema.Kinds.Elements {
		if element.Name == "" {
			continue
		}
		values = append(values, kindValue{name: element.Name, value: fmt.Sprint(len(values))})
	}
	return values
}

func (schema *astSchema) kindValuesForNode(nodeName string) []kindValue {
	names := schema.kindNamesForNode(nodeName)
	if len(names) == 0 {
		return nil
	}

	valuesByName := make(map[string]string)
	for _, value := range schema.kindElements() {
		valuesByName[value.name] = value.value
	}

	values := make([]kindValue, 0, len(names))
	for _, name := range names {
		value, ok := valuesByName[name]
		if !ok {
			panic(fmt.Sprintf("missing kind %v for %v", name, nodeName))
		}
		values = append(values, kindValue{name: name, value: value})
	}
	return values
}

func (schema *astSchema) kindNamesForNode(nodeName string) []string {
	def, ok := schema.Nodes.Definitions[nodeName]
	if !ok {
		return nil
	}

	var names []string
	for _, member := range def.Members {
		if member.Name != "Kind" && member.Name != "kind" {
			continue
		}
		names = append(names, schema.kindNamesFromTypes(member.Type, def)...)
	}
	names = append(names, schema.kindNamesFromTypes(def.Kind, def)...)
	if len(names) == 0 {
		names = append(names, nodeName)
	}
	return uniqueStrings(names)
}

func (schema *astSchema) kindNamesFromTypes(types schemaTypeNames, node schemaNodeDef) []string {
	var names []string
	for _, typeName := range types {
		names = append(names, schema.kindNamesFromType(typeName, node)...)
	}
	return names
}

func (schema *astSchema) kindNamesFromType(typeName string, node schemaNodeDef) []string {
	typeName = normalizeTypeName(typeName)
	if typeName == "" || typeName == "Kind" {
		return nil
	}
	if strings.HasPrefix(typeName, "SyntaxKind.") {
		return []string{strings.TrimPrefix(typeName, "SyntaxKind.")}
	}
	for _, tp := range node.TypeParameters {
		if tp.Name == typeName {
			return schema.kindNamesFromType(tp.Constraint, node)
		}
	}
	if _, ok := schema.Kinds.Aliases[typeName]; ok {
		return schema.expandKindAlias(typeName)
	}
	if schema.hasKindElement(typeName) {
		return []string{typeName}
	}
	return nil
}

func (schema *astSchema) expandKindAlias(name string) []string {
	raw, ok := schema.Kinds.Aliases[name]
	if !ok {
		return []string{name}
	}

	var members []string
	if err := json.Unmarshal(raw, &members); err == nil {
		var expanded []string
		for _, member := range members {
			if _, ok := schema.Kinds.Aliases[member]; ok {
				expanded = append(expanded, schema.expandKindAlias(member)...)
			} else {
				expanded = append(expanded, member)
			}
		}
		return uniqueStrings(expanded)
	}

	var rangeAlias struct {
		Range [2]string `json:"range"`
	}
	if err := json.Unmarshal(raw, &rangeAlias); err != nil {
		panic(fmt.Sprintf("invalid kind alias %v: %v", name, err))
	}

	first := schema.resolveKindMarkerValue(rangeAlias.Range[0])
	last := schema.resolveKindMarkerValue(rangeAlias.Range[1])
	elements := schema.kindElements()
	firstIndex := -1
	lastIndex := -1
	for i, element := range elements {
		if element.name == first {
			firstIndex = i
		}
		if element.name == last {
			lastIndex = i
		}
	}
	if firstIndex < 0 || lastIndex < 0 || firstIndex > lastIndex {
		panic(fmt.Sprintf("invalid kind alias range %v: %v..%v", name, first, last))
	}

	names := make([]string, 0, lastIndex-firstIndex+1)
	for _, element := range elements[firstIndex : lastIndex+1] {
		names = append(names, element.name)
	}
	return names
}

func (schema *astSchema) resolveKindMarkerValue(name string) string {
	for _, marker := range schema.Kinds.Markers {
		if marker.Name == name {
			return schema.resolveKindMarkerValue(marker.Value)
		}
	}
	return name
}

func (schema *astSchema) hasKindElement(name string) bool {
	for _, element := range schema.Kinds.Elements {
		if element.Name == name {
			return true
		}
	}
	return false
}

func writeSchemaVisitorCases(out *bytes.Buffer, schema *astSchema) {
	names := make([]string, 0, len(schema.Nodes.Definitions))
	for name := range schema.Nodes.Definitions {
		names = append(names, name)
	}
	slices.Sort(names)

	for _, nodeName := range names {
		def := schema.Nodes.Definitions[nodeName]
		members := schema.schemaMembers(nodeName, def)
		var childMembers []resolvedSchemaMember
		for _, member := range members {
			if schema.isChildMember(member) {
				childMembers = append(childMembers, member)
			}
		}
		if len(childMembers) == 0 {
			continue
		}

		kindPattern := schema.kindPatternForNode(nodeName)
		if kindPattern == "" {
			continue
		}

		if def.HandWrittenVisitor {
			switch nodeName {
			case "JSDocParameterOrPropertyTag":
				fmt.Fprintf(out, `			%s => {
				let node = unsafe { JSDocParameterOrPropertyTag::from_node_unchecked(node) };
				if unsafe { (*node.raw).isNameFirst } != 0 {
					visit_node(f, unsafe { (*node.raw).tagName }) || visit_node(f, unsafe { (*node.raw).name }) || visit_node(f, unsafe { (*node.raw).typeExpression }) || visit_node_list(f, unsafe { (*node.raw).comment.cast() })
				} else {
					visit_node(f, unsafe { (*node.raw).tagName }) || visit_node(f, unsafe { (*node.raw).typeExpression }) || visit_node(f, unsafe { (*node.raw).name }) || visit_node_list(f, unsafe { (*node.raw).comment.cast() })
				}
			},
`, kindPattern)
			default:
				panic("unsupported hand-written visitor node " + nodeName)
			}
			continue
		}

		var visitExpr strings.Builder
		for i, member := range childMembers {
			if i > 0 {
				visitExpr.WriteString(" || ")
			}

			fieldName := rustFieldName(member.name)
			switch member.list {
			case "raw":
				fmt.Fprintf(&visitExpr, "visit_node_slice(f, unsafe { (*node.raw).%v })", fieldName)
			case "NodeList", "ModifierList":
				fmt.Fprintf(&visitExpr, "visit_node_list(f, unsafe { (*node.raw).%v.cast() })", fieldName)
			default:
				fmt.Fprintf(&visitExpr, "visit_node(f, unsafe { (*node.raw).%v })", fieldName)
			}
		}

		fmt.Fprintf(out, `			%s => {
				let node = unsafe { %v::from_node_unchecked(node) };
				%s
			},
`, kindPattern, nodeName, visitExpr.String())
	}
}

func (schema *astSchema) kindPatternForNode(nodeName string) string {
	kinds := schema.kindValuesForNode(nodeName)
	var kindPattern strings.Builder
	for i, kind := range kinds {
		if i > 0 {
			kindPattern.WriteString(" | ")
		}
		fmt.Fprintf(&kindPattern, "Kind::%v", kind.name)
	}
	return kindPattern.String()
}

func (schema *astSchema) schemaMembers(nodeName string, def schemaNodeDef) []resolvedSchemaMember {
	members := make([]resolvedSchemaMember, 0, len(def.Members))
	for _, member := range def.Members {
		resolved := resolvedSchemaMember{
			name:      member.Name,
			types:     member.Type,
			list:      member.List,
			goOnly:    member.GoOnly,
			noFactory: member.NoFactory,
		}
		if member.Inherited {
			if inherited, ok := schema.inheritedField(def, member.Name); ok {
				if len(resolved.types) == 0 {
					resolved.types = inherited.Type
				}
				if resolved.list == "" {
					resolved.list = inherited.List
				}
				resolved.goOnly = resolved.goOnly || inherited.GoOnly
				resolved.noFactory = resolved.noFactory || inherited.NoFactory
			}
		}
		if resolved.goOnly || resolved.noFactory || schema.isKindMember(resolved) {
			continue
		}
		members = append(members, resolved)
	}
	return members
}

func (schema *astSchema) inheritedField(def schemaNodeDef, fieldName string) (schemaBaseField, bool) {
	for _, baseName := range def.Extends {
		if field, ok := schema.inheritedFieldFromBase(baseName, fieldName); ok {
			return field, true
		}
	}
	return schemaBaseField{}, false
}

func (schema *astSchema) inheritedFieldFromBase(baseName, fieldName string) (schemaBaseField, bool) {
	base, ok := schema.Bases[baseName]
	if !ok {
		return schemaBaseField{}, false
	}
	if field, ok := base.Fields[fieldName]; ok {
		return field, true
	}
	for _, parentName := range base.Extends {
		if field, ok := schema.inheritedFieldFromBase(parentName, fieldName); ok {
			return field, true
		}
	}
	return schemaBaseField{}, false
}

func (schema *astSchema) isKindMember(member resolvedSchemaMember) bool {
	if member.name != "Kind" && member.name != "kind" {
		return false
	}
	return schema.baseKind(member.types, schemaNodeDef{}) == "kind"
}

func (schema *astSchema) isChildMember(member resolvedSchemaMember) bool {
	if member.list != "" {
		return schema.baseKind(member.types, schemaNodeDef{}) == "node"
	}
	return schema.baseKind(member.types, schemaNodeDef{}) == "node"
}

func (schema *astSchema) baseKind(types []string, node schemaNodeDef) string {
	hasNode := false
	hasKind := false
	for _, typeName := range types {
		switch schema.baseKindOfType(typeName, node) {
		case "node", "list":
			hasNode = true
		case "kind":
			hasKind = true
		}
	}
	if hasNode {
		return "node"
	}
	if hasKind {
		return "kind"
	}
	return "primitive"
}

func (schema *astSchema) baseKindOfType(typeName string, node schemaNodeDef) string {
	typeName = normalizeTypeName(typeName)
	if typeName == "" {
		return "primitive"
	}
	if strings.HasPrefix(typeName, "SyntaxKind.") || typeName == "Kind" {
		return "kind"
	}
	for _, tp := range node.TypeParameters {
		if tp.Name == typeName {
			return schema.baseKindOfType(tp.Constraint, node)
		}
	}
	if _, ok := schema.Kinds.Aliases[typeName]; ok {
		return "kind"
	}
	if schema.hasKindElement(typeName) {
		return "kind"
	}
	if _, ok := schema.Nodes.Definitions[typeName]; ok {
		return "node"
	}
	if _, ok := schema.Bases[typeName]; ok {
		return "node"
	}
	if _, ok := schema.Nodes.Aliases[typeName]; ok {
		return "node"
	}
	if _, ok := schema.Nodes.ListAliases[typeName]; ok {
		return "list"
	}
	for _, def := range schema.Nodes.Definitions {
		if _, ok := def.InstantiationAliases[typeName]; ok {
			return "node"
		}
	}
	return "primitive"
}

func normalizeTypeName(typeName string) string {
	typeName = strings.TrimPrefix(typeName, "*")
	return strings.TrimSpace(typeName)
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		unique = append(unique, value)
	}
	return unique
}

func writeCheckerTypes(out *bytes.Buffer, scope *types.Scope, typeType *types.Named) {
	for method := range typeType.Methods() {
		methodName := method.Name()
		if !strings.HasPrefix(methodName, "As") || methodName == "AsType" {
			continue
		}

		returnTypePtr, ok := method.Signature().Results().At(0).Type().(*types.Pointer)
		if !ok {
			panic(methodName + " method can't be translated")
		}

		returnType := returnTypePtr.Elem().(*types.Named)
		typeName := returnType.Obj().Name()
		rawName := "Raw" + typeName
		wrapperName := typeName
		typeStruct := returnType.Underlying().(*types.Struct)
		matchExpr, ok := typeMatchExpr(typeName)
		if !ok {
			panic("unknown checker type match for " + typeName)
		}

		var fields []fieldInfo
		lastOffset := sizes.Sizeof(typeType.Underlying())
		padIndex := 0

		fmt.Fprintf(out, "#[repr(C)]\n#[derive(Copy, Clone)]\npub struct %v {\n\tpub base: RawType,\n", rawName)

		emitPadding := func(offset int64, size types.Type) {
			padding := offset - lastOffset
			if padding != 0 {
				fmt.Fprintf(out, "\tpub _pad%v: [u8; %v],\n", padIndex, padding)
				padIndex++
			}
			lastOffset = offset + sizes.Sizeof(size)
		}

		iterStructFields(typeStruct, 0, func(field *types.Var) bool {
			if field.Embedded() {
				switch embedded := field.Type().(type) {
				case *types.Named:
					return embedded.Obj().Name() != "Type"
				case *types.Alias:
					actual, ok := types.Unalias(embedded).(*types.Named)
					return !ok || actual.Obj().Name() != "Type"
				}
			}
			return true
		}, func(field *types.Var, offset int64) {
			mapped, ok := mapField(field)
			if !ok {
				panic(fmt.Sprintf("unknown field type for %v.%v: %v", typeName, field.Name(), field.Type()))
			}
			if mapped.skip {
				return
			}

			emitPadding(offset, field.Type())
			fmt.Fprintf(out, "\tpub %v: %v,\n", mapped.rawName, mapped.rawType)
			fields = append(fields, mapped)
		})

		out.WriteString("}\n\n")

		fmt.Fprintf(out, `#[repr(transparent)]
#[derive(Copy, Clone)]
pub struct %v<'a> {
	pub(crate) raw: *const %v,
	_marker: marker::PhantomData<&'a ()>,
}

impl<'a> %v<'a> {
	#[inline(always)]
	pub fn as_type(self) -> Type<'a> {
		Type::from_raw(self.raw.cast())
	}
`, wrapperName, rawName, wrapperName)
		for _, field := range fields {
			fmt.Fprintf(out, `
pub fn %v(self) -> %v {
	%v
}
`, field.getterName, field.getterType, field.getterBody)
		}
		out.WriteString("}\n\n")

		fmt.Fprintf(out, `impl<'a> FromType<'a> for %v<'a> {
	#[inline(always)]
	fn matches(flags: TypeFlags, _object_flags: ObjectFlags) -> bool {
		%v
	}

	#[inline(always)]
	unsafe fn from_type_unchecked(t: Type<'a>) -> Self {
		Self {
			raw: t.raw.cast(),
			_marker: marker::PhantomData,
		}
	}
}

		`, wrapperName, matchExpr)
	}
}

func typeMatchExpr(typeName string) (string, bool) {
	typeFlags := func(name string) string {
		return "TypeFlags::" + rustConstName(name)
	}
	objectFlags := func(name string) string {
		return "ObjectFlags::" + rustConstName(name)
	}

	switch typeName {
	case "IntrinsicType":
		return "flags.intersects(" + typeFlags("Intrinsic") + ")", true
	case "LiteralType":
		return "flags.intersects(" + typeFlags("Freshable") + ")", true
	case "UniqueESSymbolType":
		return "flags.intersects(" + typeFlags("UniqueESSymbol") + ")", true
	case "TupleType":
		return "flags.intersects(" + typeFlags("Object") + ") && _object_flags.intersects(" + objectFlags("Tuple") + ")", true
	case "InstantiationExpressionType":
		return "flags.intersects(" + typeFlags("Object") + ") && _object_flags.intersects(" + objectFlags("InstantiationExpressionType") + ")", true
	case "MappedType":
		return "flags.intersects(" + typeFlags("Object") + ") && _object_flags.intersects(" + objectFlags("Mapped") + ")", true
	case "ReverseMappedType":
		return "flags.intersects(" + typeFlags("Object") + ") && _object_flags.intersects(" + objectFlags("ReverseMapped") + ")", true
	case "EvolvingArrayType":
		return "flags.intersects(" + typeFlags("Object") + ") && _object_flags.intersects(" + objectFlags("EvolvingArray") + ")", true
	case "TypeParameter":
		return "flags.intersects(" + typeFlags("TypeParameter") + ")", true
	case "UnionType":
		return "flags.intersects(" + typeFlags("Union") + ")", true
	case "IntersectionType":
		return "flags.intersects(" + typeFlags("Intersection") + ")", true
	case "IndexType":
		return "flags.intersects(" + typeFlags("Index") + ")", true
	case "IndexedAccessType":
		return "flags.intersects(" + typeFlags("IndexedAccess") + ")", true
	case "TemplateLiteralType":
		return "flags.intersects(" + typeFlags("TemplateLiteral") + ")", true
	case "StringMappingType":
		return "flags.intersects(" + typeFlags("StringMapping") + ")", true
	case "SubstitutionType":
		return "flags.intersects(" + typeFlags("Substitution") + ")", true
	case "ConditionalType":
		return "flags.intersects(" + typeFlags("Conditional") + ")", true
	case "ConstrainedType":
		return "flags.intersects(" + typeFlags("StructuredOrInstantiable") + ")", true
	case "StructuredType":
		return "flags.intersects(" + typeFlags("StructuredType") + ")", true
	case "ObjectType":
		return "flags.intersects(" + typeFlags("Object") + ")", true
	case "TypeReference":
		return "flags.intersects(" + typeFlags("Object") + ") && _object_flags.intersects(" + objectFlags("Reference") + " | " + objectFlags("Tuple") + " | " + objectFlags("ClassOrInterface") + ")", true
	case "InterfaceType":
		return "flags.intersects(" + typeFlags("Object") + ") && _object_flags.intersects(" + objectFlags("ClassOrInterface") + " | " + objectFlags("Tuple") + ")", true
	case "UnionOrIntersectionType":
		return "flags.intersects(" + typeFlags("UnionOrIntersection") + ")", true
	default:
		return "", false
	}
}

func collectVisitCalls(node ast.Expr, visits *[]childVisit) {
	switch node := node.(type) {
	case *ast.CallExpr:
		if id, ok := node.Fun.(*ast.Ident); ok {
			visit := childVisit{call: node}
			switch id.Name {
			case "visitNodeList", "visitModifiers":
				visit.list = true
			case "visit":
			default:
				return
			}
			*visits = append(*visits, visit)
		}
	case *ast.BinaryExpr:
		collectVisitCalls(node.X, visits)
		collectVisitCalls(node.Y, visits)
	}
}

func writeFlagTypes(out *bytes.Buffer, specs ...flagTypeSpec) {
	for _, spec := range specs {
		typeObj := spec.scope.Lookup(spec.typeName)
		if typeObj == nil {
			panic("missing flag type " + spec.typeName)
		}

		basic, ok := typeObj.Type().Underlying().(*types.Basic)
		if !ok {
			panic(spec.typeName + " is not a basic type")
		}

		underlying, ok := rustBasicType(basic.Kind())
		if !ok {
			panic(fmt.Sprintf("unsupported underlying type for %v: %v", spec.typeName, basic.Kind()))
		}

		var entries []kindValue
		for _, decl := range spec.file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.CONST {
				continue
			}

			for _, item := range genDecl.Specs {
				valueSpec, ok := item.(*ast.ValueSpec)
				if !ok {
					continue
				}

				for _, name := range valueSpec.Names {
					if !strings.HasPrefix(name.Name, spec.typeName) {
						continue
					}

					obj := spec.scope.Lookup(name.Name)
					if obj == nil {
						panic("missing constant " + name.Name)
					}
					entries = append(entries, kindValue{
						name:  strings.TrimPrefix(name.Name, spec.typeName),
						value: obj.(*types.Const).Val().String(),
					})
				}
			}
		}

		if len(entries) == 0 {
			panic("no constants found for " + spec.typeName)
		}

		if spec.enum {
			fmt.Fprintf(out, "#[repr(%s)]\n#[derive(Debug, Copy, Clone, PartialEq, Eq)]\npub enum %s {\n", underlying, spec.typeName)
			for _, entry := range entries {
				if spec.stop != nil && spec.stop(entry.name) {
					break
				}
				fmt.Fprintf(out, "\t%s = %s,\n", rustKeyword(entry.name), entry.value)
			}
			out.WriteString("}\n\n")
			continue
		}

		fmt.Fprintf(out, `#[repr(transparent)]
#[derive(Debug, Copy, Clone, PartialEq, Eq, Hash, Default)]
pub struct %s(pub %s);

impl %s {
`, spec.typeName, underlying, spec.typeName)
		for _, entry := range entries {
			fmt.Fprintf(out, "\tpub const %s: Self = Self(%s);\n", rustConstName(entry.name), entry.value)
		}
		fmt.Fprintf(out, `
	#[inline(always)]
	pub const fn contains(self, other: Self) -> bool {
		(self.0 & other.0) == other.0
	}

	#[inline(always)]
	pub const fn intersects(self, other: Self) -> bool {
		(self.0 & other.0) != 0
	}
}

impl From<%s> for %s {
	fn from(value: %s) -> Self {
		value.0
	}
}

impl From<%s> for %s {
	fn from(value: %s) -> Self {
		Self(value)
	}
}

impl std::ops::BitOr for %s {
	type Output = Self;

	fn bitor(self, rhs: Self) -> Self::Output {
		Self(self.0 | rhs.0)
	}
}

impl std::ops::BitOrAssign for %s {
	fn bitor_assign(&mut self, rhs: Self) {
		self.0 |= rhs.0;
	}
}

impl std::ops::BitAnd for %s {
	type Output = Self;

	fn bitand(self, rhs: Self) -> Self::Output {
		Self(self.0 & rhs.0)
	}
}

impl std::ops::BitAndAssign for %s {
	fn bitand_assign(&mut self, rhs: Self) {
		self.0 &= rhs.0;
	}
}

`, spec.typeName, underlying, spec.typeName, underlying, spec.typeName, underlying, spec.typeName, spec.typeName, spec.typeName, spec.typeName)
	}
}

type kindValue struct {
	name  string
	value string
}

func kindValuesForNode(scope *types.Scope, nodeName string) []kindValue {
	names := kindNamesForNode(nodeName)
	if len(names) == 0 {
		return nil
	}

	values := make([]kindValue, 0, len(names))
	for _, name := range names {
		obj := scope.Lookup("Kind" + name)
		if obj == nil {
			panic(fmt.Sprintf("missing kind constant Kind%v for %v", name, nodeName))
		}
		values = append(values, kindValue{name: name, value: obj.(*types.Const).Val().String()})
	}

	return values
}

func kindNamesForNode(nodeName string) []string {
	switch nodeName {
	case "JSDocTagBase", "JSDocCommentBase":
		return nil
	case "FlowSwitchClauseData", "FlowReduceLabelData":
		return nil
	case "TypeAssertion":
		return []string{"TypeAssertionExpression"}
	case "ForInOrOfStatement":
		return []string{"ForInStatement", "ForOfStatement"}
	case "TypeReferenceNode", "ConditionalTypeNode", "InferTypeNode", "ImportTypeNode", "LiteralTypeNode", "ThisTypeNode", "ParenthesizedTypeNode", "TypePredicateNode", "TypeOperatorNode", "MappedTypeNode", "ArrayTypeNode", "TupleTypeNode", "UnionTypeNode", "IntersectionTypeNode", "RestTypeNode", "OptionalTypeNode", "TemplateLiteralTypeNode", "FunctionTypeNode", "ConstructorTypeNode", "TypeQueryNode", "TypeLiteralNode", "IndexedAccessTypeNode":
		return []string{strings.TrimSuffix(nodeName, "Node")}
	case "ConstructorDeclaration", "ParameterDeclaration", "TypeParameterDeclaration", "PropertySignatureDeclaration", "MethodSignatureDeclaration", "GetAccessorDeclaration", "SetAccessorDeclaration", "CallSignatureDeclaration", "ConstructSignatureDeclaration", "IndexSignatureDeclaration":
		return []string{strings.TrimSuffix(nodeName, "Declaration")}
	case "CaseOrDefaultClause":
		return []string{"CaseClause", "DefaultClause"}
	case "KeywordTypeNode":
		return []string{"AnyKeyword", "UnknownKeyword", "NumberKeyword", "BigIntKeyword", "ObjectKeyword", "BooleanKeyword", "StringKeyword", "SymbolKeyword", "VoidKeyword", "UndefinedKeyword", "NeverKeyword", "IntrinsicKeyword"}
	case "KeywordExpression":
		return []string{"ThisKeyword", "SuperKeyword", "ImportKeyword"}
	case "BindingPattern":
		return []string{"ObjectBindingPattern", "ArrayBindingPattern"}
	case "JSDocUnknownTag":
		return []string{"JSDocTag"}
	case "JSDocParameterOrPropertyTag":
		return []string{"JSDocParameterTag", "JSDocPropertyTag"}
	default:
		return []string{nodeName}
	}
}

func mapField(field *types.Var) (fieldInfo, bool) {
	rawName := rustFieldName(field.Name())
	getterName := rustGetterName(field.Name())
	rawField := "unsafe { (*self.raw)." + rawName + " }"

	switch t := field.Type().(type) {
	case *types.Basic:
		switch t.Kind() {
		case types.String:
			return fieldInfo{
				rawName:    rawName,
				getterName: getterName,
				rawType:    "GoString",
				getterType: "&'a str",
				getterBody: "unsafe { (*self.raw)." + rawName + ".as_str() }",
			}, true
		case types.Bool:
			return fieldInfo{
				rawName:    rawName,
				getterName: getterName,
				rawType:    "u8",
				getterType: "bool",
				getterBody: "unsafe { (*self.raw)." + rawName + " != 0 }",
			}, true
		default:
			rawType, ok := rustBasicType(t.Kind())
			if !ok {
				return fieldInfo{}, false
			}
			return fieldInfo{
				rawName:    rawName,
				getterName: getterName,
				rawType:    rawType,
				getterType: rawType,
				getterBody: rawField,
			}, true
		}
	case *types.Pointer:
		return mapPointerField(rawName, getterName, t.Elem())
	case *types.Named:
		if rawType, ok := rustNamedType(t); ok {
			return fieldInfo{
				rawName:    rawName,
				getterName: getterName,
				rawType:    rawType,
				getterType: rawType,
				getterBody: rawField,
			}, true
		}
		switch t.Obj().Name() {
		case "Uint32":
			if t.Obj().Pkg() != nil && t.Obj().Pkg().Path() == "sync/atomic" {
				return fieldInfo{skip: true}, true
			}
			return fieldInfo{}, false
		case "SymbolTable", "TokenFlags":
			return fieldInfo{skip: true}, true
		default:
			return fieldInfo{}, false
		}
	case *types.Alias:
		switch actual := types.Unalias(t).(type) {
		case *types.Interface:
			return fieldInfo{
				rawName:    rawName,
				getterName: getterName,
				rawType:    "GoIface",
				getterType: "GoIface",
				getterBody: rawField,
			}, true
		case *types.Named:
			if rawType, ok := rustNamedType(actual); ok {
				return fieldInfo{
					rawName:    rawName,
					getterName: getterName,
					rawType:    rawType,
					getterType: rawType,
					getterBody: rawField,
				}, true
			}
			return fieldInfo{}, false
		default:
			return fieldInfo{}, false
		}
	case *types.Interface:
		return fieldInfo{
			rawName:    rawName,
			getterName: getterName,
			rawType:    "GoIface",
			getterType: "GoIface",
			getterBody: rawField,
		}, true
	case *types.Slice:
		if basic, ok := t.Elem().(*types.Basic); ok {
			switch basic.Kind() {
			case types.String:
				return fieldInfo{
					rawName:    rawName,
					getterName: getterName,
					rawType:    "GoSlice<GoString>",
					getterType: "GoSlice<GoString>",
					getterBody: rawField,
				}, true
			default:
				rawType, ok := rustBasicType(basic.Kind())
				if !ok {
					return fieldInfo{}, false
				}
				return fieldInfo{
					rawName:    rawName,
					getterName: getterName,
					rawType:    "GoSlice<" + rawType + ">",
					getterType: "GoSlice<" + rawType + ">",
					getterBody: rawField,
				}, true
			}
		}

		if named, ok := t.Elem().(*types.Named); ok {
			if rawType, ok := rustNamedType(named); ok {
				return fieldInfo{
					rawName:    rawName,
					getterName: getterName,
					rawType:    "GoSlice<" + rawType + ">",
					getterType: "GoSlice<" + rawType + ">",
					getterBody: rawField,
				}, true
			}
			if named.Obj().Name() == "TupleElementInfo" {
				return fieldInfo{
					rawName:    rawName,
					getterName: getterName,
					rawType:    "GoSlice<RawTupleElementInfo>",
					getterType: "GoSlice<RawTupleElementInfo>",
					getterBody: rawField,
				}, true
			}
		}

		elemPtr, ok := t.Elem().(*types.Pointer)
		if !ok {
			return fieldInfo{}, false
		}

		elemInfo, ok := rustPointerElemSliceInfo(elemPtr.Elem())
		if !ok {
			return fieldInfo{}, false
		}

		return fieldInfo{
			rawName:    rawName,
			getterName: getterName,
			rawType:    "GoSlice<*const " + elemInfo.rawType + ">",
			getterType: "SliceIter<'a, " + elemInfo.rawType + ", " + elemInfo.wrapperType + ">",
			getterBody: "slice_iter(" + rawField + ")",
		}, true
	case *types.Map:
		return fieldInfo{skip: true}, true
	default:
		return fieldInfo{}, false
	}
}

func mapPointerField(rawName, getterName string, elem types.Type) (fieldInfo, bool) {
	switch elem := elem.(type) {
	case *types.Named:
		return mapNamedPointerField(rawName, getterName, elem)
	case *types.Alias:
		actual, ok := types.Unalias(elem).(*types.Named)
		if !ok {
			return fieldInfo{}, false
		}
		return mapNamedPointerField(rawName, getterName, actual)
	default:
		return fieldInfo{}, false
	}
}

func mapNamedPointerField(rawName, getterName string, elem *types.Named) (fieldInfo, bool) {
	rawField := "unsafe { (*self.raw)." + rawName + " }"
	elemName := elem.Obj().Name()
	pkgPath := ""
	if elem.Obj().Pkg() != nil {
		pkgPath = elem.Obj().Pkg().Path()
	}

	switch elemName {
	case "FlowNode", "TypeMapper", "Checker":
		return fieldInfo{skip: true}, true
	case "Symbol":
		return fieldInfo{
			rawName:    rawName,
			getterName: getterName,
			rawType:    "*const RawSymbol",
			getterType: "Option<Symbol<'a>>",
			getterBody: "symbol_from_raw(" + rawField + ")",
		}, true
	case "Node":
		return fieldInfo{
			rawName:    rawName,
			getterName: getterName,
			rawType:    "*const RawNode",
			getterType: "Option<Node<'a>>",
			getterBody: "node_from_raw(" + rawField + ")",
		}, true
	case "NodeList":
		return fieldInfo{
			rawName:    rawName,
			getterName: getterName,
			rawType:    "*const RawNodeList",
			getterType: "Option<NodeList<'a>>",
			getterBody: "node_list_from_raw(" + rawField + ")",
		}, true
	case "ModifierList":
		return fieldInfo{
			rawName:    rawName,
			getterName: getterName,
			rawType:    "*const RawModifierList",
			getterType: "Option<ModifierList<'a>>",
			getterBody: "modifier_list_from_raw(" + rawField + ")",
		}, true
	case "Type":
		return fieldInfo{
			rawName:    rawName,
			getterName: getterName,
			rawType:    "*const RawType",
			getterType: "Option<Type<'a>>",
			getterBody: "type_from_raw(" + rawField + ")",
		}, true
	case "TypeAlias":
		return fieldInfo{
			rawName:    rawName,
			getterName: getterName,
			rawType:    "*const RawTypeAlias",
			getterType: "Option<TypeAlias<'a>>",
			getterBody: "type_alias_from_raw(" + rawField + ")",
		}, true
	case "Signature":
		return fieldInfo{
			rawName:    rawName,
			getterName: getterName,
			rawType:    "*const RawSignature",
			getterType: "Option<Signature<'a>>",
			getterBody: "signature_from_raw(" + rawField + ")",
		}, true
	case "TypePredicate":
		return fieldInfo{
			rawName:    rawName,
			getterName: getterName,
			rawType:    "*const RawCheckerTypePredicate",
			getterType: "Option<CheckerTypePredicate<'a>>",
			getterBody: "type_predicate_from_raw(" + rawField + ")",
		}, true
	case "IndexInfo":
		return fieldInfo{
			rawName:    rawName,
			getterName: getterName,
			rawType:    "*const RawIndexInfo",
			getterType: "Option<IndexInfo<'a>>",
			getterBody: "index_info_from_raw(" + rawField + ")",
		}, true
	case "ConditionalRoot":
		return fieldInfo{
			rawName:    rawName,
			getterName: getterName,
			rawType:    "*const RawConditionalRoot",
			getterType: "Option<ConditionalRoot<'a>>",
			getterBody: "conditional_root_from_raw(" + rawField + ")",
		}, true
	case "CompositeSignature":
		return fieldInfo{
			rawName:    rawName,
			getterName: getterName,
			rawType:    "*const RawCompositeSignature",
			getterType: "Option<CompositeSignature<'a>>",
			getterBody: "composite_signature_from_raw(" + rawField + ")",
		}, true
	default:
		if pkgPath == tsgoInternalPrefix+"ast" && strings.HasSuffix(elemName, "Node") {
			return fieldInfo{
				rawName:    rawName,
				getterName: getterName,
				rawType:    "*const RawNode",
				getterType: "Option<Node<'a>>",
				getterBody: "node_from_raw(" + rawField + ")",
			}, true
		}
		return fieldInfo{}, false
	}
}

func rustBasicType(kind types.BasicKind) (string, bool) {
	switch kind {
	case types.Int:
		return "isize", true
	case types.Int8:
		return "i8", true
	case types.Int16:
		return "i16", true
	case types.Int32:
		return "i32", true
	case types.Int64:
		return "i64", true
	case types.Uint:
		return "usize", true
	case types.Uint8:
		return "u8", true
	case types.Uint16:
		return "u16", true
	case types.Uint32:
		return "u32", true
	case types.Uint64:
		return "u64", true
	case types.Uintptr:
		return "usize", true
	case types.Float32:
		return "f32", true
	case types.Float64:
		return "f64", true
	default:
		return "", false
	}
}

func rustNamedType(named *types.Named) (string, bool) {
	switch named.Obj().Name() {
	case "Kind", "SymbolFlags", "CheckFlags", "ModifierFlags", "TypeFlags", "ObjectFlags", "TypeId", "SignatureFlags", "IndexFlags", "AccessFlags", "ElementFlags", "TypePredicateKind":
		return named.Obj().Name(), true
	default:
		return "", false
	}
}

type rustPointerElemInfo struct {
	rawType     string
	wrapperType string
}

func rustPointerElemSliceInfo(elem types.Type) (rustPointerElemInfo, bool) {
	var named *types.Named
	switch elem := elem.(type) {
	case *types.Named:
		named = elem
	case *types.Alias:
		actual, ok := types.Unalias(elem).(*types.Named)
		if !ok {
			return rustPointerElemInfo{}, false
		}
		named = actual
	default:
		return rustPointerElemInfo{}, false
	}

	name := named.Obj().Name()
	pkgPath := ""
	if named.Obj().Pkg() != nil {
		pkgPath = named.Obj().Pkg().Path()
	}

	switch name {
	case "Node":
		return rustPointerElemInfo{rawType: "RawNode", wrapperType: "Node<'a>"}, true
	case "NodeList":
		return rustPointerElemInfo{rawType: "RawNodeList", wrapperType: "NodeList<'a>"}, true
	case "ModifierList":
		return rustPointerElemInfo{rawType: "RawModifierList", wrapperType: "ModifierList<'a>"}, true
	case "Symbol":
		return rustPointerElemInfo{rawType: "RawSymbol", wrapperType: "Symbol<'a>"}, true
	case "Type":
		return rustPointerElemInfo{rawType: "RawType", wrapperType: "Type<'a>"}, true
	case "TypeAlias":
		return rustPointerElemInfo{rawType: "RawTypeAlias", wrapperType: "TypeAlias<'a>"}, true
	case "Signature":
		return rustPointerElemInfo{rawType: "RawSignature", wrapperType: "Signature<'a>"}, true
	case "TypePredicate":
		return rustPointerElemInfo{rawType: "RawCheckerTypePredicate", wrapperType: "CheckerTypePredicate<'a>"}, true
	case "IndexInfo":
		return rustPointerElemInfo{rawType: "RawIndexInfo", wrapperType: "IndexInfo<'a>"}, true
	case "ConditionalRoot":
		return rustPointerElemInfo{rawType: "RawConditionalRoot", wrapperType: "ConditionalRoot<'a>"}, true
	case "CompositeSignature":
		return rustPointerElemInfo{rawType: "RawCompositeSignature", wrapperType: "CompositeSignature<'a>"}, true
	default:
		if pkgPath == tsgoInternalPrefix+"ast" && strings.HasSuffix(name, "Node") {
			return rustPointerElemInfo{rawType: "RawNode", wrapperType: "Node<'a>"}, true
		}
		return rustPointerElemInfo{}, false
	}
}

func rustFieldName(name string) string {
	if name == "" {
		return name
	}

	if name[0] >= 'A' && name[0] <= 'Z' {
		name = strings.ToLower(name[:1]) + name[1:]
	}

	return rustKeyword(name)
}

func rustGetterName(name string) string {
	if name == "" {
		return name
	}

	var out strings.Builder
	for i := range len(name) {
		c := name[i]
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				prev := name[i-1]
				hasNext := i+1 < len(name)
				nextIsLower := hasNext && name[i+1] >= 'a' && name[i+1] <= 'z'
				if (prev >= 'a' && prev <= 'z') || (prev >= '0' && prev <= '9') || (prev >= 'A' && prev <= 'Z' && nextIsLower) {
					out.WriteByte('_')
				}
			}
			out.WriteByte(c + ('a' - 'A'))
			continue
		}
		out.WriteByte(c)
	}

	return rustKeyword(out.String())
}

func rustConstName(name string) string {
	name = strings.TrimPrefix(rustGetterName(name), "r_")
	return strings.ToUpper(name)
}

func rustKeyword(name string) string {
	switch name {
	case "type":
		return "r_" + name
	default:
		return name
	}
}

func embeddedNodeOffset(s *types.Struct, offset int64) (int64, bool) {
	fieldsCount := s.NumFields()
	for fieldIdx := range fieldsCount {
		field := s.Field(fieldIdx)
		fieldAlign := sizes.Alignof(field.Type())
		offset = align(offset, fieldAlign)
		fieldSize := sizes.Sizeof(field.Type())
		if fieldSize == 0 {
			if fieldIdx == fieldsCount-1 {
				offset += int64(unsafe.Sizeof(uintptr(0)))
			}
			continue
		}

		if field.Embedded() {
			fieldType := field.Type()
			if named, ok := fieldType.(*types.Named); ok && named.Obj().Name() == "Node" {
				return offset, true
			}
			if embedded, ok := fieldType.Underlying().(*types.Struct); ok {
				if nodeOffset, ok := embeddedNodeOffset(embedded, offset); ok {
					return nodeOffset, true
				}
			}
		}

		offset += fieldSize
	}

	return 0, false
}

func iterStructFields(s *types.Struct, offset int64, filterField func(*types.Var) bool, genField func(field *types.Var, offset int64)) int64 {
	fieldsCount := s.NumFields()
	for fieldIdx := range fieldsCount {
		field := s.Field(fieldIdx)
		fieldAlign := sizes.Alignof(field.Type())
		offset = align(offset, fieldAlign)
		fieldSize := sizes.Sizeof(field.Type())
		if fieldSize == 0 {
			// https://dave.cheney.net/2015/10/09/padding-is-hard
			if fieldIdx == fieldsCount-1 {
				offset += int64(unsafe.Sizeof(uintptr(0)))
			}
			continue
		}

		if !filterField(field) {
			offset += fieldSize
			continue
		}

		if field.Embedded() {
			fieldType := field.Type()
			embedded := fieldType.Underlying().(*types.Struct)
			iterStructFields(embedded, offset, filterField, genField)
			offset += fieldSize
			continue
		}

		genField(field, offset)
		offset += fieldSize
	}

	return offset
}

// copied from go/types/sizes.go
//
// align returns the smallest y >= x such that y % a == 0.
// a must be within 1 and 8 and it must be a power of 2.
// The result may be negative due to overflow.
func align(x, a int64) int64 {
	return (x + a - 1) &^ (a - 1)
}
