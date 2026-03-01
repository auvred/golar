import os from 'node:os'
import path from 'node:path'
import spawn, { SubprocessError } from 'nano-spawn'

const repoRoot = path.join(import.meta.dirname, '..')
const ext = os.platform() === 'win32' ? '.exe' : ''

export async function runGolar(opts: {
	cwd: string
	args: string[]
	plugins: {
		astro?: boolean
		ember?: boolean
		svelte?: boolean
		vue?: boolean
	}
}) {
	const plugins = [
		opts.plugins.astro && [
			path.join(repoRoot, 'packages', 'astro', 'astro' + ext),
		],
		opts.plugins.ember && [
			process.execPath,
			path.join(repoRoot, 'packages', 'ember', 'src', 'bin.ts'),
		],
		opts.plugins.svelte && [
			process.execPath,
			path.join(repoRoot, 'packages', 'svelte', 'src', 'bin.ts'),
		],
		opts.plugins.vue && [
			process.execPath,
			path.join(repoRoot, 'packages', 'vue', 'src', 'bin.ts'),
		],
	]
		.filter((cmd) => !!cmd)
		.map((cmd) => cmd.join('\x1f'))
		.join('\x1e')

	return await spawn(path.join(repoRoot, 'golar' + ext), opts.args, {
		env: {
			GOLAR_PLUGINS: plugins,
		},
		cwd: opts.cwd,
	}).catch((e) => e as SubprocessError)
}
