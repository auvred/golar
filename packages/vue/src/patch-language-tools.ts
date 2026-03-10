import assert from 'node:assert/strict'
import fs from 'node:fs'
import { createRequire } from 'node:module'

const require = createRequire(import.meta.url)

const originalReadFileSync = fs.readFileSync
function patchFile(filePath: string, patch: (src: string) => string) {
	let patched = false
	// @ts-expect-error - TypeScript doesn't understand that the overloads do match up.
	fs.readFileSync = (...args) => {
		const src = originalReadFileSync(...args).toString()
		fs.readFileSync = originalReadFileSync
		patched = true
		return patch(src)
	}
	require(filePath)
	fs.readFileSync = originalReadFileSync
	assert.ok(
		patched,
		`Golar bug: File ${filePath} wasn't patched; most probably it has been already loaded`,
	)
}

patchFile('@vue/language-core/lib/codegen/template/context.js', (src) => {
	src = replaceOrThrow(
		src,
		'function createTemplateCodegenContext()',
		() => `function createTemplateCodegenContext(options)`,
	)
	src = replaceOrThrow(
		src,
		'function resolveCodeFeatures',
		(s) => `${s}(...args) {
		let features = _resolveCodeFeatures(...args)
		const data = stack.at(-1)
		if (data?.expectError != null) {
			features = {
				...features,
				__expectErrorCommentLoc: [
					(options?.template?.startTagEnd ?? 0) + data.expectError.node.loc.start.offset,
					(options?.template?.startTagEnd ?? 0) + data.expectError.node.loc.end.offset,
				],
			}
		}
		return features
	}

	function _resolveCodeFeatures`,
	)

	return src
})

patchFile('@vue/language-core/lib/codegen/template/index.js', (src) => {
	src = replaceOrThrow(
		src,
		'createTemplateCodegenContext)()',
		() => `createTemplateCodegenContext)(options)`,
	)

	return src
})

function replaceOrThrow(
	source: string,
	search: RegExp | string,
	replace: (substring: string, ...args: string[]) => string,
): string {
	const before = source
	source = source.replace(search, replace)
	const after = source
	if (after === before) {
		throw new Error('Golar bug: failed to replace: ' + search.toString())
	}
	return after
}
