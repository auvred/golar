import util from 'node:util'
import path from 'node:path'
import { test, expect } from 'vitest'
import { SubprocessError } from 'nano-spawn'
import { runGolar } from '../utils.ts'

// svelte-check drops errors in svelte2tsx's `Ωignore` regions and the missing
// third argument of transition calls; Golar should report the same.
test('svelte-check diagnostic filters', async () => {
	const res = await runGolar({
		cwd: path.join(import.meta.dirname, 'fixture'),
		args: [],
	})

	expect(res).instanceof(SubprocessError)
	expect(util.stripVTControlCharacters(res.output)).toMatchInlineSnapshot(`
		"Using config from ./golar.config.ts...
		comp.svelte(14,9): error TS2322: Type 'number' is not assignable to type 'string'."
	`)
})
