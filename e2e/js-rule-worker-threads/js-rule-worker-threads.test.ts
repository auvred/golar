import util from 'node:util'
import path from 'node:path'
import { test, expect } from 'vitest'
import { runGolar } from '../utils.ts'

// With more than one linted file, JS rules also run on worker threads. The
// workers must load the addon themselves when no JS codegen plugin did.
test('JS rules on worker threads without codegen plugins', async () => {
	const res = await runGolar({
		cwd: path.join(import.meta.dirname, 'fixture'),
		args: ['lint'],
	})

	// threads report in any order
	const lines = util.stripVTControlCharacters(res.output).split('\n').sort()
	expect(lines).toMatchInlineSnapshot(`
		[
		  "Using config from ./golar.config.ts...",
		  "a.ts:1:17: no-arrow-functions: Arrow function.",
		  "b.ts:1:17: no-arrow-functions: Arrow function.",
		  "c.ts:1:17: no-arrow-functions: Arrow function.",
		  "d.ts:1:17: no-arrow-functions: Arrow function.",
		]
	`)
})
