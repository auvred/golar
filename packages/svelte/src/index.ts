import { svelte2tsx } from 'svelte2tsx'
import { createPlugin } from '@golar/plugin'

createPlugin({
	filename: import.meta.filename,
	extraExtensions: ['.svelte'],
	createServiceCode(fileName, sourceText) {
		// TODO: handle parsing errors
		// TODO: .d.ts references
		const tsx = svelte2tsx(sourceText, {
			isTsFile: true,
			mode: "ts",
		});

		return {
			serviceText: tsx.code,
			// TODO
			scriptKind: 'tsx',
			sourceMap: tsx.map.mappings,
		}
	},
})
