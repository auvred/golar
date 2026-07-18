import {
	createParsedCommandLine,
	forEachEmbeddedCode,
} from '@vue/language-core'
import * as languageCore from '@vue/language-core'
import * as ts from './typescript-lite.js'
import compilerDom from '@vue/compiler-dom'
import { createParsedCommandLineByJson } from '@vue/language-core'
import { VueVirtualCode } from '@vue/language-core/lib/virtualCode/index.js'
import PluginVueTsx from '@vue/language-core/lib/plugins/vue-tsx.js'
import PluginFileVue from '@vue/language-core/lib/plugins/file-vue.js'
import PluginVueScriptJs from '@vue/language-core/lib/plugins/vue-script-js.js'
import PluginVueTemplateHtml from '@vue/language-core/lib/plugins/vue-template-html.js'
import PluginVueStyleCSS from '@vue/language-core/lib/plugins/vue-style-css.js'
import PluginFileMd from '@vue/language-core/lib/plugins/file-md.js'
import type { VolarLanguagePlugin } from '@golar/volar'

const SUPPRESSED_DIAGNOSTIC_CODES = [2339, 2551, 2353, 2561, 6133] as const

class GolarVueVirtualCode extends VueVirtualCode {
	override get embeddedCodes() {
		const embeddedCodes = super.embeddedCodes
		const walk = (code: (typeof embeddedCodes)[number]) => {
			for (const m of code.mappings) {
				const data = m.data as {
					verification?: unknown
					__suppressedDiagnostics?: number[]
					__expectErrorCommentLoc?: [number, number]
				}
				if (data.__suppressedDiagnostics != null) {
					continue
				}
				// Mappings covered by a `@vue-expect-error` directive are handled by the
				// range-based expect-error mechanism (see `__expectErrorCommentLoc` in
				// @golar/volar). Baking `__suppressedDiagnostics` here as well would drop the
				// diagnostic before the directive can consume it, causing the directive to be
				// reported as an unused `@ts-expect-error` (TS2578).
				if ('__expectErrorCommentLoc' in data) {
					continue
				}
				if (
					data.verification == null ||
					typeof data.verification !== 'object' ||
					!('shouldReport' in data.verification) ||
					typeof data.verification.shouldReport !== 'function'
				) {
					continue
				}

				const shouldReport = data.verification.shouldReport as (
					source: string | undefined,
					code: string | number,
				) => boolean
				const suppressed = SUPPRESSED_DIAGNOSTIC_CODES.filter(
					(code) => !shouldReport(undefined, code),
				)
				if (suppressed.length) {
					data.__suppressedDiagnostics = [...suppressed]
				}
			}
			for (const child of code.embeddedCodes ?? []) {
				walk(child)
			}
		}
		for (const code of embeddedCodes) {
			walk(code)
		}
		return embeddedCodes
	}
}

export function vueLanguagePlugin(
	cwd: string,
	configFileName: string | null,
): VolarLanguagePlugin {
	const { options: compilerOptions, vueOptions: vueCompilerOptions } =
		configFileName == null
			? createParsedCommandLineByJson(ts, ts.sys, cwd, {})
			: createParsedCommandLine(ts, ts.sys, configFileName)
	const plugins = [
		PluginVueTsx.default,
		PluginFileVue.default,
		PluginVueScriptJs.default,
		PluginVueTemplateHtml.default,
		PluginVueStyleCSS.default,
		PluginFileMd.default,
		...vueCompilerOptions.plugins,
	].flatMap((ctor) =>
		ctor({
			modules: {
				typescript: ts,
				'@vue/compiler-dom': compilerDom,
				'@vue/language-core': languageCore,
			},
			compilerOptions,
			vueCompilerOptions,
			config: {},
		}),
	)
	return {
		getLanguageId(scriptId) {
			for (const plugin of plugins) {
				const languageId = plugin.getLanguageId?.(scriptId)
				if (languageId) {
					return languageId
				}
			}
			return undefined
		},
		createVirtualCode(scriptId, languageId, snapshot) {
			return new GolarVueVirtualCode(
				scriptId,
				languageId,
				snapshot,
				vueCompilerOptions,
				plugins,
				ts,
			)
		},
		getVirtualCodeErrors(root) {
			return (root as VueVirtualCode)
				.vueSfc!.errors.filter((e) => 'code' in e)
				.map((e) => ({
					start: e.loc?.start.offset ?? 0,
					end: e.loc?.end.offset ?? 0,
					message: e.message,
				}))
		},
		typescript: {
			extraFileExtensions: [],
			getServiceScript(root) {
				for (const code of forEachEmbeddedCode(root)) {
					if (/script_(js|jsx|ts|tsx)/.test(code.id)) {
						const lang = code.id.slice('script_'.length)
						return {
							code,
							extension: '.' + lang,
							scriptKind:
								lang === 'js'
									? ts.ScriptKind.JS
									: lang === 'jsx'
										? ts.ScriptKind.JSX
										: lang === 'tsx'
											? ts.ScriptKind.TSX
											: ts.ScriptKind.TS,
						}
					}
				}
				return undefined
			},
		},
	}
}
