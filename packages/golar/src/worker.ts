import assert from 'node:assert/strict'
import worker_threads from 'node:worker_threads'

import { loadConfig, resolveConfig } from './config.ts'
import { globalState, JsCodegenPlugin } from './codegen-plugin.ts'
import { Workspace } from './workspace.ts'
import { loadAddon } from './addon.ts'

export type WorkspaceMeta = {
	pointerLo: number
	pointerHi: number
	programsCount: number
	sourceFilesCount: number
}

export type WorkerEnv = {
	configPath: string
	cwd: string
	mode: 'codegen-only' | 'lint'
}

export type WorkerLintMessage = {
	meta: WorkspaceMeta
	jsFiles: string[]
}

const { parentPort } = worker_threads
assert.ok(parentPort != null)
const env = worker_threads.getEnvironmentData('golar-env') as WorkerEnv

const config = await loadConfig(env.configPath)

if (env.mode === 'codegen-only') {
	for (const plugin of globalState.codegenPlugins.values()) {
		// only js plugins must be registered from worker threads
		if (plugin instanceof JsCodegenPlugin) {
			plugin.register()
		}
	}
	parentPort.postMessage('done')
} else if (env.mode === 'lint') {
	// JS rules call into the addon from this thread even when no JS codegen
	// plugin is registered (registering one is what loads it otherwise).
	loadAddon()
	for (const plugin of globalState.codegenPlugins.values()) {
		// only js plugins must be registered from worker threads
		if (plugin instanceof JsCodegenPlugin) {
			plugin.register()
		}
	}
	parentPort.once('message', (message: WorkerLintMessage) => {
		const { files, jsRulesByFile } = resolveConfig(env.cwd, config)

		const ws = Workspace.createWorker(
			message.meta.pointerLo,
			message.meta.pointerHi,
			message.meta.programsCount,
			message.meta.sourceFilesCount,
			env.cwd,
			files,
		)
		ws.preloadRequestedFiles(message.jsFiles)
		ws.lintJs(
			new Map(
				message.jsFiles.map((file) => [
					file,
					Array.from(jsRulesByFile.get(file)!.values()),
				]),
			),
		)
		parentPort.postMessage('done')
	})
}
