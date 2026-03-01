import os from 'node:os'
import path from 'node:path'
import spawn from 'nano-spawn'
import { repoRoot } from './utils.ts'

const ext = os.platform() === 'win32' ? '.exe' : ''

await spawn(
	'go',
	[
		'build',
		'-o',
		path.join(repoRoot, 'golar' + ext),
		path.join(repoRoot, 'thirdparty', 'typescript-go', 'cmd', 'tsgo'),
	],
	{
		env: {
			CGO_ENABLED: '0',
		},
		stdio: 'inherit',
	},
)

await spawn(
	'go',
	[
		'build',
		'-o',
		path.join(repoRoot, 'packages', 'astro', 'astro' + ext),
		path.join(repoRoot, 'packages', 'astro'),
	],
	{
		env: {
			CGO_ENABLED: '0',
		},
		stdio: 'inherit',
	},
)
