#!/usr/bin/env node
import path from 'node:path'
import process from 'node:process'
import child_process from 'node:child_process'
import { styleText } from './utils.ts'

if (process.versions.webcontainer != null) {
	console.error(
		`${styleText('red', 'Error:')} Golar does not currently support running in a WebContainer`,
	)
	process.exit(1)
}

const { golarAddonPath, isMusl } = await import('./addon.ts')

if (isMusl) {
	const selfExtname = path.extname(import.meta.filename)

	const cliPath = path.join(import.meta.dirname, `cli${selfExtname}`)
	const env = {
		...process.env,
		// workaround https://github.com/golang/go/issues/54805
		LD_PRELOAD:
			golarAddonPath +
			((process.env.LD_PRELOAD?.length ?? 0) > 0
				? ` ${process.env.LD_PRELOAD}`
				: ''),
	}

	process.execve?.(
		process.execPath,
		[process.execPath, cliPath, ...process.argv.slice(2)],
		env,
	)

	try {
		child_process.execFileSync(
			process.execPath,
			[cliPath, ...process.argv.slice(2)],
			{ stdio: 'inherit', env },
		)
	} catch (e) {
		if (e instanceof Error && 'status' in e && typeof e.status === 'number') {
			process.exit(e.status)
		}
		throw e
	}
} else {
	await import('./cli.ts')
}
