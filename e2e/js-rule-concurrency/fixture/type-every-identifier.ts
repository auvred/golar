import { defineRule } from 'golar/unstable'
import { isIdentifier, TypeFlags } from 'golar/unstable-tsgo'

// Asks the checker for the type of every identifier, so that rules running
// on several threads query the same program at the same time.
export const typeEveryIdentifier = defineRule({
	name: 'type-every-identifier',
	setup(context) {
		context.sourceFile.forEachChild(function visit(node) {
			if (isIdentifier(node)) {
				const type = context.program.getTypeAtLocation(node)
				if (
					type != null &&
					(type.flags & TypeFlags.Any) !== 0 &&
					type.intrinsicName === 'any'
				) {
					context.report({
						message: `'${node.text}' is any.`,
						range: { begin: node.pos, end: node.end },
					})
				}
			}
			node.forEachChild(visit)
		})
	},
})
