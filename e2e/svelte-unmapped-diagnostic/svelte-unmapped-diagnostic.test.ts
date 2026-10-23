import util from 'node:util'
import path from 'node:path'
import { test, expect } from 'vitest'
import { SubprocessError } from 'nano-spawn'
import { runGolar } from '../utils.ts'

// svelte2tsx reports a missing prop on the generated `props: {}` object,
// which has no source mapping. The error belongs next to `<Child />`, not
// at the start of the file.
test('diagnostic on generated Svelte code', async () => {
	const res = await runGolar({
		cwd: path.join(import.meta.dirname, 'fixture'),
		args: [],
	})

	expect(res).instanceof(SubprocessError)
	expect(util.stripVTControlCharacters(res.output)).toMatchInlineSnapshot(`
		"Using config from ./golar.config.ts...
		Parent.svelte(5,7): error TS2741: Property 'name' is missing in type '{}' but required in type '$$ComponentProps'."
	`)
})
