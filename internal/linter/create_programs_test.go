package linter

import (
	"encoding/binary"
	"testing"
	"unsafe"

	_ "github.com/auvred/golar/internal/golar"

	apiencoder "github.com/microsoft/typescript-go/pkg/api/encoder"
	"github.com/microsoft/typescript-go/pkg/ast"
	"github.com/microsoft/typescript-go/pkg/bundled"
	"github.com/microsoft/typescript-go/pkg/compiler"
	"github.com/microsoft/typescript-go/pkg/tspath"
	"github.com/microsoft/typescript-go/pkg/vfs/vfstest"
	"gotest.tools/v3/assert"
)

const (
	inferredProjectKey   = "__inferred__"
	anyConfiguredProject = "__configured__"
)

func TestCreatePrograms_EmptyInput(t *testing.T) {
	t.Parallel()

	programs, orderedPrograms, _ := createPrograms(t, map[string]any{}, nil, true)
	assert.Assert(t, programs == nil)
	assert.Assert(t, orderedPrograms == nil)
	assert.Equal(t, len(programs), 0)
	assert.Equal(t, len(orderedPrograms), 0)
}

func TestCreatePrograms_ReturnsProgramsInFirstInputOrder(t *testing.T) {
	t.Parallel()

	files := map[string]any{
		"/repo/p1/tsconfig.json": `{"include": ["src/**/*.ts"]}`,
		"/repo/p1/src/a.ts":      `export const a = 1;`,
		"/repo/p1/src/b.ts":      `export const b = 2;`,
		"/repo/p2/tsconfig.json": `{"include": ["src/**/*.ts"]}`,
		"/repo/p2/src/c.ts":      `export const c = 3;`,
	}

	programs, orderedPrograms, opts := createPrograms(t, files, []string{
		"/repo/p2/src/c.ts",
		"/repo/p1/src/a.ts",
		"/repo/p1/src/b.ts",
	}, true)

	assert.Equal(t, len(orderedPrograms), 2)
	assert.Assert(t, orderedPrograms[0] == requireProgramForInput(t, programs, "/repo/p2/src/c.ts", opts).Program)
	assert.Assert(t, orderedPrograms[1] == requireProgramForInput(t, programs, "/repo/p1/src/a.ts", opts).Program)
	assert.Assert(t, orderedPrograms[1] == requireProgramForInput(t, programs, "/repo/p1/src/b.ts", opts).Program)
	assert.Equal(t, len(programs), 3)
}

func TestCreatePrograms_AssignmentEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		files            map[string]any
		inputs           []string
		useCaseSensitive bool
		expectedKeys     map[string]string
		sameGroups       [][]string
		differentPairs   [][2]string
	}{
		{
			name: "reuse configured program for same config",
			files: map[string]any{
				"/repo/tsconfig.json": `{"include": ["src/**/*.ts"]}`,
				"/repo/src/a.ts":      `export const a = 1;`,
				"/repo/src/b.ts":      `export const b = 2;`,
			},
			inputs: []string{"/repo/src/a.ts", "/repo/src/b.ts"},
			expectedKeys: map[string]string{
				"/repo/src/a.ts": "/repo/tsconfig.json",
				"/repo/src/b.ts": "/repo/tsconfig.json",
			},
			sameGroups: [][]string{{"/repo/src/a.ts", "/repo/src/b.ts"}},
		},
		{
			name: "assign different configs to different programs",
			files: map[string]any{
				"/repo/p1/tsconfig.json": `{"include": ["src/**/*.ts"]}`,
				"/repo/p1/src/a.ts":      `export const a = 1;`,
				"/repo/p2/tsconfig.json": `{"include": ["src/**/*.ts"]}`,
				"/repo/p2/src/b.ts":      `export const b = 2;`,
			},
			inputs: []string{"/repo/p1/src/a.ts", "/repo/p2/src/b.ts"},
			expectedKeys: map[string]string{
				"/repo/p1/src/a.ts": "/repo/p1/tsconfig.json",
				"/repo/p2/src/b.ts": "/repo/p2/tsconfig.json",
			},
			differentPairs: [][2]string{{"/repo/p1/src/a.ts", "/repo/p2/src/b.ts"}},
		},
		{
			name: "normalize relative and duplicate-path inputs",
			files: map[string]any{
				"/repo/tsconfig.json": `{"include": ["src/**/*.ts"]}`,
				"/repo/src/a.ts":      `export const a = 1;`,
			},
			inputs: []string{"/repo/src/a.ts", "/repo/src/../src/a.ts", "repo/src/a.ts"},
			expectedKeys: map[string]string{
				"/repo/src/a.ts":        "/repo/tsconfig.json",
				"/repo/src/../src/a.ts": "/repo/tsconfig.json",
				"repo/src/a.ts":         "/repo/tsconfig.json",
			},
			sameGroups: [][]string{{"/repo/src/a.ts", "/repo/src/../src/a.ts", "repo/src/a.ts"}},
		},
		{
			name:             "case-insensitive fs maps same file casing",
			useCaseSensitive: false,
			files: map[string]any{
				"/Repo/App/tsconfig.json": `{"include": ["src/**/*.ts"]}`,
				"/Repo/App/src/Main.ts":   `export const main = 1;`,
			},
			inputs: []string{"/repo/app/src/main.ts", "/REPO/APP/SRC/MAIN.ts"},
			expectedKeys: map[string]string{
				"/repo/app/src/main.ts": anyConfiguredProject,
				"/REPO/APP/SRC/MAIN.ts": anyConfiguredProject,
			},
			sameGroups: [][]string{{"/repo/app/src/main.ts", "/REPO/APP/SRC/MAIN.ts"}},
		},
		{
			name: "prefer tsconfig over jsconfig in same directory",
			files: map[string]any{
				"/repo/tsconfig.json": `{"files": []}`,
				"/repo/jsconfig.json": `{"include": ["src/**/*.ts"]}`,
				"/repo/src/main.ts":   `export const main = 1;`,
			},
			inputs: []string{"/repo/src/main.ts"},
			expectedKeys: map[string]string{
				"/repo/src/main.ts": inferredProjectKey,
			},
		},
		{
			name: "use jsconfig when tsconfig is absent",
			files: map[string]any{
				"/repo/jsconfig.json": `{"include": ["src/**/*.js"]}`,
				"/repo/src/main.js":   `export const main = 1;`,
			},
			inputs: []string{"/repo/src/main.js"},
			expectedKeys: map[string]string{
				"/repo/src/main.js": "/repo/jsconfig.json",
			},
		},
		{
			name: "use ancestor config when nearest does not include file",
			files: map[string]any{
				"/repo/tsconfig.json":              `{"include": ["packages/**/*.ts"]}`,
				"/repo/packages/pkg/tsconfig.json": `{"include": ["src/**/*.ts"]}`,
				"/repo/packages/pkg/src/a.ts":      `export const a = 1;`,
				"/repo/packages/pkg/tests/test.ts": `export const t = 1;`,
			},
			inputs: []string{"/repo/packages/pkg/src/a.ts", "/repo/packages/pkg/tests/test.ts"},
			expectedKeys: map[string]string{
				"/repo/packages/pkg/src/a.ts":      "/repo/packages/pkg/tsconfig.json",
				"/repo/packages/pkg/tests/test.ts": "/repo/tsconfig.json",
			},
		},
		{
			name: "use ancestor config when nearest does not include file 2",
			files: map[string]any{
				"/repo/tsconfig.json":              `{"include": ["packages/**/*.ts"]}`,
				"/repo/packages/pkg/tsconfig.json": `{"include": ["src/**/*.ts"]}`,
				"/repo/packages/pkg/src/a.ts":      `export const a = 1;`,
				"/repo/packages/pkg/tests/test.ts": `export const t = 1;`,
			},
			inputs: []string{"/repo/packages/pkg/tests/test.ts"},
			expectedKeys: map[string]string{
				"/repo/packages/pkg/tests/test.ts": "/repo/tsconfig.json",
			},
		},
		{
			name: "skip composite nearest config when file is outside composite roots",
			files: map[string]any{
				"/repo/tsconfig.json": `{"include": ["packages/**/*.ts"]}`,
				"/repo/packages/pkg/tsconfig.json": `{
					"include": ["src/**/*.ts"],
					"compilerOptions": {"composite": true}
				}`,
				"/repo/packages/pkg/src/a.ts":      `export const a = 1;`,
				"/repo/packages/pkg/tests/test.ts": `export const t = 1;`,
			},
			inputs: []string{"/repo/packages/pkg/tests/test.ts"},
			expectedKeys: map[string]string{
				"/repo/packages/pkg/tests/test.ts": "/repo/tsconfig.json",
			},
		},
		{
			name: "disableSolutionSearching blocks ancestor lookup",
			files: map[string]any{
				"/repo/tsconfig.json": `{"include": ["packages/**/*.ts"]}`,
				"/repo/packages/pkg/tsconfig.json": `{
					"include": ["src/**/*.ts"],
					"compilerOptions": {"disableSolutionSearching": true}
				}`,
				"/repo/packages/pkg/src/a.ts":      `export const a = 1;`,
				"/repo/packages/pkg/tests/test.ts": `export const t = 1;`,
			},
			inputs: []string{"/repo/packages/pkg/tests/test.ts"},
			expectedKeys: map[string]string{
				"/repo/packages/pkg/tests/test.ts": inferredProjectKey,
			},
		},
		{
			name: "reuse configured program when different start configs converge to same owner",
			files: map[string]any{
				"/repo/tsconfig.json":                `{"include": ["packages/**/*.ts"]}`,
				"/repo/packages/p1/tsconfig.json":    `{"include": ["src/**/*.ts"]}`,
				"/repo/packages/p1/src/a.ts":         `export const a = 1;`,
				"/repo/packages/p1/tests/extra-a.ts": `export const extraA = 1;`,
				"/repo/packages/p2/tsconfig.json":    `{"include": ["src/**/*.ts"]}`,
				"/repo/packages/p2/src/b.ts":         `export const b = 1;`,
				"/repo/packages/p2/tests/extra-b.ts": `export const extraB = 1;`,
			},
			inputs: []string{"/repo/packages/p1/tests/extra-a.ts", "/repo/packages/p2/tests/extra-b.ts"},
			expectedKeys: map[string]string{
				"/repo/packages/p1/tests/extra-a.ts": "/repo/tsconfig.json",
				"/repo/packages/p2/tests/extra-b.ts": "/repo/tsconfig.json",
			},
			sameGroups: [][]string{{"/repo/packages/p1/tests/extra-a.ts", "/repo/packages/p2/tests/extra-b.ts"}},
		},
		{
			name: "fall back to inferred when nearest config excludes file",
			files: map[string]any{
				"/repo/tsconfig.json": `{"include": ["src/**/*.ts"]}`,
				"/repo/other/file.ts": `export const x = 1;`,
			},
			inputs: []string{"/repo/other/file.ts"},
			expectedKeys: map[string]string{
				"/repo/other/file.ts": inferredProjectKey,
			},
		},
		{
			name: "use one inferred program when no config exists",
			files: map[string]any{
				"/repo/a.ts": `export const a = 1;`,
				"/repo/b.ts": `export const b = 2;`,
			},
			inputs: []string{"/repo/a.ts", "/repo/b.ts"},
			expectedKeys: map[string]string{
				"/repo/a.ts": inferredProjectKey,
				"/repo/b.ts": inferredProjectKey,
			},
			sameGroups: [][]string{{"/repo/a.ts", "/repo/b.ts"}},
		},
		{
			name: "node_modules cutoff prevents climbing to workspace config",
			files: map[string]any{
				"/repo/tsconfig.json":             `{"include": ["**/*.ts"]}`,
				"/repo/node_modules/pkg/index.ts": `export const pkg = 1;`,
			},
			inputs: []string{"/repo/node_modules/pkg/index.ts"},
			expectedKeys: map[string]string{
				"/repo/node_modules/pkg/index.ts": inferredProjectKey,
			},
		},
		{
			name: "resolve through solution references to nonstandard config",
			files: map[string]any{
				"/repo/tsconfig.json": `{
					"files": [],
					"references": [
						{"path": "./packages/app"},
						{"path": "./shared/tsconfig.lib.json"}
					]
				}`,
				"/repo/packages/app/tsconfig.json": `{"include": ["src/**/*.ts"]}`,
				"/repo/packages/app/src/main.ts":   `export const app = 1;`,
				"/repo/shared/tsconfig.lib.json":   `{"include": ["src/**/*.ts"], "compilerOptions": {"composite": true}}`,
				"/repo/shared/src/lib.ts":          `export const lib = 1;`,
			},
			inputs: []string{"/repo/shared/src/lib.ts"},
			expectedKeys: map[string]string{
				"/repo/shared/src/lib.ts": "/repo/shared/tsconfig.lib.json",
			},
		},
		{
			name: "prefer direct inclusion over source-from-reference fallback",
			files: map[string]any{
				"/repo/tsconfig.json": `{
					"files": [],
					"references": [
						{"path": "./main/tsconfig.main.json"},
						{"path": "./dep/tsconfig.dep.json"}
					]
				}`,
				"/repo/main/tsconfig.main.json": `{
					"compilerOptions": {"composite": true},
					"references": [{"path": "../dep/tsconfig.dep.json"}]
				}`,
				"/repo/main/main.ts": `
					import { fn1 } from '../decls/fns';
					fn1();
				`,
				"/repo/dep/tsconfig.dep.json": `{
					"compilerOptions": {
						"composite": true,
						"declarationDir": "../decls"
					}
				}`,
				"/repo/dep/fns.ts": `export function fn1() {}`,
			},
			inputs: []string{"/repo/main/main.ts", "/repo/dep/fns.ts"},
			expectedKeys: map[string]string{
				"/repo/main/main.ts": "/repo/main/tsconfig.main.json",
				"/repo/dep/fns.ts":   "/repo/dep/tsconfig.dep.json",
			},
			differentPairs: [][2]string{{"/repo/main/main.ts", "/repo/dep/fns.ts"}},
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			programs, _, opts := createPrograms(t, tc.files, tc.inputs, tc.useCaseSensitive)
			assert.Equal(t, len(programs), len(tc.inputs))

			byInput := make(map[string]*compiler.Program, len(tc.inputs))
			for _, input := range tc.inputs {
				result := requireProgramForInput(t, programs, input, opts)
				program := result.Program
				byInput[input] = program

				expectedKey := tc.expectedKeys[input]
				switch expectedKey {
				case inferredProjectKey:
					assert.Equal(t, programKey(program), inferredProjectKey)
				case anyConfiguredProject:
					assert.Assert(t, programKey(program) != inferredProjectKey)
				default:
					assert.Equal(t, programKey(program), expectedKey)
				}
			}

			for _, group := range tc.sameGroups {
				for i := 1; i < len(group); i++ {
					assert.Assert(t, byInput[group[0]] == byInput[group[i]], "%s and %s should share the same program", group[0], group[i])
				}
			}

			for _, pair := range tc.differentPairs {
				assert.Assert(t, byInput[pair[0]] != byInput[pair[1]], "%s and %s should map to different programs", pair[0], pair[1])
			}
		})
	}
}

func TestCreatePrograms_ForcesSourceRedirectWhenConfigDisablesIt(t *testing.T) {
	t.Parallel()

	files := map[string]any{
		"/repo/tsconfig.json": `{
			"files": [],
			"references": [{"path": "./main/tsconfig.main.json"}]
		}`,
		"/repo/main/tsconfig.main.json": `{
			"compilerOptions": {
				"composite": true,
				"disableSourceOfProjectReferenceRedirect": true
			},
			"references": [{"path": "../dependency/tsconfig.dep.json"}]
		}`,
		"/repo/main/main.ts": `
			import { fn1 } from '../decls/fns';
			fn1();
		`,
		"/repo/dependency/tsconfig.dep.json": `{
			"compilerOptions": {
				"composite": true,
				"declarationDir": "../decls"
			}
		}`,
		"/repo/dependency/fns.ts": `export function fn1() {}`,
	}

	inputs := []string{"/repo/main/main.ts"}
	programs, _, opts := createPrograms(t, files, inputs, true)
	result := requireProgramForInput(t, programs, "/repo/main/main.ts", opts)
	program := result.Program

	assert.Equal(t, programKey(program), "/repo/main/tsconfig.main.json")
	assert.Assert(t, result.SourceFile == program.GetSourceFile("/repo/main/main.ts"))
	assert.Assert(t, program.GetSourceFile("/repo/dependency/fns.ts") != nil)
	assert.Assert(t, program.GetSourceFile("/repo/decls/fns.d.ts") == nil)

	depPath := tspath.ToPath("/repo/dependency/fns.ts", opts.CurrentDirectory, opts.FS.UseCaseSensitiveFileNames())
	assert.Assert(t, program.IsSourceFromProjectReference(depPath))
}

func TestCreatePrograms_StampsNodeIDsToMatchEncodedOrder(t *testing.T) {
	t.Parallel()

	files := map[string]any{
		"/repo/tsconfig.json": `{"include": ["src/**/*.ts"]}`,
		"/repo/src/main.ts": `
			/** docs */
			export function foo<T>(arg: number) {
				return arg
			}
		`,
	}

	programs, _, opts := createPrograms(t, files, []string{"/repo/src/main.ts"}, true)
	result := requireProgramForInput(t, programs, "/repo/src/main.ts", opts)
	sourceFile := result.SourceFile

	assert.Equal(t, sourceFile.AsNode().Id, uint32(1))
	assert.Equal(t, sourceFile.AsNode().SourceFileId, sourceFile.Id)

	encoded, _, err := apiencoder.EncodeSourceFile(sourceFile)
	assert.NilError(t, err)

	encodedIndexes := encodedNodeIndexes(encoded)
	visitSourceFileNodes(sourceFile, func(node *ast.Node) {
		assert.Equal(t, node.SourceFileId, sourceFile.Id)
		if node == sourceFile.AsNode() {
			return
		}

		index, ok := encodedIndexes[uintptr(unsafe.Pointer(node))]
		assert.Assert(t, ok, "expected encoded index for %s", node.Kind)
		assert.Equal(t, node.Id, index)
	})
}

func TestCreatePrograms_StampsImportedNonInputFiles(t *testing.T) {
	t.Parallel()

	files := map[string]any{
		"/repo/tsconfig.json": `{"include": ["src/**/*.ts"]}`,
		"/repo/src/main.ts":   `import { foo } from './dep'; export const bar = foo;`,
		"/repo/src/dep.ts":    `export const foo = 1;`,
	}

	programs, _, opts := createPrograms(t, files, []string{"/repo/src/main.ts"}, true)
	result := requireProgramForInput(t, programs, "/repo/src/main.ts", opts)
	dep := result.Program.GetSourceFile("/repo/src/dep.ts")

	assert.Assert(t, dep != nil)
	assert.Equal(t, dep.AsNode().Id, uint32(1))
	assert.Equal(t, dep.AsNode().SourceFileId, dep.Id)

	var foundStampedChild bool
	visitSourceFileNodes(dep, func(node *ast.Node) {
		if node != dep.AsNode() && node.SourceFileId == dep.Id && node.Id != 0 {
			foundStampedChild = true
		}
	})
	assert.Assert(t, foundStampedChild)
}

func createPrograms(t *testing.T, files map[string]any, inputs []string, useCaseSensitive bool) (map[string]ProgramSourceFile, []*compiler.Program, CreateProgramsOptions) {
	t.Helper()

	fs := bundled.WrapFS(vfstest.FromMap(files, useCaseSensitive))
	opts := CreateProgramsOptions{
		FS:                 fs,
		CurrentDirectory:   "/",
		DefaultLibraryPath: bundled.LibPath(),
	}
	programsByInput, programs := CreatePrograms(inputs, opts)
	return programsByInput, programs, opts
}

func requireProgramForInput(t *testing.T, programs map[string]ProgramSourceFile, input string, opts CreateProgramsOptions) ProgramSourceFile {
	t.Helper()

	result, ok := programs[input]
	assert.Assert(t, ok, "expected program for input %s", input)
	assert.Assert(t, result.Program != nil, "expected non-nil program for input %s", input)
	assert.Assert(t, result.SourceFile != nil, "expected non-nil source file for input %s", input)

	normalized := tspath.GetNormalizedAbsolutePath(input, opts.CurrentDirectory)
	path := tspath.ToPath(normalized, opts.CurrentDirectory, opts.FS.UseCaseSensitiveFileNames())
	assert.Assert(t, result.SourceFile == result.Program.GetSourceFileByPath(path), "expected source file for %s to match %s", input, normalized)

	return result
}

func programKey(program *compiler.Program) string {
	commandLine := program.CommandLine()
	if commandLine == nil || commandLine.ConfigFile == nil {
		return inferredProjectKey
	}
	return commandLine.ConfigName()
}

func encodedNodeIndexes(encoded []byte) map[uintptr]uint32 {
	offsetNodes := int(binary.LittleEndian.Uint32(encoded[apiencoder.HeaderOffsetNodes:]))
	nodeCount := (len(encoded) - offsetNodes) / apiencoder.NodeSize
	indexes := make(map[uintptr]uint32, nodeCount)
	for i := 1; i < nodeCount; i++ {
		byteIndex := offsetNodes + i*apiencoder.NodeSize
		pointerLo := binary.LittleEndian.Uint32(encoded[byteIndex+apiencoder.NodeOffsetPointer:])
		pointerHi := binary.LittleEndian.Uint32(encoded[byteIndex+apiencoder.NodeOffsetPointer+4:])
		if pointerLo == 0 && pointerHi == 0 {
			continue
		}
		indexes[uintptr(uint64(pointerHi)<<32|uint64(pointerLo))] = uint32(i)
	}
	return indexes
}

func visitSourceFileNodes(sourceFile *ast.SourceFile, visit func(*ast.Node)) {
	seen := map[*ast.Node]struct{}{}
	var walk func(*ast.Node)
	walk = func(node *ast.Node) {
		if node == nil {
			return
		}
		if _, ok := seen[node]; ok {
			return
		}
		seen[node] = struct{}{}
		visit(node)
		node.ForEachChild(func(child *ast.Node) bool {
			walk(child)
			return false
		})
		for _, jsdoc := range node.JSDoc(sourceFile) {
			walk(jsdoc)
		}
	}
	walk(sourceFile.AsNode())
}
