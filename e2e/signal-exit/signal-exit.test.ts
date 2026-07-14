import path from 'node:path'
import spawn from 'nano-spawn'
import { expect, onTestFinished, test, vi } from 'vitest'
import { setTimeout as sleep } from 'node:timers/promises'

const entryPath = path.join(import.meta.dirname, 'entry.ts')

test.each(['SIGINT', 'SIGHUP', 'SIGTERM'] as const)(
	'exits on %s',
	async (signal) => {
		const ctrl = new AbortController()
		setTimeout(() => ctrl.abort(), 1000)
		const proc = spawn(process.execPath, [entryPath], {
			signal: ctrl.signal,
			killSignal: signal,
		})

		await expect(proc).rejects.toThrow()
		await sleep(500)
		const nodeProc = await proc.nodeChildProcess
		expect(nodeProc.pid).toBeDefined()
		onTestFinished(() => {
			try {
				process.kill(nodeProc.pid!, 'SIGKILL')
			} catch {}
		})
		await vi.waitFor(() => {
			try {
				process.kill(nodeProc.pid!, 0)
			} catch {
				return
			}
			throw new Error('process still exists')
		})
	},
)
