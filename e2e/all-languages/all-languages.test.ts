import { test, expect } from 'vitest'
import path from 'node:path'
import util from 'node:util'
import { SubprocessError } from 'nano-spawn'
import { runGolar } from '../utils.ts'

test('all languages', async () => {
	const res = await runGolar({
		cwd: path.join(import.meta.dirname, 'fixture'),
		args: ['--noEmit', '--pretty'],
		plugins: {
			astro: true,
			ember: true,
			svelte: true,
			vue: true,
		},
	})

	expect(res).instanceof(SubprocessError)
	expect(util.stripVTControlCharacters(res.output)).toMatchInlineSnapshot(`
		"comp.astro:5:7 - error TS2322: Type 'number' is not assignable to type 'string'.

		5 const astro: string = 123
		        ~~~~~

		comp.astro:8:8 - error TS2304: Cannot find name 'unknownVar'.

		8 <div>{ unknownVar }</div>
		         ~~~~~~~~~~

		comp.gts:1:7 - error TS2322: Type 'number' is not assignable to type 'string'.

		1 const ember: string = 123
		        ~~~~~

		comp.gts:3:30 - error TS2304: Cannot find name 'unknownVar'.

		3 export default <template> {{ unknownVar }} </template>
		                               ~~~~~~~~~~

		comp.svelte:2:9 - error TS2322: Type 'number' is not assignable to type 'string'.

		2   const svelte: string = 123
		          ~~~~~~

		comp.svelte:5:19 - error TS2304: Cannot find name 'unknownVar'.

		5 <button on:click={unknownVar}></button>
		                    ~~~~~~~~~~

		comp.vue:2:7 - error TS2322: Type 'number' is not assignable to type 'string'.

		2 const vue: string = 123
		        ~~~

		comp.vue:6:10 - error TS2339: Property 'unknownVar' does not exist on type 'ComponentPublicInstance<{}, {}, {}, {}, {}, {}, {}, {}, false, ComponentOptionsBase<any, any, any, any, any, any, any, any, any, {}, {}, string, {}, {}, {}, string, ComponentProvideOptions>, ... 5 more ..., any>'.

		6  <div>{{ unknownVar }}</div>
		           ~~~~~~~~~~


		Found 8 errors in 4 files.

		Errors  Files
		     2  comp.astro:5
		     2  comp.gts:1
		     2  comp.svelte:2
		     2  comp.vue:2
		"
	`)
})
