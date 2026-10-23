import '@golar/svelte'
import { defineConfig } from 'golar/unstable'
import { typeEveryIdentifier } from './type-every-identifier.ts'

export default defineConfig({
	lint: {
		use: [
			{
				files: ['src/**/*.ts', 'src/**/*.svelte'],
				rules: [typeEveryIdentifier],
			},
		],
	},
})
