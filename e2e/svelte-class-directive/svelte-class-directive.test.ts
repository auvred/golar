import util from 'node:util'
import path from 'node:path'
import { test, expect } from 'vitest'
import { runGolar } from '../utils.ts'

// A rule can report on text the service code doesn't contain by giving a
// range in the original file.
test('lint report on a Svelte class directive name', async () => {
	const res = await runGolar({
		cwd: path.join(import.meta.dirname, 'fixture'),
		args: ['lint'],
	})

	expect(util.stripVTControlCharacters(res.output)).toMatchInlineSnapshot(`
		"Using config from ./golar.config.ts...
		comp.svelte:5:25: class-directives: Class directive 'is-open'."
	`)
})
