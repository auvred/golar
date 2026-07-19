/// <reference types="@volar/typescript" />

import {
	JsCodegenPlugin,
	type FileExtension,
	type ExpectErrorDirectiveCodegenMapping,
	type IgnoreDirectiveCodegenMapping,
	type CodegenMapping,
	type Promisable,
	type CodegenScriptKind,
	type ServiceCodeError,
} from 'golar/unstable'
import type { LanguagePlugin, VirtualCode } from '@volar/language-core'
import type ts from 'typescript'

export type VolarLanguagePlugin = LanguagePlugin<string> & {
	getVirtualCodeErrors?(root: VirtualCode): ServiceCodeError[]
}
export type CreateVolarPluginOptions = {
	filename: string
	extensions: FileExtension[]
	languagePlugins:
		| VolarLanguagePlugin[]
		| ((
				cwd: string,
				configFileName: string | null,
		  ) => Promisable<VolarLanguagePlugin[]>)
}

export function createVolarPlugin(opts: CreateVolarPluginOptions) {
	const languagePluginsByProject = new Map<
		string,
		Promisable<VolarLanguagePlugin[]>
	>()
	new JsCodegenPlugin({
		id: opts.filename,
		extensions: opts.extensions,
		async createServiceCode(cwd, configFileName, fileName, sourceText) {
			let languagePlugins: VolarLanguagePlugin[]
			if (Array.isArray(opts.languagePlugins)) {
				languagePlugins = opts.languagePlugins
			} else {
				const project = `${cwd}::${configFileName}`
				let plugins = languagePluginsByProject.get(project)
				if (plugins == null) {
					languagePluginsByProject.set(
						project,
						(plugins = opts.languagePlugins(cwd, configFileName)),
					)
				}
				languagePlugins = await plugins
			}
			for (const plugin of languagePlugins) {
				if (plugin.createVirtualCode == null) {
					continue
				}
				const languageId = plugin.getLanguageId(fileName)
				if (languageId == null) {
					continue
				}

				const virtualCode = plugin.createVirtualCode(
					fileName,
					languageId,
					{
						getLength() {
							return sourceText.length
						},
						getText(start, end) {
							return sourceText.slice(start, end)
						},
						dispose() {},
						getChangeRange() {
							return undefined
						},
					},
					{
						getAssociatedScript(scriptId) {
							return undefined
						},
					},
				)
				if (virtualCode == null) {
					continue
				}

				{
					const errors = plugin.getVirtualCodeErrors?.(virtualCode)
					if (errors?.length) {
						return {
							errors,
						}
					}
				}

				const serviceScript = plugin.typescript!.getServiceScript(virtualCode)
				const serviceText = serviceScript!.code.snapshot.getText(
					0,
					serviceScript!.code.snapshot.getLength(),
				)

				const verificationMappings = serviceScript!.code.mappings.filter(
					(m) => m.data.verification,
				)
				const sourceOffsets = new Set<number>()
				const serviceOffsets = new Set<number>()
				type ExpectErrorGroup = {
					sourceOffset: number
					sourceLength: number
					serviceRanges: [number, number][]
				}
				const expectErrorGroups: ExpectErrorGroup[] = []
				let currentExpectErrorGroup: ExpectErrorGroup | undefined

				for (const m of verificationMappings) {
					for (const [i, offset] of m.sourceOffsets.entries()) {
						sourceOffsets.add(offset)
						sourceOffsets.add(offset + m.lengths[i]!)
					}
					for (const [i, offset] of m.generatedOffsets.entries()) {
						serviceOffsets.add(offset)
						serviceOffsets.add(offset + (m.generatedLengths ?? m.lengths)[i]!)
					}
					if ('__expectErrorCommentLoc' in m.data) {
						const [start, end] = m.data.__expectErrorCommentLoc as [
							number,
							number,
						]
						sourceOffsets.add(start).add(end)
					}
				}

				const serviceCovered: [number, number][] = []
				const expectErrorMappings: ExpectErrorDirectiveCodegenMapping[] = []

				const mappings = verificationMappings.flatMap((m): CodegenMapping[] => {
					const mappingData = m.data as {
						__suppressedDiagnostics?: number[]
						__expectErrorCommentLoc?: [number, number]
					}
					const suppressedDiagnostics = Array.isArray(
						mappingData.__suppressedDiagnostics,
					)
						? mappingData.__suppressedDiagnostics
						: undefined
					if ('__expectErrorCommentLoc' in mappingData) {
						const [start, end] = mappingData.__expectErrorCommentLoc!
						const sourceLength = end - start
						if (
							currentExpectErrorGroup == null ||
							currentExpectErrorGroup.sourceOffset !== start ||
							currentExpectErrorGroup.sourceLength !== sourceLength
						) {
							currentExpectErrorGroup = {
								sourceOffset: start,
								sourceLength,
								serviceRanges: [],
							}
							expectErrorGroups.push(currentExpectErrorGroup)
						}
					} else {
						currentExpectErrorGroup = undefined
					}
					return m.sourceOffsets.map((sourceOffset, i): CodegenMapping => {
						const generatedOffset = m.generatedOffsets[i]!
						const sourceLength = m.lengths[i]!
						const generatedLength = m.generatedLengths?.[i] ?? sourceLength
						if (generatedLength > 0) {
							serviceCovered.push([
								generatedOffset,
								generatedOffset + generatedLength,
							])
						}

						if (currentExpectErrorGroup != null) {
							currentExpectErrorGroup.serviceRanges.push([
								generatedOffset,
								generatedOffset + generatedLength,
							])
						}

						return {
							sourceOffset,
							serviceOffset: generatedOffset,
							sourceLength,
							serviceLength: generatedLength,
							suppressedDiagnostics: suppressedDiagnostics?.length
								? [...suppressedDiagnostics]
								: undefined,
						}
					})
				})

				serviceCovered.sort((a, b) => a[0] - b[0] || a[1] - b[1])

				const merged: [number, number][] = []
				for (const [s, e] of serviceCovered) {
					const last = merged.at(-1)
					if (!last) {
						merged.push([s, e])
					} else if (s <= last[1]) {
						last[1] = Math.max(last[1], e)
					} else {
						merged.push([s, e])
					}
				}

				const ignoreMappings: IgnoreDirectiveCodegenMapping[] = []
				let cursor = 0

				for (const [s, e] of merged) {
					if (s > cursor) {
						ignoreMappings.push({
							serviceOffset: cursor,
							serviceLength: s - cursor,
						})
					}
					cursor = Math.max(cursor, e)
				}

				ignoreMappings.push({
					serviceOffset: cursor,
					serviceLength: serviceText.length - cursor,
				})

				for (const group of expectErrorGroups) {
					if (group.serviceRanges.length === 0) {
						continue
					}

					let start = Number.POSITIVE_INFINITY
					let end = Number.NEGATIVE_INFINITY
					for (const [s, e] of group.serviceRanges) {
						start = Math.min(start, s)
						end = Math.max(end, e)
					}

					expectErrorMappings.push({
						sourceOffset: group.sourceOffset,
						serviceOffset: start,
						sourceLength: group.sourceLength,
						serviceLength: Math.max(end - start, 0),
					})
				}

				return {
					serviceText,
					scriptKind: tsScriptKindToGolar(serviceScript?.scriptKind),
					mappings,
					ignoreMappings,
					expectErrorMappings,
					ignoreNotMappedDiagnostics: true,
				}
			}
			throw new Error(`Cannot determine language id for file ${fileName}`)
		},
	})
}

function tsScriptKindToGolar(
	scriptKind: ts.ScriptKind | undefined,
): CodegenScriptKind {
	switch (scriptKind) {
		case 1 satisfies ts.ScriptKind.JS:
			return 'js'
		case 2 satisfies ts.ScriptKind.JSX:
			return 'jsx'
		case 3 satisfies ts.ScriptKind.TS:
			return 'ts'
		case 4 satisfies ts.ScriptKind.TSX:
			return 'tsx'
		default:
			return 'ts'
	}
}
