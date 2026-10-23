import process from 'node:process'
import os from 'node:os'
import path from 'node:path'
import assert from 'node:assert/strict'
import worker_threads from 'node:worker_threads'
import fs from 'node:fs'

import { loadConfig, resolveConfig, type Config } from './config.ts'
import { globalState, JsCodegenPlugin } from './codegen-plugin.ts'
import { addon, loadAddon, syncBuf, syncView } from './addon.ts'
import { Workspace } from './workspace.ts'
import type { WorkerEnv, WorkerLintMessage } from './worker.ts'
import { styleText } from './utils.ts'

const argv = process.argv.slice(2)

const cwd = process.cwd()
// TODO: find-up?
let configPath = path.join(cwd, 'golar.config.ts')

if (argv[0] === '--help') {
	console.log(`
Usage: golar [command]

Commands:
  golar                    Lint and typecheck the current workspace (recommended)
  golar lint               Run lint checks only
  golar typecheck          Run typechecking only
  golar tsc [tsc args...]  Forward arguments to TypeScript CLI

Options:
  --help                   Show this help message
  --version                Print Golar version`)
	process.exit(0)
} else if (argv[0] === '--version') {
	const {
		default: { version },
	} = await import('../package.json', { with: { type: 'json' } })
	console.log(`Golar version ${version}`)
	process.exit(0)
}

if (!fs.existsSync(configPath)) {
	try {
		configPath = await Promise.any(
			[
				'golar.config.mts',
				'golar.config.mjs',
				'golar.config.cts',
				'golar.config.cjs',
				'golar.config.js',
			].map((p) => fs.promises.stat(p).then(() => path.join(cwd, p))),
		)
	} catch {
		console.log(`${styleText('red', 'Error:')} ./golar.config.ts not found`)
		process.exit(1)
	}
}
console.log(
	`${styleText('dim', 'Using config from')} ./${path.basename(configPath)}${styleText('dim', '...')}`,
)

loadAddon()

const config = await loadConfig(configPath)
// TODO: error message; here and in other places

const hasJsCodegenPlugins = globalState.codegenPlugins
	.values()
	.some((v) => v instanceof JsCodegenPlugin)

const selfExtname = path.extname(import.meta.filename)
const workerPath = path.join(import.meta.dirname, `worker${selfExtname}`)

const textEncoder = new TextEncoder()

if (argv[0] === 'tsc') {
	for (const plugin of globalState.codegenPlugins.values()) {
		plugin.register()
	}

	const args = argv.slice(1)

	let offset = 0
	syncView.setUint32(offset, args.length, true)
	offset += 4
	for (const arg of args) {
		const { written: argLen } = textEncoder.encodeInto(
			arg,
			new Uint8Array(syncBuf, offset + 4),
		)
		syncView.setUint32(offset, argLen, true)
		offset += 4 + argLen
	}

	addon.tsc(worker_threads.threadId, (exitCode) => process.exit(exitCode))

	if (hasJsCodegenPlugins) {
		// spawn workers after tsc, so tsconfig parsing and initial module resolution
		// start as early as possible
		worker_threads.setEnvironmentData('golar-env', {
			configPath,
			cwd,
			mode: 'codegen-only',
		} satisfies WorkerEnv)

		// main thread is already created
		const workersCount =
			Math.floor(Math.max(Math.min(os.availableParallelism() / 2, 4), 1)) - 1

		const promises: Promise<void>[] = []
		for (let i = 0; i < workersCount; i++) {
			const w = new worker_threads.Worker(workerPath)
			promises.push(new Promise((resolve) => w.once('message', resolve)))
		}
		await Promise.all(promises)
	}
} else {
	const lintOnly = argv[0] === 'lint'
	const typecheckOnly = argv[0] === 'typecheck'

	// TODO: remove this in a little while (this helps folks to migrate from v0.0 to v0.1)
	if (
		!lintOnly &&
		!typecheckOnly &&
		[
			'--noEmit',
			'-b',
			'--build',
			'--declaration',
			'--emitDeclarationOnly',
		].some((flag) => argv.includes(flag))
	) {
		let message = `${styleText('red', 'Error:')} Golar v0.1+ doesn't support passing tsc flags to the root subcommand.\n`
		if (
			argv.includes('--declaration') ||
			argv.includes('--emitDeclarationOnly')
		) {
			message += `Instead, run: ${styleText('bold', `golar ${styleText('green', 'tsc')} ${argv.join(' ')}`)}`
		} else {
			message += `Instead, run ${styleText('bold', 'golar')} without any arguments.`
		}
		console.log(message)
		process.exit(1)
	}

	for (const plugin of globalState.codegenPlugins.values()) {
		plugin.register()
	}

	const {
		files,
		builtinRulesByFile,
		nativeRulesByFile,
		jsFilesByWorker,
		jsRulesByFile,
		typecheckFiles,
	} = resolveConfig(cwd, config)

	const { meta, workspace: wsPromise } = await Workspace.create(cwd, files)
	const ws = await wsPromise
	if (!typecheckOnly) {
		await ws.lintBuiltin(builtinRulesByFile)
		ws.lintNative(nativeRulesByFile, path.dirname(configPath))
		const promises: Promise<void>[] = []
		if (jsRulesByFile.size > 0) {
			worker_threads.setEnvironmentData('golar-env', {
				configPath,
				cwd,
				mode: 'lint',
			} satisfies WorkerEnv)
			// last chunk of files is the smallest (due to Math.ceil), it's processed by main thread
			const workers = jsFilesByWorker.slice(0, -1).map((workerFiles) => {
				const worker = new worker_threads.Worker(workerPath)
				promises.push(new Promise((resolve) => worker.once('message', resolve)))
				worker.postMessage({
					meta,
					jsFiles: workerFiles,
				} satisfies WorkerLintMessage)
				return worker
			})
		}
		if (jsRulesByFile.size > 0) {
			const mainJsFiles = jsFilesByWorker.at(-1)
			assert.ok(mainJsFiles)
			ws.preloadRequestedFiles(mainJsFiles)
			ws.lintJs(
				new Map(
					mainJsFiles.map((file) => [
						file,
						Array.from(jsRulesByFile.get(file)!.values()),
					]),
				),
			)
		}
		await Promise.all(promises)
	}
	const lintExitCode = !typecheckOnly && ws.lintReportCount() > 0 ? 1 : 0
	if (!lintOnly) {
		const typecheckExitCode = await ws.typecheck(typecheckFiles)
		process.exit(typecheckExitCode || lintExitCode)
	}
	process.exit(lintExitCode)
}
