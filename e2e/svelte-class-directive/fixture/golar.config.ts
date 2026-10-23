import fs from 'node:fs'
import '@golar/svelte'
import { defineConfig, defineRule } from 'golar/unstable'

// svelte2tsx keeps only the expression of `class:is-open={open}`, so the
// class name exists only in the original file text.
const classDirectives = defineRule({
	name: 'class-directives',
	setup(context) {
		const text = fs.readFileSync(context.sourceFile.fileName, 'utf8')
		for (const match of text.matchAll(/class:([\w-]+)/g)) {
			const begin = match.index + 'class:'.length
			context.report({
				message: `Class directive '${match[1]}'.`,
				sourceRange: { begin, end: begin + match[1]!.length },
			})
		}
	},
})

export default defineConfig({
	lint: {
		use: [
			{
				files: ['comp.svelte'],
				rules: [classDirectives],
			},
		],
	},
	typecheck: {
		include: [],
	},
})
