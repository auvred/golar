import { defineConfig, defineRule } from 'golar/unstable'
import { isArrowFunction } from 'golar/unstable-tsgo'

const noArrowFunctions = defineRule({
	name: 'no-arrow-functions',
	setup(context) {
		context.sourceFile.forEachChild(function visit(node) {
			if (isArrowFunction(node)) {
				context.report({
					message: 'Arrow function.',
					range: { begin: node.pos, end: node.end },
				})
			}
			node.forEachChild(visit)
		})
	},
})

export default defineConfig({
	lint: {
		use: [
			{
				files: ['*.ts'],
				rules: [noArrowFunctions],
			},
		],
	},
	typecheck: {
		include: [],
	},
})
