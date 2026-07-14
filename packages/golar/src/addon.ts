import os from 'node:os'
import process from 'node:process'
import url from 'node:url'
import worker_threads from 'node:worker_threads'
import { family as libcFamily, GLIBC, MUSL } from 'detect-libc'

const addonModule = {
	exports: {} as {
		setSyncBuffer(threadId: number, buffer: ArrayBuffer): void
		registerJsCodegen(threadId: number, createServiceCode: () => void): void
		registerIpcCodegen(): void
		tsc(threadId: number, done: (exitCode: number) => void): void

		workspace_New(cb: () => void): void
		workspace_ReadRequestedFileAt(threadId: number): void
		workspace_ReadFileById(threadId: number): void
		workspace_GetTypeAtLocation(threadId: number): void
		workspace_Lint(): void
		workspace_TypeCheck(): number
		workspace_Report(threadId: number): void
	},
}

let addonPackageName = `@golar/${process.platform}-${process.arch}`
let _isMusl = false
if (process.platform === 'linux') {
	const family = await libcFamily()
	switch (family) {
		case GLIBC:
			break
		case MUSL:
			addonPackageName = `@golar/${process.platform}-${process.arch}-musl`
			_isMusl = true
			break
		default:
			throw new Error('Unknown Linux libc family')
	}
}

export const isMusl = _isMusl
export const golarAddonPath = url.fileURLToPath(
	import.meta.resolve(`${addonPackageName}/golar.node`),
)

let addonLoaded = false
export function loadAddon() {
	if (addonLoaded) {
		return
	}
	const signals = [
		['SIGINT', 2],
		['SIGTERM', 15],
	] as const
	for (const [sig, code] of signals) {
		// exit directly to prevent recursive re-entry when Node re-raises the signal
		process.once(sig, () => process.exit(128 + code))
	}

	addonLoaded = true
	process.dlopen(
		addonModule,
		golarAddonPath,
		os.constants.dlopen.RTLD_NOW | os.constants.dlopen.RTLD_GLOBAL,
	)
	addon.setSyncBuffer(worker_threads.threadId, syncBuf)
}
export const { exports: addon } = addonModule

// TODO: grow dynamically?
export const syncBuf = new ArrayBuffer(10 * 1024 * 1024)
export const syncView = new DataView(syncBuf)
