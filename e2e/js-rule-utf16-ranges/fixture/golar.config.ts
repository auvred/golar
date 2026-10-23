import { defineConfig, defineRule } from 'golar/unstable'
import { isPropertyAccessExpression } from 'golar/unstable-tsgo'

const noPropertyAccess = defineRule({
	name: 'no-property-access',
	setup(context) {
		context.sourceFile.forEachChild(function visit(node) {
			if (isPropertyAccessExpression(node)) {
				context.report({
					message: `Property access '${node.name.text}'.`,
					range: { begin: node.name.pos, end: node.name.end },
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
				files: ['index.ts'],
				rules: [noPropertyAccess],
			},
		],
	},
	typecheck: {
		include: [],
	},
})
