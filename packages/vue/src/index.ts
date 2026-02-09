import { createVolarPlugin } from '@golar/volar'
import { forEachEmbeddedCode } from '@vue/language-core'
import * as ts from './typescript-lite.js'
import compilerDom from '@vue/compiler-dom'
import { createParsedCommandLineByJson } from '@vue/language-core'
import { VueVirtualCode } from '@vue/language-core/lib/virtualCode/index.js'
import PluginVueTsx from '@vue/language-core/lib/plugins/vue-tsx.js'
import PluginFileVue from '@vue/language-core/lib/plugins/file-vue.js'
import PluginVueScriptJs from '@vue/language-core/lib/plugins/vue-script-js.js'
import PluginVueTemplateHtml from '@vue/language-core/lib/plugins/vue-template-html.js'

const { vueOptions } = createParsedCommandLineByJson(ts, ts.sys, ts.sys.getCurrentDirectory(), {})

const plugins = (await Promise.all([
	PluginVueTsx,
	PluginFileVue,
	PluginVueScriptJs,
	PluginVueTemplateHtml,
])).flatMap(({ default: ctor }) => ctor({
	modules: {
		typescript: ts,
		"@vue/compiler-dom": compilerDom
	},
	compilerOptions: {},
	vueCompilerOptions: vueOptions,
}))

createVolarPlugin({
	filename: import.meta.filename,
	languagePlugins: [
		{
			getLanguageId(scriptId) {
			  return scriptId.endsWith('.vue') ? 'vue' : undefined
			},
			createVirtualCode(scriptId, languageId, snapshot) {
				return new VueVirtualCode(
					scriptId,
					languageId,
					snapshot,
					vueOptions,
					plugins,
					ts,
				);
			},
			typescript: {
				extraFileExtensions: [{
					extension: 'vue',
					isMixedContent: true,
					scriptKind: 7 satisfies import('typescript').ScriptKind.Deferred,
				}],
				getServiceScript(root) {
					for (const code of forEachEmbeddedCode(root)) {
						if (/script_(js|jsx|ts|tsx)/.test(code.id)) {
							const lang = code.id.slice('script_'.length);
							return {
								code,
								extension: '.' + lang,
								scriptKind: lang === 'js'
									? ts.ScriptKind.JS
									: lang === 'jsx'
									? ts.ScriptKind.JSX
									: lang === 'tsx'
									? ts.ScriptKind.TSX
									: ts.ScriptKind.TS,
							};
						}
					}
					return undefined
				}
			}
		},
	]
})
