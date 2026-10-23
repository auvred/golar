import util from 'node:util'
import path from 'node:path'
import { test, expect } from 'vitest'
import { SubprocessError } from 'nano-spawn'
import { runGolar } from '../utils.ts'

test('a .svelte file next to a same-named .ts file stays in its tsconfig program', async () => {
	const res = await runGolar({
		cwd: path.join(import.meta.dirname, 'fixture'),
		args: [],
	})

	expect(res).instanceof(SubprocessError)
	expect(util.stripVTControlCharacters(res.output)).toMatchInlineSnapshot(`
		"Using config from ./golar.config.ts...
		src/routes/+page.svelte(3,9): error TS2322: Type 'number' is not assignable to type 'string'."
	`)
})
