import util from 'node:util'
import path from 'node:path'
import { test, expect } from 'vitest'
import { SubprocessError } from 'nano-spawn'
import { runGolar } from '../utils.ts'

test('@vue-expect-error', async () => {
	const res = await runGolar({
		cwd: path.join(import.meta.dirname, 'fixture'),
		args: ['tsc', '--noEmit', '--pretty', 'false'],
	})

	expect(res).instanceof(SubprocessError)
	expect(util.stripVTControlCharacters(res.output)).toMatchInlineSnapshot(`
		"Using config from ./golar.config.ts...
		nested-directives.vue(2,2): error TS2578: Unused '@ts-expect-error' directive.
		nested-directives.vue(12,2): error TS2578: Unused '@ts-expect-error' directive.
		nested-directives.vue(14,6): error TS2339: Property 'missing' does not exist on type 'ComponentPublicInstance<{}, {}, {}, {}, {}, {}, {}, {}, false, ComponentOptionsBase<any, any, any, any, any, any, any, any, any, {}, {}, string, {}, {}, {}, string, ComponentProvideOptions>, ... 4 more ..., any>'.
		nested-directives.vue(16,2): error TS2578: Unused '@ts-expect-error' directive.
		nested-directives.vue(19,4): error TS2578: Unused '@ts-expect-error' directive.
		suppressed-unknown-component.vue(2,2): error TS2578: Unused '@ts-expect-error' directive."
	`)
})
