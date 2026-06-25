import assert from 'node:assert/strict'
import type {
	RemoteNode,
	RemoteNodeList,
} from '../../../thirdparty/typescript-go/_packages/native-preview/dist/api/node/node.js'
import {
	ObjectFlags,
	TypeFlags,
	type ElementFlags,
	type Node,
} from './unstable-tsgo.ts'

export interface NodeHandle {
	readonly nodeIndex: number
	readonly sourceFileId: number
}

export interface NodeResolver {
	resolveNode(handle: NodeHandle): RemoteNode | RemoteNodeList
}

export const intrinsicNames = [
	'any',
	'unresolved',
	'intrinsic',
	'unknown',
	'undefined',
	'null',
	'string',
	'number',
	'bigint',
	'symbol',
	'void',
	'never',
	'object',
	'error',
] as const

export const intrinsicNameIds = new Map<string, number>(
	intrinsicNames.map((name, id) => [name, id]),
)

const byteStringDecoder = new TextDecoder()

const literalValueKindString = 1
const literalValueKindNumber = 2
const literalValueKindBoolean = 3
const literalValueKindBigInt = 4

export class Registry {
	readonly types: Array<Type | undefined> = []
	readonly symbols: Array<Symbol | undefined> = []
	readonly nodeResolver: NodeResolver

	constructor(nodeResolver: NodeResolver) {
		this.nodeResolver = nodeResolver
	}

	getType(data: DataView, offset: number): [Type | undefined, number] {
		const id = data.getUint32(offset, true)
		if (id === 0) {
			return [undefined, 4]
		}

		const type = this.types[id]
		if (type) {
			return [type, 4] as const
		}

		const decoded = new Type(data, offset, this)
		return [decoded, decoded.len]
	}

	getSymbol(data: DataView, offset: number): [Symbol | undefined, number] {
		const id = data.getUint32(offset, true)
		if (id === 0) {
			return [undefined, 4]
		}

		const symbol = this.symbols[id]
		if (symbol) {
			return [symbol, 4] as const
		}

		const decoded = new Symbol(data, offset, this)
		return [decoded, decoded.len]
	}
}

export class Type {
	readonly id: number
	readonly pointerHi: number
	readonly pointerLo: number
	readonly flags: TypeFlags
	readonly objectFlags: ObjectFlags
	readonly symbol: Symbol | undefined
	readonly value: string | number | boolean | undefined
	readonly target: Type | undefined
	readonly typeParameters: readonly Type[] | undefined
	readonly outerTypeParameters: Type[] | undefined
	readonly localTypeParameters: Type[] | undefined
	readonly elementFlags: ElementFlags[] | undefined
	readonly fixedLength: number | undefined
	readonly readonly: boolean | undefined
	readonly objectType: Type | undefined
	readonly indexType: Type | undefined
	readonly checkType: Type | undefined
	readonly extendsType: Type | undefined
	readonly baseType: Type | undefined
	readonly constraint: Type | undefined
	readonly texts: string[] | undefined
	readonly intrinsicNameId: number | undefined
	readonly intrinsicName: string | undefined
	readonly types: Type[] | undefined
	readonly isThisType: boolean | undefined
	readonly len: number

	constructor(data: DataView, offset: number, registry: Registry) {
		this.id = data.getUint32(offset, true)
		registry.types[this.id] = this
		this.pointerLo = data.getUint32(offset + 4, true)
		this.pointerHi = data.getUint32(offset + 8, true)
		this.flags = data.getUint32(offset + 12, true)
		this.objectFlags = data.getUint32(offset + 16, true)

		const [symbol, symbolLen] = registry.getSymbol(data, offset + 20)
		this.symbol = symbol

		let childOffset = offset + 20 + symbolLen

		if (
			(this.flags & TypeFlags.Object) !== 0 &&
			(this.objectFlags &
				(ObjectFlags.Reference |
					ObjectFlags.ClassOrInterface |
					ObjectFlags.Tuple)) !==
				0
		) {
			const [target, targetLen] = registry.getType(data, childOffset)
			this.target = target
			childOffset += targetLen
		} else {
			this.target = undefined
		}

		if (
			(this.flags & TypeFlags.Object) !== 0 &&
			(this.objectFlags &
				(ObjectFlags.ClassOrInterface | ObjectFlags.Tuple)) !==
				0
		) {
			const [typeParameters, typeParametersLen] = readTypeArray(
				registry,
				data,
				childOffset,
			)
			this.typeParameters = typeParameters
			childOffset += typeParametersLen

			const [outerTypeParameters, outerTypeParametersLen] = readTypeArray(
				registry,
				data,
				childOffset,
			)
			this.outerTypeParameters = outerTypeParameters
			childOffset += outerTypeParametersLen

			const [localTypeParameters, localTypeParametersLen] = readTypeArray(
				registry,
				data,
				childOffset,
			)
			this.localTypeParameters = localTypeParameters
			childOffset += localTypeParametersLen
		}

		if (
			(this.flags & TypeFlags.Object) !== 0 &&
			(this.objectFlags & ObjectFlags.Tuple) !== 0
		) {
			const [elementFlags, elementFlagsLen] = readElementFlags(
				data,
				childOffset,
			)
			this.elementFlags = elementFlags
			childOffset += elementFlagsLen
			this.fixedLength = data.getUint32(childOffset, true)
			childOffset += 4
			this.readonly = data.getUint8(childOffset) !== 0
			childOffset += 1
		}

		if ((this.flags & TypeFlags.UnionOrIntersection) !== 0) {
			const [types, typesLen] = readTypeArray(registry, data, childOffset)
			this.types = types
			childOffset += typesLen
		} else if (
			(this.flags &
				(TypeFlags.StringLiteral |
					TypeFlags.NumberLiteral |
					TypeFlags.BooleanLiteral |
					TypeFlags.BigIntLiteral)) !==
			0
		) {
			const [value, valueLen] = readLiteralValue(data, childOffset)
			this.value = value
			childOffset += valueLen
		} else if ((this.flags & TypeFlags.Intrinsic) !== 0) {
			const intrinsicNameId = data.getUint8(childOffset)
			this.intrinsicNameId = intrinsicNameId
			this.intrinsicName = intrinsicNames[intrinsicNameId]
			childOffset += 1
		} else if ((this.flags & TypeFlags.Index) !== 0) {
			const [target, targetLen] = registry.getType(data, childOffset)
			this.target = target
			childOffset += targetLen
		} else if ((this.flags & TypeFlags.IndexedAccess) !== 0) {
			const [objectType, objectTypeLen] = registry.getType(data, childOffset)
			childOffset += objectTypeLen
			const [indexType, indexTypeLen] = registry.getType(data, childOffset)
			this.objectType = objectType
			this.indexType = indexType
			childOffset += indexTypeLen
		} else if ((this.flags & TypeFlags.Conditional) !== 0) {
			const [checkType, checkTypeLen] = registry.getType(data, childOffset)
			childOffset += checkTypeLen
			const [extendsType, extendsTypeLen] = registry.getType(data, childOffset)
			this.checkType = checkType
			this.extendsType = extendsType
			childOffset += extendsTypeLen
		} else if ((this.flags & TypeFlags.Substitution) !== 0) {
			const [baseType, baseTypeLen] = registry.getType(data, childOffset)
			childOffset += baseTypeLen
			const [constraint, constraintLen] = registry.getType(data, childOffset)
			this.baseType = baseType
			this.constraint = constraint
			childOffset += constraintLen
		} else if ((this.flags & TypeFlags.TemplateLiteral) !== 0) {
			const [texts, textsLen] = readStringArray(data, childOffset)
			childOffset += textsLen
			const [types, typesLen] = readTypeArray(registry, data, childOffset)
			this.types = types
			this.texts = texts
			childOffset += typesLen
		} else if ((this.flags & TypeFlags.StringMapping) !== 0) {
			const [target, targetLen] = registry.getType(data, childOffset)
			this.target = target
			childOffset += targetLen
		}

		this.len = childOffset - offset
	}
}

export class Symbol {
	readonly id: number
	readonly pointerHi: number
	readonly pointerLo: number
	readonly flags: number
	readonly checkFlags: number
	readonly name: string
	private readonly registry: Registry
	private readonly declarationPointers_: NodeHandle[]
	private readonly valueDeclarationPointer_: NodeHandle
	readonly parent: Symbol | undefined
	readonly exportSymbol: Symbol | undefined
	readonly members: Map<string, Symbol>
	readonly exports: Map<string, Symbol>
	readonly len: number
	private declarations_: RemoteNode[] | undefined
	private valueDeclaration_: RemoteNode | undefined

	constructor(data: DataView, offset: number, registry: Registry) {
		this.id = data.getUint32(offset, true)
		registry.symbols[this.id] = this
		this.registry = registry
		this.pointerLo = data.getUint32(offset + 4, true)
		this.pointerHi = data.getUint32(offset + 8, true)
		this.flags = data.getUint32(offset + 12, true)
		this.checkFlags = data.getUint32(offset + 16, true)

		const nameLen = data.getUint32(offset + 20, true)
		this.name = decodeByteString(data, offset + 24, nameLen)

		let childOffset = offset + 24 + nameLen

		const declarationCount = data.getUint32(childOffset, true)
		childOffset += 4
		this.declarationPointers_ = new Array<NodeHandle>(declarationCount)
		for (let i = 0; i < declarationCount; i++) {
			this.declarationPointers_[i] = readNodeHandle(data, childOffset)
			childOffset += 8
		}

		this.valueDeclarationPointer_ = readNodeHandle(data, childOffset)
		childOffset += 8

		const [parent, parentLen] = registry.getSymbol(data, childOffset)
		this.parent = parent
		childOffset += parentLen

		const [exportSymbol, exportSymbolLen] = registry.getSymbol(
			data,
			childOffset,
		)
		this.exportSymbol = exportSymbol
		childOffset += exportSymbolLen

		const [members, membersLen] = readSymbolTable(registry, data, childOffset)
		this.members = members
		childOffset += membersLen

		const [exports, exportsLen] = readSymbolTable(registry, data, childOffset)
		this.exports = exports
		childOffset += exportsLen

		this.len = childOffset - offset
	}

	get declarations(): Node[] {
		this.declarations_ ??= this.declarationPointers_.map(
			(pointer) =>
				this.registry.nodeResolver.resolveNode(pointer) as RemoteNode,
		)
		// @ts-expect-error
		return this.declarations_
	}

	get valueDeclaration(): Node | undefined {
		if (this.valueDeclaration_ === undefined) {
			this.valueDeclaration_ = this.registry.nodeResolver.resolveNode(
				this.valueDeclarationPointer_,
			) as RemoteNode
		}
		// @ts-expect-error
		return this.valueDeclaration_
	}
}

function readNodeHandle(data: DataView, offset: number): NodeHandle {
	return {
		nodeIndex: data.getUint32(offset, true),
		sourceFileId: data.getUint32(offset + 4, true),
	}
}

function readTypeArray(registry: Registry, data: DataView, offset: number) {
	const count = data.getUint32(offset, true)
	let childOffset = offset + 4
	const types = new Array<Type>(count)
	for (let i = 0; i < count; i++) {
		const [type, len] = registry.getType(data, childOffset)
		assert.ok(type != null, 'Unexpected nil child type')
		types[i] = type
		childOffset += len
	}
	return [types, childOffset - offset] as const
}

function readElementFlags(data: DataView, offset: number) {
	const count = data.getUint32(offset, true)
	let childOffset = offset + 4
	const flags = new Array<ElementFlags>(count)
	for (let i = 0; i < count; i++) {
		flags[i] = data.getUint32(childOffset, true)
		childOffset += 4
	}
	return [flags, childOffset - offset] as const
}

function readStringArray(data: DataView, offset: number) {
	const count = data.getUint32(offset, true)
	let childOffset = offset + 4
	const values = new Array<string>(count)
	for (let i = 0; i < count; i++) {
		const len = data.getUint32(childOffset, true)
		values[i] = decodeByteString(data, childOffset + 4, len)
		childOffset += 4 + len
	}
	return [values, childOffset - offset] as const
}

function readLiteralValue(data: DataView, offset: number) {
	const kind = data.getUint8(offset)
	switch (kind) {
		case literalValueKindString: {
			const len = data.getUint32(offset + 1, true)
			return [decodeByteString(data, offset + 5, len), 5 + len] as const
		}
		case literalValueKindNumber:
			return [data.getFloat64(offset + 1, true), 9] as const
		case literalValueKindBoolean:
			return [data.getUint8(offset + 1) !== 0, 2] as const
		case literalValueKindBigInt: {
			const len = data.getUint32(offset + 1, true)
			return [decodeByteString(data, offset + 5, len), 5 + len] as const
		}
		default:
			throw new Error(`Unknown literal value kind: ${kind}`)
	}
}

function readSymbolTable(registry: Registry, data: DataView, offset: number) {
	const count = data.getUint32(offset, true)
	let childOffset = offset + 4
	const symbols = new Map<string, Symbol>()
	for (let i = 0; i < count; i++) {
		const keyLen = data.getUint32(childOffset, true)
		const key = decodeByteString(data, childOffset + 4, keyLen)
		childOffset += 4 + keyLen

		const [symbol, len] = registry.getSymbol(data, childOffset)
		assert.ok(symbol != null, 'Unexpected nil child symbol')
		symbols.set(key, symbol)
		childOffset += len
	}
	return [symbols, childOffset - offset] as const
}

function decodeByteString(data: DataView, offset: number, len: number) {
	return byteStringDecoder.decode(
		new Uint8Array(data.buffer, data.byteOffset + offset, len),
	)
}
