import path from 'node:path'
import util from 'node:util'
import { test, expect } from 'vitest'
import { SubprocessError } from 'nano-spawn'
import { runGolar } from '../utils.ts'

test('vue pug template typecheck', async () => {
	const res = await runGolar({
		cwd: path.join(import.meta.dirname, 'fixture'),
		args: ['tsc', '--noEmit', '--pretty'],
	})

	expect(res).instanceof(SubprocessError)
	expect(util.stripVTControlCharacters(res.output)).toMatchInlineSnapshot(`
		"Using config from ./golar.config.ts...
		comp.vue:8:28 - error TS2345: Argument of type 'number' is not assignable to parameter of type 'string'.

		8 button(@click=\"handleClick(123)\") Click
		                             ~~~


		Found 1 error in comp.vue:8
		"
	`)
})
