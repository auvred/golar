import util from 'node:util'
import path from 'node:path'
import { test, expect } from 'vitest'
import { runGolar } from '../utils.ts'

// Nodes handed to JS rules carry UTF-16 positions; a report after non-ASCII
// text must still land on the reported node.
test('JS rule report after non-ASCII text', async () => {
	const res = await runGolar({
		cwd: path.join(import.meta.dirname, 'fixture'),
		args: ['lint'],
	})

	expect(util.stripVTControlCharacters(res.output)).toMatchInlineSnapshot(`
		"Using config from ./golar.config.ts...
		index.ts:2:9: no-property-access: Property access 'log'."
	`)
})
