import path from 'node:path'
import {build} from 'esbuild'

const typescriptSrcPath = path.join(import.meta.dirname, '../../../thirdparty/typescript/src')
function tsImport(relpath: string) {
	return JSON.stringify(path.join(typescriptSrcPath, relpath))
}

const typescriptLite = `
export * from ${tsImport("./compiler/corePublic.ts")};
export * from ${tsImport("./compiler/commandLineParser.ts")};
export { findConfigFile } from ${tsImport("./compiler/program.ts")};
export * from ${tsImport("./compiler/types.ts")};
export * from ${tsImport("./compiler/factory/nodeTests.ts")};
export * from ${tsImport("./compiler/scanner.ts")};
export * from ${tsImport("./compiler/sys.ts")};
export * from ${tsImport("./compiler/parser.ts")};
export * from ${tsImport("./compiler/utilitiesPublic.ts")};
export { getTokenPosOfNode } from ${tsImport("./compiler/utilities.ts")};
`

await build({
	entryPoints: ['virtual:entry'],
	outfile: path.join(import.meta.dirname, '../src/typescript-lite.js'),
	write: true,
	banner: {
		js: `
const require = (await import("node:module")).createRequire(import.meta.url);
const __filename = (await import("node:url")).fileURLToPath(import.meta.url);
const __dirname = (await import("node:path")).dirname(__filename);
`
	},
	format: 'esm',
	platform: 'node',
	bundle: true,
	plugins: [
		{
			name: 'load-entry',
			setup(build) {
			  build.onResolve({filter: /virtual:entry/}, () => ({
						path: 'virtual:entry',
						namespace: 'virtual:entry',
				}))

				build.onLoad({ filter: /.*/, namespace: 'virtual:entry' }, () => ({
					contents:typescriptLite,
					resolveDir: import.meta.dirname,
				}))
			},
		}
	]
})
