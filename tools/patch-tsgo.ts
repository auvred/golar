import path from 'node:path'
import fs from 'node:fs/promises'
import spawn from 'nano-spawn'
import {
	applyPatches,
	commitUninternal,
	repoRoot,
	resetSubmodule,
	uninternal,
} from './utils.ts'

const dir = path.join(repoRoot, 'thirdparty', 'typescript-go')

await resetSubmodule(dir)

await fs.rm(path.join(dir, 'pkg'), {
	recursive: true,
	force: true,
})

await uninternal(
	[path.join(dir, 'cmd'), path.join(dir, 'internal')],
	'github.com/microsoft/typescript-go/internal',
	'github.com/microsoft/typescript-go/pkg',
)

await fs.rename(path.join(dir, 'internal'), path.join(dir, 'pkg'))

await commitUninternal(dir)

await applyPatches('typescript-go', dir)

await spawn('npm', ['install'], {
	cwd: dir,
	stdio: 'inherit',
})

await spawn('npx', ['tsc', '-b'], {
	cwd: path.join(dir, '_packages', 'native-preview'),
	stdio: 'inherit',
})
