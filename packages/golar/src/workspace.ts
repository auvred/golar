import os from 'node:os'
import path from 'node:path'
import process from 'node:process'
import worker_threads from 'node:worker_threads'
import assert from 'node:assert/strict'

import type {
	Node,
	SourceFile,
} from '../../../thirdparty/typescript-go/_packages/native-preview/dist/ast/ast.js'
import {
	type RemoteNode,
	RemoteSourceFile,
	type RemoteNodeList,
} from '../../../thirdparty/typescript-go/_packages/native-preview/dist/api/node/node.js'
import { Registry, Type, type NodeHandle } from './type-decoder.ts'
import * as v from 'valibot'
import { addon, golarAddonPath, syncBuf, syncView } from './addon.ts'
import type { LintConfiguredRule } from './config.ts'

export type SourceFileWithProgram = {
	file: SourceFile
	program: Program
}

const textDecoder = new TextDecoder()
const textEncoder = new TextEncoder()
const nativeAddons = new Map<
	string,
	{
		setup?(golarAddonPath: string): void
		lint(workspacePtr: bigint, fileIdx: number, ruleNames: string[]): string
	}
>()

function writeNodePointer(offset: number, node: Node): number {
	const n = node as RemoteNode
	syncView.setUint32(offset, n.pointerLo, true)
	syncView.setUint32((offset += 4), n.pointerHi, true)
	return offset + 4
}

function materializeFile(file: RemoteSourceFile) {
	file.forEachChild(function visit(node) {
		node.forEachChild(visit)
	})
}

export class Program {
	private readonly id: number
	private readonly workspace: Workspace
	private readonly registry: Registry

	constructor(id: number, workspace: Workspace) {
		this.id = id
		this.workspace = workspace
		this.registry = new Registry({
			resolveNode(handle) {
				return workspace['resolveNode'](handle)
			},
		})
	}

	private writeId(offset: number): number {
		syncView.setUint32(offset, this.id, true)
		return offset + 4
	}

	getTypeAtLocation(node: Node): Type | undefined {
		let offset = this.workspace['writePointer'](0)
		offset = this.writeId(offset)
		offset = writeNodePointer(offset, node)
		addon.workspace_GetTypeAtLocation(worker_threads.threadId)

		return this.registry.getType(syncView, 0)[0]
	}
}

export class Workspace implements Disposable {
	private readonly pointerHi: number
	private readonly pointerLo: number

	readonly requestedFiles = new Map<string, SourceFileWithProgram>()
	// TODO: normalization, etc, etc.
	readonly filenameToIndex = new Map<string, number>()
	private readonly programs: Program[]
	private readonly sourceFileById: (RemoteSourceFile | null)[]

	static async create(cwd: string, filenames: string[]) {
		assert.ok(
			worker_threads.isMainThread,
			'Workspace.create can only be called from the main thread',
		)
		// go side relies on this
		assert.equal(worker_threads.threadId, 0)
		let offset = 0
		const { written: cwdLen } = textEncoder.encodeInto(
			cwd,
			new Uint8Array(syncBuf, offset + 4),
		)
		syncView.setUint32(offset, cwdLen, true)
		offset += 4 + cwdLen
		syncView.setUint32(offset, filenames.length, true)
		offset += 4
		for (const filename of filenames) {
			const { written: filenameLen } = textEncoder.encodeInto(
				filename,
				new Uint8Array(syncBuf, offset + 4),
			)
			syncView.setUint32(offset, filenameLen, true)
			offset += 4 + filenameLen
		}

		const ws = Promise.withResolvers<Workspace>()
		const wsMeta = Promise.withResolvers<{
			pointerLo: number
			pointerHi: number
			programsCount: number
			sourceFilesCount: number
		}>()
		addon.workspace_New(() => {
			let offset = 0
			const pointerLo = syncView.getUint32(offset, true)
			const pointerHi = syncView.getUint32((offset += 4), true)
			const programsCount = syncView.getUint32((offset += 4), true)
			const sourceFilesCount = syncView.getUint32((offset += 4), true)
			wsMeta.resolve({
				pointerLo,
				pointerHi,
				programsCount,
				sourceFilesCount,
			})
			// allow the calling site to spawn worker threads
			setImmediate(() => {
				ws.resolve(
					new Workspace(
						pointerLo,
						pointerHi,
						programsCount,
						sourceFilesCount,
						cwd,
						filenames,
					),
				)
			})
		})
		return {
			meta: await wsMeta.promise,
			workspace: ws.promise,
		}
	}

	static createWorker(
		pointerLo: number,
		pointerHi: number,
		programsCount: number,
		sourceFilesCount: number,
		cwd: string,
		filenames: string[],
	) {
		return new Workspace(
			pointerLo,
			pointerHi,
			programsCount,
			sourceFilesCount,
			cwd,
			filenames,
		)
	}

	// TODO: register finalizer
	private constructor(
		pointerLo: number,
		pointerHi: number,
		programsCount: number,
		sourceFilesCount: number,
		cwd: string,
		filenames: string[],
	) {
		this.pointerLo = pointerLo
		this.pointerHi = pointerHi

		this.programs = new Array(programsCount)
			.fill(null)
			.map((_, id) => new Program(id, this))

		this.sourceFileById = new Array(sourceFilesCount).fill(null)

		for (const [i, filename] of filenames.entries()) {
			this.filenameToIndex.set(filename, i)
		}
	}

	preloadRequestedFiles(filenames: string[]) {
		for (const filename of filenames) {
			this.loadRequestedFile(filename)
		}
	}

	private loadRequestedFile(filename: string) {
		const existing = this.requestedFiles.get(filename)
		if (existing != null) {
			return existing
		}

		const index = this.filenameToIndex.get(filename)
		assert.ok(index != null, `missing requested file for ${filename}`)

		let offset = this.writePointer(0)
		syncView.setUint32(offset, index, true)
		addon.workspace_ReadRequestedFileAt(worker_threads.threadId)

		const programId = syncView.getUint32((offset = 0), true)
		const program = this.programs[programId]!
		const sourceFileId = syncView.getUint32((offset += 4), true)
		const file = this.readSourceFile((offset += 4))
		this.sourceFileById[sourceFileId] = file

		const requestedFile = {
			file: file as unknown as SourceFile,
			program,
		}
		this.requestedFiles.set(filename, requestedFile)
		return requestedFile
	}

	private writePointer(offset: number): number {
		syncView.setUint32(offset, this.pointerLo, true)
		syncView.setUint32((offset += 4), this.pointerHi, true)
		return (offset += 4)
	}

	private readSourceFile(offset: number) {
		const encodedLen = syncView.getUint32(offset, true)

		const encodedFile = new Uint8Array(encodedLen)
		encodedFile.set(new Uint8Array(syncBuf, (offset += 4), encodedLen))
		const file = new RemoteSourceFile(encodedFile, textDecoder)
		// for nodeIndex lookup
		materializeFile(file)
		return file
	}

	private resolveNode(handle: NodeHandle): RemoteNode | RemoteNodeList {
		if (handle.nodeIndex === 0 && handle.sourceFileId) {
			throw new Error('nil node')
		}
		let file = this.sourceFileById[handle.sourceFileId]
		if (file == null) {
			let offset = this.writePointer(0)
			syncView.setUint32(offset, handle.sourceFileId, true)
			addon.workspace_ReadFileById(worker_threads.threadId)
			file = this.sourceFileById[handle.sourceFileId] = this.readSourceFile(0)
		}
		return file.nodes[handle.nodeIndex]!
	}

	typecheck(files: string[]): number {
		if (files.length === 0) {
			return 0
		}

		let offset = this.writePointer(0)
		syncView.setUint32(offset, files.length, true)
		offset += 4
		for (const filename of files) {
			const fileIdx = this.filenameToIndex.get(filename)
			assert.ok(fileIdx != null)
			syncView.setUint32(offset, fileIdx, true)
			offset += 4
		}
		return addon.workspace_TypeCheck()
	}

	lintBuiltin(rulesByFile: Map<string, LintConfiguredRule[]>) {
		if (rulesByFile.size === 0) {
			return
		}

		let ruleIdxCounter = 0
		const denseRuleIndexes = new Map<LintConfiguredRule, number>()
		const denseRules: {
			name: string
			options: unknown
		}[] = []
		const files: {
			file: number
			ruleIndexes: number[]
		}[] = []

		for (const [file, rules] of rulesByFile) {
			const indexes: number[] = []
			files.push({
				file: this.filenameToIndex.get(file)!,
				ruleIndexes: indexes,
			})
			for (const rule of rules) {
				let idx = denseRuleIndexes.get(rule)
				if (idx == null) {
					denseRuleIndexes.set(rule, (idx = ruleIdxCounter++))
					denseRules.push({
						// @ts-expect-error
						name: rule.rule.name,
						// @ts-expect-error
						options: v.parse(v.object(rule.rule.options), rule.options ?? {}),
					})
				}
				indexes.push(idx)
			}
		}

		let offset = this.writePointer(0)
		const { written: length } = textEncoder.encodeInto(
			JSON.stringify({ files, rules: denseRules }),
			new Uint8Array(syncBuf, offset + 4),
		)
		syncView.setUint32(offset, length, true)
		offset += 4 + length
		addon.workspace_Lint()
	}

	lintNative(rulesByFile: Map<string, LintConfiguredRule[]>, baseDir: string) {
		if (rulesByFile.size === 0) {
			return
		}

		const workspacePtr =
			(BigInt(this.pointerHi) << 32n) | BigInt(this.pointerLo)

		for (const [fileName, rules] of rulesByFile) {
			const fileIdx = this.filenameToIndex.get(fileName)
			assert.ok(fileIdx != null, `missing requested file for ${fileName}`)

			const rulesByAddon = new Map<string, Set<string>>()
			for (const configuredRule of rules) {
				// @ts-expect-error
				const addonPath = path.resolve(baseDir, configuredRule.rule.addonPath)
				let addonRuleNames = rulesByAddon.get(addonPath)
				if (addonRuleNames == null) {
					rulesByAddon.set(addonPath, (addonRuleNames = new Set()))
				}
				// @ts-expect-error
				addonRuleNames.add(configuredRule.rule.name)
			}

			for (const [nativeAddonPath, addonRuleNames] of rulesByAddon) {
				let nativeAddon = nativeAddons.get(nativeAddonPath)
				if (nativeAddon == null) {
					const addonModule = {
						exports: {} as {
							setup?(golarAddonPath: string): void
							lint(
								workspacePtr: bigint,
								fileIdx: number,
								ruleNames: string[],
							): string
						},
					}
					process.dlopen(
						addonModule,
						nativeAddonPath,
						os.constants.dlopen.RTLD_NOW,
					)
					nativeAddon = addonModule.exports
					nativeAddon.setup?.(golarAddonPath)
					nativeAddons.set(nativeAddonPath, nativeAddon)
				}

				nativeAddon.lint(workspacePtr, fileIdx, Array.from(addonRuleNames))
			}
		}
	}

	lintJs(jsFileToRules: Map<string, any[]>) {
		const workspace = this
		for (const [fileName, rules] of jsFileToRules) {
			const file = this.loadRequestedFile(fileName)
			const fileIdx = this.filenameToIndex.get(fileName)
			assert.ok(fileIdx != null, `missing requested file for ${fileName}`)
			for (const configuredRule of rules) {
				configuredRule.rule.setup({
					program: file.program,
					sourceFile: file.file,
					report(report: {
						message: string
						range: {
							begin: number
							end: number
						}
					}) {
						let offset = 0
						offset = workspace.writePointer(offset)
						syncView.setUint32(offset, fileIdx, true)
						offset += 4
						syncView.setUint32(offset, report.range.begin, true)
						offset += 4
						syncView.setUint32(offset, report.range.end, true)
						offset += 4

						const { written: ruleNameLength } = textEncoder.encodeInto(
							configuredRule.rule.name,
							new Uint8Array(syncBuf, offset + 4),
						)
						syncView.setUint32(offset, ruleNameLength, true)
						offset += 4 + ruleNameLength

						const { written: messageLength } = textEncoder.encodeInto(
							report.message,
							new Uint8Array(syncBuf, offset + 4),
						)
						syncView.setUint32(offset, messageLength, true)
						offset += 4 + messageLength

						addon.workspace_Report(worker_threads.threadId)
					},
				})
			}
		}
	}

	[Symbol.dispose]() {
		// TODO: dispose
	}
}
