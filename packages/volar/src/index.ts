/// <reference types="@volar/typescript" />

import { createPlugin, type IgnoreDirectiveMapping, type Mapping, type ScriptKind } from '@golar/plugin'
import type { LanguagePlugin } from '@volar/language-core'
import type ts from 'typescript'

type Promisable<T> = T | Promise<T>

export type CreateVolarPluginOptions = {
	filename: string
	languagePlugins: LanguagePlugin<string>[]
}

export function createVolarPlugin(opts: CreateVolarPluginOptions) {
	createPlugin({
		filename: opts.filename,
		extraExtensions: opts.languagePlugins.flatMap(p => p.typescript?.extraFileExtensions?.map(e => `.${e.extension}`) ?? []),
		async createServiceCode(fileName, sourceText) {
			for (const plugin of opts.languagePlugins) {
				if (plugin.createVirtualCode == null) {
					continue
				}
				const languageId = plugin.getLanguageId(fileName)
				if (languageId == null) {
					continue
				}

				const virtualCode = plugin.createVirtualCode(fileName, languageId, {
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
				}, {
					getAssociatedScript(scriptId) {
						return undefined
					},
				})
				if (virtualCode == null) {
					continue
				}

				const serviceScript = plugin.typescript!.getServiceScript(virtualCode)
				const serviceText = serviceScript!.code.snapshot.getText(0, serviceScript!.code.snapshot.getLength())

				const verificationMappings = serviceScript!.code.mappings.filter(m => m.data.verification)
				const sourceOffsets = new Set<number>()
				const serviceOffsets = new Set<number>()

				for (const m of verificationMappings) {
					for (const [i, offset] of m.sourceOffsets.entries()) {
						sourceOffsets.add(offset)
						sourceOffsets.add(offset + m.lengths[i]!)
					}
					for (const [i, offset] of m.generatedOffsets.entries()) {
						serviceOffsets.add(offset)
						serviceOffsets.add(offset + (m.generatedLengths ?? m.lengths)[i]!)
					}
				}

				const sourceOffsetsUtf8 = new Map<number, number>()
				const serviceOffsetsUtf8 = new Map<number, number>()

				let currentUtf8Pos = 0
				const sortedSourceOffsets = Array.from(sourceOffsets).sort((a, b) => a - b)
				for (const [i, offset] of sortedSourceOffsets.entries()) {
					sourceOffsetsUtf8.set(offset, currentUtf8Pos += Buffer.byteLength(sourceText.slice(sortedSourceOffsets[i-1] ?? 0, offset)))
				}
				currentUtf8Pos = 0
				const sortedServiceOffsets = Array.from(serviceOffsets).sort((a, b) => a - b)
				for (const [i, offset] of sortedServiceOffsets.entries()) {
					serviceOffsetsUtf8.set(offset, currentUtf8Pos += Buffer.byteLength(serviceText.slice(sortedServiceOffsets[i-1] ?? 0, offset)))
				}

				const serviceCovered: [number, number][] = []
				const mappings = verificationMappings
					.flatMap((m): Mapping[] => {
						return m.sourceOffsets.map((sourceOffset, i) => {
							const generatedOffset = m.generatedOffsets[i]!
							const sourceLength = m.lengths[i]!
							const generatedLength = m.generatedLengths?.[i] ?? sourceLength
							if (generatedLength > 0) {
								serviceCovered.push([serviceOffsetsUtf8.get(generatedOffset)!, serviceOffsetsUtf8.get(generatedOffset + generatedLength)!])
							}

							const sourceOffsetUtf8 = sourceOffsetsUtf8.get(sourceOffset)!
							const generatedOffsetUtf8 = serviceOffsetsUtf8.get(generatedOffset)!
							return {
								sourceOffset: sourceOffsetUtf8,
								serviceOffset: generatedOffsetUtf8,
								sourceLength: sourceOffsetsUtf8.get(sourceOffset + sourceLength)! - sourceOffsetUtf8,
								serviceLengths: serviceOffsetsUtf8.get(generatedOffset + generatedLength)! - generatedOffsetUtf8,
							}
						})
					})

  			serviceCovered.sort((a, b) => a[0] - b[0] || a[1] - b[1]);

  			const merged: [number, number][] = [];
  			for (const [s, e] of serviceCovered) {
  			  const last = merged.at(-1);
  			  if (!last) {
						merged.push([s, e])
					} else if (s <= last[1]) {
						last[1] = Math.max(last[1], e)
					} else {
						merged.push([s, e]);
					}
  			}

  			const ignoreMappings: IgnoreDirectiveMapping[] = [];
  			let cursor = 0;

  			for (const [s, e] of merged) {
  			  if (s > cursor) {
						ignoreMappings.push({ serviceOffset: cursor, serviceLength: s - cursor });
					}
  			  cursor = Math.max(cursor, e);
  			}

  			ignoreMappings.push({ serviceOffset: cursor, serviceLength: Buffer.byteLength(serviceText) - cursor });

				return {
					serviceText,
					scriptKind: tsScriptKindToGolar(serviceScript?.scriptKind),
					mappings,
					ignoreMappings,
				}
			}
			throw new Error('Unknown language')
		},
	})
}

function tsScriptKindToGolar(scriptKind: ts.ScriptKind | undefined): ScriptKind {
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
