import path from 'node:path'
import { expect, test } from 'vitest'
import {
	ElementFlags,
	ObjectFlags,
	TypeFlags,
	cast,
	isIdentifier,
	isInterfaceDeclaration,
	isPropertySignatureDeclaration,
	isTypeAliasDeclaration,
	isVariableStatement,
	SyntaxKind,
	type SourceFile,
} from '../../packages/golar/src/unstable-tsgo.ts'
import { loadAddon } from '../../packages/golar/src/addon.ts'
import { Workspace } from '../../packages/golar/src/workspace.ts'

const fixtureDir = path.join(import.meta.dirname, 'fixture')
const typesFile = path.join(fixtureDir, 'types.ts')

const indexFile = path.join(fixtureDir, 'index.ts')
const extraFile = path.join(fixtureDir, 'extra.ts')

loadAddon()

test('getTypeAtLocation decodes intrinsic property types', async () => {
	const workspace = await (
		await Workspace.create(fixtureDir, [indexFile])
	).workspace
	workspace.preloadRequestedFiles([indexFile])
	const { file, program } = workspace.requestedFiles.get(indexFile)!
	const userDecl = cast(file.statements[0], isInterfaceDeclaration)
	const stringMember = cast(userDecl.members[0]!, isPropertySignatureDeclaration)
	const numberMember = cast(userDecl.members[1]!, isPropertySignatureDeclaration)
	const stringProperty = cast(stringMember.name, isIdentifier)
	const numberProperty = cast(numberMember.name, isIdentifier)

	expect(stringProperty.text).toBe('name')
	expect(numberProperty.text).toBe('age')

	const stringType = program.getTypeAtLocation(stringProperty)!
	const numberType = program.getTypeAtLocation(numberProperty)!

	expect(stringType.intrinsicName).toBe('string')
	expect(numberType.intrinsicName).toBe('number')
})

test('getTypeAtLocation decodes union types', async () => {
	const workspace = await (
		await Workspace.create(fixtureDir, [indexFile])
	).workspace
	workspace.preloadRequestedFiles([indexFile])
	const { file, program } = workspace.requestedFiles.get(indexFile)!
	const unionValue = getVariableDeclaration(file, 1)

	expect(unionValue.text).toBe('unionValue')

	const unionType = program.getTypeAtLocation(unionValue)!
	expect(
		unionType.types?.map((type) => type.intrinsicName).sort(),
	).toStrictEqual(['number', 'string'])
})

test('getTypeAtLocation decodes symbol metadata', async () => {
	const workspace = await (
		await Workspace.create(fixtureDir, [indexFile])
	).workspace
	workspace.preloadRequestedFiles([indexFile])
	const { file, program } = workspace.requestedFiles.get(indexFile)!
	const userDecl = cast(file.statements[0], isInterfaceDeclaration)

	const userDeclType = program.getTypeAtLocation(userDecl)!

	expect(userDeclType.symbol?.name).toBe('User')
	expect(userDeclType.symbol?.id).toBeGreaterThan(0)
	expect(userDeclType.symbol?.declarations).toHaveLength(1)
	expect(userDeclType.symbol!.declarations[0]).toEqual(userDecl)
})

test('getTypeAtLocation lazily fetches sibling declaration source files', async () => {
	const workspace = await (
		await Workspace.create(fixtureDir, [indexFile])
	).workspace
	workspace.preloadRequestedFiles([indexFile])
	const index = workspace.requestedFiles.get(indexFile)!

	const varStmt = getVariableDeclaration(index.file, 3)
	expect(varStmt.text).toBe('foo')

	expect(
		workspace['sourceFileById'].filter((f) => !!f).map((f) => f.fileName),
	).toEqual([indexFile.replaceAll('\\', '/')])

	const type = index.program.getTypeAtLocation(varStmt)!

	expect(type.symbol?.name).toBe('Foo')

	expect(
		workspace['sourceFileById'].filter((f) => !!f).map((f) => f.fileName),
	).toEqual([indexFile.replaceAll('\\', '/')])

	expect(type.symbol!.declarations[0]!.kind).toEqual(
		SyntaxKind.VariableDeclaration,
	)
	expect.assert.sameMembers(
		workspace['sourceFileById'].filter((f) => !!f).map((f) => f.fileName),
		[extraFile.replaceAll('\\', '/'), indexFile.replaceAll('\\', '/')],
	)
})

test('getTypeAtLocation decodes literal values', async () => {
	const workspace = await (
		await Workspace.create(fixtureDir, [typesFile])
	).workspace
	workspace.preloadRequestedFiles([typesFile])
	const { file, program } = workspace.requestedFiles.get(typesFile)!

	const stringType = program.getTypeAtLocation(getVariableDeclaration(file, 0))!
	const numberType = program.getTypeAtLocation(getVariableDeclaration(file, 1))!
	const booleanType = program.getTypeAtLocation(
		getVariableDeclaration(file, 2),
	)!
	const bigintType = program.getTypeAtLocation(getVariableDeclaration(file, 3))!

	expect(stringType.flags & TypeFlags.StringLiteral).toBeTruthy()
	expect(stringType.value).toBe('hello')
	expect(numberType.flags & TypeFlags.NumberLiteral).toBeTruthy()
	expect(numberType.value).toBe(42)
	expect(booleanType.flags & TypeFlags.BooleanLiteral).toBeTruthy()
	expect(booleanType.value).toBe(true)
	expect(bigintType.flags & TypeFlags.BigIntLiteral).toBeTruthy()
	expect(bigintType.value).toBe('123')
})

test('getTypeAtLocation decodes reference', async () => {
	const workspace = await (
		await Workspace.create(fixtureDir, [typesFile])
	).workspace
	workspace.preloadRequestedFiles([typesFile])
	const { file, program } = workspace.requestedFiles.get(typesFile)!

	const boxType = program.getTypeAtLocation(
		cast(file.statements[4], isInterfaceDeclaration).name,
	)!
	expect(boxType.flags & TypeFlags.Object).toBeTruthy()
	expect(boxType.typeParameters).toHaveLength(2)
	expect(boxType.outerTypeParameters).toHaveLength(0)
	expect(boxType.localTypeParameters).toHaveLength(2)
	expect(
		boxType.typeParameters!.every((type) =>
			Boolean(type.flags & TypeFlags.TypeParameter),
		),
	).toBe(true)
})

test('getTypeAtLocation decodes interface', async () => {
	const workspace = await (
		await Workspace.create(fixtureDir, [typesFile])
	).workspace
	workspace.preloadRequestedFiles([typesFile])
	const { file, program } = workspace.requestedFiles.get(typesFile)!

	const arrayType = program.getTypeAtLocation(getVariableDeclaration(file, 5))!
	expect(arrayType.flags & TypeFlags.Object).toBeTruthy()
	expect(arrayType.objectFlags & ObjectFlags.Reference).toBeTruthy()
	expect(arrayType.target).toBeDefined()
	expect(arrayType.target!.flags & TypeFlags.Object).toBeTruthy()
})

test('getTypeAtLocation decodes tuple type fields', async () => {
	const workspace = await (
		await Workspace.create(fixtureDir, [typesFile])
	).workspace
	workspace.preloadRequestedFiles([typesFile])
	const { file, program } = workspace.requestedFiles.get(typesFile)!

	const tupleTypeRef = program.getTypeAtLocation(
		getVariableDeclaration(file, 6),
	)!
	const tupleType =
		tupleTypeRef.objectFlags & ObjectFlags.Tuple
			? tupleTypeRef
			: tupleTypeRef.target!
	expect(tupleType.objectFlags & ObjectFlags.Tuple).toBeTruthy()
	expect(tupleType.elementFlags).toStrictEqual([
		ElementFlags.Required,
		ElementFlags.Optional,
		ElementFlags.Rest,
	])
	expect(tupleType.fixedLength).toBe(2)
	expect(tupleType.readonly).toBe(true)
	expect(tupleTypeRef.target).toBeDefined()
})

test('getTypeAtLocation decodes advanced type relationships', async () => {
	const workspace = await (
		await Workspace.create(fixtureDir, [typesFile])
	).workspace
	workspace.preloadRequestedFiles([typesFile])
	const { file, program } = workspace.requestedFiles.get(typesFile)!

	const keyOfType = program.getTypeAtLocation(getTypeAliasDeclaration(file, 7))!
	expect(keyOfType.flags & TypeFlags.Index).toBeTruthy()
	expect(keyOfType.target).toBeDefined()
	expect(keyOfType.target!.flags & TypeFlags.TypeParameter).toBeTruthy()

	const lookupType = program.getTypeAtLocation(
		getTypeAliasDeclaration(file, 8),
	)!
	expect(lookupType.flags & TypeFlags.IndexedAccess).toBeTruthy()
	expect(lookupType.objectType).toBeDefined()
	expect(lookupType.indexType).toBeDefined()

	const conditionalType = program.getTypeAtLocation(
		getTypeAliasDeclaration(file, 9),
	)!
	expect(conditionalType.flags & TypeFlags.Conditional).toBeTruthy()
	expect(conditionalType.checkType).toBeDefined()
	expect(
		conditionalType.checkType!.flags & TypeFlags.TypeParameter,
	).toBeTruthy()
	expect(conditionalType.extendsType).toBeDefined()
	expect(conditionalType.extendsType!.intrinsicName).toBe('string')

	const templateLiteralType = program.getTypeAtLocation(
		getVariableDeclaration(file, 10),
	)!
	expect(templateLiteralType.flags & TypeFlags.TemplateLiteral).toBeTruthy()
	expect(templateLiteralType.texts).toStrictEqual(['hello ', ''])
	expect(templateLiteralType.types).toHaveLength(1)
	expect(templateLiteralType.types![0]!.intrinsicName).toBe('string')

	const stringMappingType = program.getTypeAtLocation(
		getTypeAliasDeclaration(file, 11),
	)!
	expect(stringMappingType.flags & TypeFlags.StringMapping).toBeTruthy()
	expect(stringMappingType.target).toBeDefined()
	expect(stringMappingType.target!.flags & TypeFlags.TypeParameter).toBeTruthy()
})

function getVariableDeclaration(file: SourceFile, index: number) {
	return cast(
		cast(file.statements[index], isVariableStatement).declarationList
			.declarations[0]!.name,
		isIdentifier,
	)
}
function getTypeAliasDeclaration(file: SourceFile, index: number) {
	return cast(
		cast(file.statements[index], isTypeAliasDeclaration).name,
		isIdentifier,
	)
}
