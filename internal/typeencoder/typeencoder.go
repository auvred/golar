package typeencoder

import (
	"encoding/binary"
	"math"
	"unsafe"

	"github.com/microsoft/TypeScript/tsc/pkg/ast"
	"github.com/microsoft/TypeScript/tsc/pkg/checker"
	"github.com/microsoft/TypeScript/tsc/pkg/jsnum"
)

var intrinsicNames = [...]string{
	"any",
	"unresolved",
	"intrinsic",
	"unknown",
	"undefined",
	"null",
	"string",
	"number",
	"bigint",
	"symbol",
	"void",
	"never",
	"object",
	"error",
}

var intrinsicNameIds = make(map[string]uint8, len(intrinsicNames))

func init() {
	for i, name := range intrinsicNames {
		intrinsicNameIds[name] = uint8(i)
	}
}

type Encoder struct {
	// checker resolves lazily computed type data, such as type parameter
	// constraints. It must be the checker the encoded types come from.
	checker     *checker.Checker
	seenTypes   []uint64
	seenSymbols []uint64
	// Compact wire IDs are independent of upstream's block-allocated 64-bit IDs.
	symbolIDs map[ast.SymbolId]uint32
}

const (
	literalValueKindString byte = iota + 1
	literalValueKindNumber
	literalValueKindBoolean
	literalValueKindBigInt
)

const supportedLiteralFlags = checker.TypeFlagsStringLiteral |
	checker.TypeFlagsNumberLiteral |
	checker.TypeFlagsBooleanLiteral |
	checker.TypeFlagsBigIntLiteral

func New(c *checker.Checker) *Encoder {
	return &Encoder{checker: c, symbolIDs: make(map[ast.SymbolId]uint32)}
}

func (e *Encoder) Reset() {
	e.seenTypes = e.seenTypes[:0]
	e.seenSymbols = e.seenSymbols[:0]
}

func (e *Encoder) EncodeType(buf []byte, t *checker.Type) []byte {
	if t == nil {
		// type id is >1 (see func(*Checker)newType)
		return binary.LittleEndian.AppendUint32(buf, 0)
	}
	id := uint32(t.Id())
	buf = binary.LittleEndian.AppendUint32(buf, id)
	if e.markTypeSeen(id) {
		return buf
	}

	buf = appendPointer(buf, t)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(t.Flags()))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(t.ObjectFlags()))
	buf = e.EncodeSymbol(buf, t.Symbol())
	buf = e.appendObjectTypeFields(buf, t)

	switch {
	case t.Flags()&checker.TypeFlagsUnionOrIntersection != 0:
		buf = e.appendTypes(buf, t.Types())
	case t.Flags()&supportedLiteralFlags != 0:
		buf = appendLiteralValue(buf, t.AsLiteralType().Value())
	case t.Flags()&checker.TypeFlagsIntrinsic != 0:
		buf = append(buf, intrinsicNameId(t.AsIntrinsicType().IntrinsicName()))
	case t.Flags()&checker.TypeFlagsIndex != 0:
		buf = e.EncodeType(buf, t.AsIndexType().Target())
	case t.Flags()&checker.TypeFlagsIndexedAccess != 0:
		buf = e.EncodeType(buf, t.AsIndexedAccessType().ObjectType())
		buf = e.EncodeType(buf, t.AsIndexedAccessType().IndexType())
	case t.Flags()&checker.TypeFlagsConditional != 0:
		buf = e.EncodeType(buf, t.AsConditionalType().CheckType())
		buf = e.EncodeType(buf, t.AsConditionalType().ExtendsType())
	case t.Flags()&checker.TypeFlagsSubstitution != 0:
		buf = e.EncodeType(buf, t.AsSubstitutionType().BaseType())
		buf = e.EncodeType(buf, t.AsSubstitutionType().SubstConstraint())
	case t.Flags()&checker.TypeFlagsTemplateLiteral != 0:
		buf = appendStringArray(buf, t.AsTemplateLiteralType().Texts())
		buf = e.appendTypes(buf, t.AsTemplateLiteralType().Types())
	case t.Flags()&checker.TypeFlagsStringMapping != 0:
		buf = e.EncodeType(buf, t.AsStringMappingType().Target())
	case t.Flags()&checker.TypeFlagsTypeParameter != 0:
		buf = e.EncodeType(buf, e.checker.GetConstraintOfTypeParameter(t))
	}

	return buf
}

func (e *Encoder) appendObjectTypeFields(buf []byte, t *checker.Type) []byte {
	if t.Flags()&checker.TypeFlagsObject == 0 {
		return buf
	}

	objectFlags := t.ObjectFlags()
	if objectFlags&(checker.ObjectFlagsReference|checker.ObjectFlagsClassOrInterface|checker.ObjectFlagsTuple) != 0 {
		buf = e.EncodeType(buf, t.Target())
	}
	if objectFlags&(checker.ObjectFlagsClassOrInterface|checker.ObjectFlagsTuple) != 0 {
		iface := t.AsInterfaceType()
		buf = e.appendTypes(buf, iface.TypeParameters())
		buf = e.appendTypes(buf, iface.OuterTypeParameters())
		buf = e.appendTypes(buf, iface.LocalTypeParameters())
	}
	if objectFlags&checker.ObjectFlagsTuple != 0 {
		tuple := t.AsTupleType()
		buf = appendElementFlags(buf, tuple.ElementFlags())
		buf = binary.LittleEndian.AppendUint32(buf, uint32(tuple.FixedLength()))
		buf = appendBool(buf, tuple.IsReadonly())
	}
	return buf
}

func (e *Encoder) appendTypes(buf []byte, types []*checker.Type) []byte {
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(types)))
	for _, child := range types {
		buf = e.EncodeType(buf, child)
	}
	return buf
}

func (e *Encoder) EncodeSymbol(buf []byte, s *ast.Symbol) []byte {
	var id uint32
	if s != nil {
		symbolID := ast.GetSymbolId(s)
		id = e.symbolIDs[symbolID]
		if id == 0 {
			if uint64(len(e.symbolIDs)) == math.MaxUint32 {
				panic("encoder: exhausted symbol wire IDs")
			}
			id = uint32(len(e.symbolIDs) + 1)
			e.symbolIDs[symbolID] = id
		}
	}
	buf = binary.LittleEndian.AppendUint32(buf, id)
	if id == 0 {
		return buf
	}
	if e.markSymbolSeen(id) {
		return buf
	}

	buf = appendPointer(buf, s)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(s.Flags()))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(s.CheckFlags()))
	buf = appendByteString(buf, s.Name())

	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(s.Declarations())))
	for _, declaration := range s.Declarations() {
		buf = appendNodePointer(buf, declaration)
	}

	buf = appendNodePointer(buf, s.ValueDeclaration())
	buf = e.EncodeSymbol(buf, s.Parent())
	buf = e.EncodeSymbol(buf, s.ExportSymbol())

	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(s.Members())))
	for key, member := range s.Members() {
		buf = appendByteString(buf, key)
		buf = e.EncodeSymbol(buf, member)
	}

	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(s.Exports())))
	for key, exported := range s.Exports() {
		buf = appendByteString(buf, key)
		buf = e.EncodeSymbol(buf, exported)
	}

	return buf
}

func (e *Encoder) markTypeSeen(id uint32) bool {
	return markSeen(&e.seenTypes, id)
}

func (e *Encoder) markSymbolSeen(id uint32) bool {
	return markSeen(&e.seenSymbols, id)
}

func markSeen(seen *[]uint64, id uint32) bool {
	ensureSeen(seen, id)
	word := id >> 6
	mask := uint64(1) << (id & 63)
	wasSeen := (*seen)[word]&mask != 0
	(*seen)[word] |= mask
	return wasSeen
}

func ensureSeen(seen *[]uint64, id uint32) {
	need := int(id>>6) + 1
	if need <= len(*seen) {
		return
	}
	newLen := len(*seen)
	if newLen == 0 {
		newLen = 64
	}
	for newLen < need {
		newLen *= 2
	}
	*seen = append(*seen, make([]uint64, newLen-len(*seen))...)
}

func intrinsicNameId(name string) uint8 {
	id, ok := intrinsicNameIds[name]
	if !ok {
		panic("encoder: unknown intrinsic type name: " + name)
	}
	return id
}

func appendByteString(buf []byte, s string) []byte {
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(s)))
	return append(buf, s...)
}

func appendStringArray(buf []byte, values []string) []byte {
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(values)))
	for _, value := range values {
		buf = appendByteString(buf, value)
	}
	return buf
}

func appendElementFlags(buf []byte, flags []checker.ElementFlags) []byte {
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(flags)))
	for _, flag := range flags {
		buf = binary.LittleEndian.AppendUint32(buf, uint32(flag))
	}
	return buf
}

func appendBool(buf []byte, value bool) []byte {
	if value {
		return append(buf, 1)
	}
	return append(buf, 0)
}

func appendLiteralValue(buf []byte, value any) []byte {
	switch value := value.(type) {
	case string:
		buf = append(buf, literalValueKindString)
		return appendByteString(buf, value)
	case jsnum.Number:
		buf = append(buf, literalValueKindNumber)
		return binary.LittleEndian.AppendUint64(buf, math.Float64bits(float64(value)))
	case bool:
		buf = append(buf, literalValueKindBoolean)
		return appendBool(buf, value)
	case jsnum.PseudoBigInt:
		buf = append(buf, literalValueKindBigInt)
		return appendByteString(buf, value.String())
	default:
		panic("encoder: unsupported literal type value")
	}
}

func appendNodePointer(buf []byte, node *ast.Node) []byte {
	if node == nil {
		buf = binary.LittleEndian.AppendUint32(buf, 0)
		return binary.LittleEndian.AppendUint32(buf, 0)
	}
	buf = binary.LittleEndian.AppendUint32(buf, node.Id)
	return binary.LittleEndian.AppendUint32(buf, node.SourceFileId)
}

func appendPointer[T any](buf []byte, value *T) []byte {
	if value == nil {
		return binary.LittleEndian.AppendUint64(buf, 0)
	}
	return binary.LittleEndian.AppendUint64(buf, uint64(uintptr(unsafe.Pointer(value))))
}
