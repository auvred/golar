import util from 'node:util'
import path from 'node:path'
import { test, expect } from 'vitest'
import { runGolar } from '../utils.ts'

// Type-aware JS rules run on several threads at once. Each thread queries the
// same program, so a checker shared between threads crashes the process with
// a data race ('concurrent map read and map write' and friends).
test(
	'type-aware JS rules on several threads',
	{ timeout: 60_000 },
	async () => {
		for (let run = 0; run < 3; run++) {
			const res = await runGolar({
				cwd: path.join(import.meta.dirname, 'fixture'),
				args: ['lint'],
			})

			expect(util.stripVTControlCharacters(res.output)).toBe(
				'Using config from ./golar.config.ts...',
			)
		}
	},
)
