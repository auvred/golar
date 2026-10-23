import util from 'node:util'
import path from 'node:path'
import { test, expect } from 'vitest'
import { SubprocessError } from 'nano-spawn'
import { runGolar } from '../utils.ts'

test('lint and typecheck', async () => {
	const res = await runGolar({
		cwd: path.join(import.meta.dirname, 'fixture'),
		args: [],
	})

	expect(res).instanceof(SubprocessError)
	expect(util.stripVTControlCharacters(res.output)).toMatchInlineSnapshot(`
		"Using config from ./golar.config.ts...
		index.ts:3:26: explicit-anys: Unexpected any. Specify a different type.
		index.ts:3:29: rust/unsafe-calls: Unsafe any call.
		index.vue:9:4: rust/unsafe-calls: Unsafe any call.
		index.vue:9:4: js/unsafe-calls: Unsafe any call.
		index.ts:3:29: js/unsafe-calls: Unsafe any call.
		index.ts(1,7): error TS2322: Type '"bar"' is not assignable to type '"foo"'.
		index.vue(8,13): error TS2345: Argument of type 'number' is not assignable to parameter of type 'string'."
	`)
})

test('lint findings fail `golar lint`', async () => {
	const res = await runGolar({
		cwd: path.join(import.meta.dirname, 'fixture'),
		args: ['lint'],
	})

	expect(res).instanceof(SubprocessError)
	expect((res as SubprocessError).exitCode).toBe(1)
})
