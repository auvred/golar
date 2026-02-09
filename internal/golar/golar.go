package golar

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/auvred/golar/internal/mapping"
	"github.com/auvred/golar/internal/pluginhost"
	"github.com/auvred/golar/internal/utils"
	"github.com/auvred/golar/internal/vue/codegen"
	"github.com/auvred/golar/internal/vue/parser"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/compiler"
	"github.com/microsoft/typescript-go/shim/core"
	"github.com/microsoft/typescript-go/shim/diagnostics"
	"github.com/microsoft/typescript-go/shim/golarext"
	"github.com/microsoft/typescript-go/shim/parser"
	"github.com/microsoft/typescript-go/shim/scanner"
	"github.com/microsoft/typescript-go/shim/sourcemap"
	"github.com/microsoft/typescript-go/shim/tspath"
	"github.com/microsoft/typescript-go/shim/vfs"
)

var unused_directive = &diagnostics.Message{}

func init() {
	diagnostics.Message_Set_code(unused_directive, 1_000_000)
	diagnostics.Message_Set_category(unused_directive, diagnostics.CategoryError)
	diagnostics.Message_Set_key(unused_directive, "Unused_directive")
	diagnostics.Message_Set_text(unused_directive, "Unused directive.")

	// Always register .vue as a supported extension so the file loader
	// includes .vue files in the program. Without this, the filesparser
	// rejects .vue files when AllowNonTsExtensions is not set.
	tspath.RegisterSupportedExtension(".vue")
}

type compilerHostProxy struct {
	compiler.CompilerHost
}

type languageData struct {
	sourceText            string
	sourceMap             *mapping.SourceMap
	ignoreDirectives      []mapping.IgnoreDirectiveMapping
	expectErrorDirectives []mapping.ExpectErrorDirectiveMapping
}

func (h *compilerHostProxy) GetSourceFile(opts ast.SourceFileParseOptions) *ast.SourceFile {
	if strings.HasSuffix(opts.FileName, ".vue") || strings.HasSuffix(opts.FileName, ".svelte") || strings.HasSuffix(opts.FileName, ".astro") {
		sourceText, ok := h.CompilerHost.FS().ReadFile(opts.FileName)
		if !ok {
			return nil
		}
		return parseFile(h.FS(), opts, sourceText, core.GetScriptKindFromFileName(opts.FileName))
	}
	return h.CompilerHost.GetSourceFile(opts)
}

func wrapCompilerHost(host compiler.CompilerHost) compiler.CompilerHost {
	return &compilerHostProxy{host}
}

var vuePlugin *pluginhost.Plugin
var sveltePlugin *pluginhost.Plugin
var astroPlugin *pluginhost.Plugin

func init() {
	var err error
	pluginNames, ok := os.LookupEnv("GOLAR_PLUGIN")
	if !ok {
		return
	}

	for pluginName := range strings.SplitSeq(pluginNames, ",") {
		switch pluginName {
		case "vue":
			vuePlugin, err = pluginhost.NewPlugin([]string{"node", "/home/auvred/dev/personal/github/auvred/golar/packages/vue/src/index.ts"})
			if err != nil {
				panic(err)
			}
			for _, ext := range vuePlugin.ExtraExtensions {
				tspath.RegisterSupportedExtension(ext)
			}
		case "svelte":
			sveltePlugin, err = pluginhost.NewPlugin([]string{"node", "/home/auvred/dev/personal/github/auvred/golar/packages/svelte/src/index.ts"})
			if err != nil {
				panic(err)
			}
			for _, ext := range sveltePlugin.ExtraExtensions {
				tspath.RegisterSupportedExtension(ext)
			}
		case "astro":
			astroPlugin, err = pluginhost.NewPlugin([]string{"/home/auvred/dev/personal/github/auvred/golar/packages/astro/astro"})
			if err != nil {
				panic(err)
			}
			for _, ext := range astroPlugin.ExtraExtensions {
				tspath.RegisterSupportedExtension(ext)
			}
		}
	}
}

func sourceMapToMapping(inputMappings string, sourceText string, serviceText string) []mapping.Mapping {
	dec := sourcemap.DecodeMappings(inputMappings)
	serviceLineMap := core.ComputeECMALineStarts(serviceText)
	sourceLineMap := core.ComputeECMALineStarts(sourceText)
	mappings := make([]mapping.Mapping, 0)

	type currentMapping struct {
		genOffset    uint32
		sourceOffset uint32
	}

	var current *currentMapping

	for decoded, done := dec.Next(); !done; decoded, done = dec.Next() {
		if decoded == nil {
			continue
		}
		genOffset := uint32(scanner.ComputePositionOfLineAndCharacterEx(
			serviceLineMap,
			decoded.GeneratedLine,
			decoded.GeneratedCharacter,
			&serviceText,
			false,
		))
		if current != nil {
			length := genOffset - current.genOffset
			if length > 0 {
				sourceEnd := min(current.sourceOffset+length, uint32(len(sourceText)))
				genEnd := min(current.genOffset+length, uint32(len(serviceText)))
				sourceChunk := sourceText[current.sourceOffset:sourceEnd]
				genChunk := serviceText[current.genOffset:genEnd]
				if sourceChunk != genChunk {
					length = 0
					maxLen := min(len(sourceChunk), len(genChunk))
					for i := range maxLen {
						if sourceChunk[i] == genChunk[i] {
							length = uint32(i) + 1
						} else {
							break
						}
					}
				}
			}
			if length > 0 {
				if len(mappings) > 0 {
					last := &mappings[len(mappings)-1]
					if last.ServiceOffset+last.SourceLength == current.genOffset &&
						last.SourceOffset+last.SourceLength == current.sourceOffset {
						last.SourceLength += length
					} else {
						mappings = append(mappings, mapping.Mapping{
							SourceOffset:  current.sourceOffset,
							ServiceOffset: current.genOffset,
							SourceLength:  length,
						})
					}
				} else {
					mappings = append(mappings, mapping.Mapping{
						SourceOffset:  current.sourceOffset,
						ServiceOffset: current.genOffset,
						SourceLength:  length,
					})
				}
			}
			current = nil
		}
		if decoded.IsSourceMapping() {
			if decoded.SourceIndex != 0 {
				continue
			}
			sourceOffset := uint32(scanner.ComputePositionOfLineAndCharacterEx(
				sourceLineMap,
				decoded.SourceLine,
				decoded.SourceCharacter,
				&sourceText,
				true,
			))
			current = &currentMapping{
				genOffset:    genOffset,
				sourceOffset: sourceOffset,
			}
		}
	}

	if err := dec.Error(); err != nil {
		panic(err)
	}

	return mappings
}

func parseFile(fs vfs.FS, opts ast.SourceFileParseOptions, sourceText string, scriptKind core.ScriptKind) *ast.SourceFile {
	// .vue files from disk get ScriptKindUnknown; default to TS
	if scriptKind == core.ScriptKindUnknown && (strings.HasSuffix(opts.FileName, ".vue") || strings.HasSuffix(opts.FileName, ".svelte") || strings.HasSuffix(opts.FileName, ".astro")) {
		scriptKind = core.ScriptKindTS
	}

	var plugin *pluginhost.Plugin

	if strings.HasSuffix(opts.FileName, ".vue") {
		plugin = vuePlugin
	} else if strings.HasSuffix(opts.FileName, ".svelte") {
		plugin = sveltePlugin
	} else if strings.HasSuffix(opts.FileName, ".astro") {
		plugin = astroPlugin
	}

	if plugin != nil {
		resp := <-plugin.CreateServiceCode(opts.FileName, sourceText)

		file := parser.ParseSourceFile(opts, resp.ServiceText, resp.ScriptKind)
		// TODO: figure out better way; .astro files have virtual: dev-only imports
		if strings.Contains(opts.FileName, "/node_modules/") {
			file.IsDeclarationFile = true
		}
		file.WrapDiagnostics = func (diags []*ast.Diagnostic) []*ast.Diagnostic {
			newFile := ast.SourceFile{
			}
			newFile.GolarLanguageData = file.GolarLanguageData
			newFile.SetText(sourceText)
			newFile.SetParseOptions(file.ParseOptions())
			return wrapDiagnostics(&newFile, diags, false)
		}
		file.WrapSemanticDiagnostics = func (diags []*ast.Diagnostic) []*ast.Diagnostic {
			// TODO: this is hack
			newFile := ast.SourceFile{}
			newFile.GolarLanguageData = file.GolarLanguageData
			newFile.SetText(sourceText)
			newFile.SetParseOptions(file.ParseOptions())
			return wrapDiagnostics(&newFile, diags, true)
		}
		langData := languageData{
			sourceText:            sourceText,
		}
		if resp.SourceMap != "" {
			langData.sourceMap = mapping.NewSourceMap(sourceMapToMapping(resp.SourceMap, sourceText, resp.ServiceText))
		} else {
			langData.sourceMap = mapping.NewSourceMap(resp.Mappings)
			langData.ignoreDirectives = resp.IgnoreMappings
		}
		file.GolarLanguageData = langData
		// diags := file.Diagnostics()
		// for i, diag := range diags {
		// 	diags[i] = adjustDiagnostic(file, diag)
		// }

		return file
	}
	if !strings.HasSuffix(opts.FileName, ".vue") {
		return parser.ParseSourceFile(opts, sourceText, scriptKind)
	}
	vueAst, parsingErrors := vue_parser.Parse(sourceText)
	var serviceText string
	var mappings []mapping.Mapping
	var ignoreDirectives []mapping.IgnoreDirectiveMapping
	var expectErrorDirectives []mapping.ExpectErrorDirectiveMapping
	var fileDiagnostics []*ast.Diagnostic
	if len(parsingErrors) > 0 {
		// TODO: error recovery?
		fileDiagnostics = make([]*ast.Diagnostic, len(parsingErrors))
		for i, err := range parsingErrors {
			// TODO: statically define parsing errors as diagnostics
			msg := &diagnostics.Message{}
			diagnostics.Message_Set_code(msg, 1_000_999)
			diagnostics.Message_Set_category(msg, diagnostics.CategoryError)
			diagnostics.Message_Set_key(msg, diagnostics.Key(err.Message))
			diagnostics.Message_Set_text(msg, err.Message)
			fileDiagnostics[i] = ast.NewDiagnostic(nil, core.NewTextRange(err.Pos, err.Pos), msg)
		}
	} else if vueVersion, ok := resolveVueVersion(fs, opts.FileName); ok {
		options := vue_codegen.VueOptions{
			Version: vueVersion,
		}
		serviceText, mappings, ignoreDirectives, expectErrorDirectives, fileDiagnostics = vue_codegen.Codegen(sourceText, vueAst, options)
	} else {
		msg := &diagnostics.Message{}
		diagnostics.Message_Set_code(msg, 1_000_999)
		diagnostics.Message_Set_category(msg, diagnostics.CategoryError)
		diagnostics.Message_Set_key(msg, "unable_to_resolve_vue_version")
		diagnostics.Message_Set_text(msg, "Unable to resolve Vue version")
		fileDiagnostics = []*ast.Diagnostic{ast.NewDiagnostic(nil, core.NewTextRange(0, 0), msg)}
	}
	file := parser.ParseSourceFile(opts, serviceText, scriptKind)
		file.WrapDiagnostics = func (diags []*ast.Diagnostic) []*ast.Diagnostic {
			newFile := ast.SourceFile{
			}
			newFile.GolarLanguageData = file.GolarLanguageData
			newFile.SetText(sourceText)
			newFile.SetParseOptions(file.ParseOptions())
			return wrapDiagnostics(&newFile, diags, false)
		}
		file.WrapSemanticDiagnostics = func (diags []*ast.Diagnostic) []*ast.Diagnostic {
			// TODO: this is hack
			newFile := ast.SourceFile{}
			newFile.GolarLanguageData = file.GolarLanguageData
			newFile.SetText(sourceText)
			newFile.SetParseOptions(file.ParseOptions())
			return wrapDiagnostics(&newFile, diags, true)
		}
		diags := file.Diagnostics()
		for i, diag := range diags {
			diags[i] = adjustDiagnostic(file, diag)
		}
	file.SetDiagnostics(append(file.Diagnostics(), fileDiagnostics...))
	file.GolarLanguageData = languageData{
		sourceText:            sourceText,
		sourceMap:             mapping.NewSourceMap(mappings),
		ignoreDirectives:      ignoreDirectives,
		expectErrorDirectives: expectErrorDirectives,
	}

	return file
}

func resolveVueVersion(fs vfs.FS, fileName string) (vue_codegen.VueVersion, bool) {
	dir := tspath.GetDirectoryPath(fileName)
	for {
		pkgPath := tspath.CombinePaths(dir, "node_modules", "vue", "package.json")
		if fs != nil {
			contents, ok := fs.ReadFile(pkgPath)
			if ok {
				var pkg struct {
					Version string `json:"version"`
				}
				if json.Unmarshal([]byte(contents), &pkg) == nil {
					return parseVueVersion(pkg.Version)
				}
				break
			}
		}
		parent := tspath.GetDirectoryPath(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return 0, false
}

func parseVueVersion(version string) (vue_codegen.VueVersion, bool) {
	version = strings.TrimSpace(version)
	if version == "" {
		return 0, false
	}
	if version[0] == 'v' || version[0] == 'V' {
		version = version[1:]
	}
	parts := [3]int{}
	partIdx := 0
	digits := 0
	for i := 0; i < len(version) && partIdx < len(parts); i++ {
		ch := version[i]
		if ch >= '0' && ch <= '9' {
			parts[partIdx] = parts[partIdx]*10 + int(ch-'0')
			digits++
			continue
		}
		if ch == '.' {
			if digits == 0 {
				return 0, false
			}
			partIdx++
			digits = 0
			continue
		}
		break
	}
	if partIdx == 0 && digits == 0 {
		return 0, false
	}
	return vue_codegen.NewVueVersionFromSemver(parts[0], parts[1], parts[2]), true
}

func adjustDiagnostic(file *ast.SourceFile, diagnostic *ast.Diagnostic) *ast.Diagnostic {
	diagnostic.SetFile(file)
	for _, s := range diagnostic.MessageChain() {
		s.SetFile(file)
	}
	for _, s := range diagnostic.RelatedInformation() {
		s.SetFile(file)
	}
	if file.GolarLanguageData == nil || diagnostic.Code() >= 1_000_000 {
		return diagnostic
	}
	langData := file.GolarLanguageData.(languageData)
	for _, sourceRange := range langData.sourceMap.ToSourceRange(uint32(diagnostic.Pos()), uint32(diagnostic.End()), true) {
		diagnostic.SetLocation(core.NewTextRange(int(sourceRange.MappedStart), int(sourceRange.MappedEnd)))
		return diagnostic
	}

	diagnostic.SetLocation(core.NewTextRange(0, 0))

	return diagnostic
}

func wrapDiagnostics(file *ast.SourceFile, diagnostics []*ast.Diagnostic, collectUnused bool) []*ast.Diagnostic {
	res := []*ast.Diagnostic{}
	if file.GolarLanguageData == nil {
		return nil
	}
	langData := file.GolarLanguageData.(languageData)
	directiveMap := mapping.NewDirectiveMap(langData.ignoreDirectives, langData.expectErrorDirectives)
	for _, diag := range diagnostics {
		if directiveMap.IsServiceRangeIgnored(diag.Loc()) {
			continue
		}
		res = append(res, adjustDiagnostic(file, diag))
	}
	if !collectUnused {
		return res
	}
	for _, e := range directiveMap.CollectUnused() {
		res = append(res, ast.NewDiagnostic(file, core.NewTextRange(int(e.SourceOffset), int(e.SourceOffset+e.SourceLength)), unused_directive))
	}
	return res
}

// TODO: for hover and other LS methods we should analyze multiple mappings
// instead of returning the first mapping
func positionToService(file *ast.SourceFile, pos int) int {
	if file.GolarLanguageData == nil {
		return pos
	}

	langData := file.GolarLanguageData.(languageData)
	for _, serviceLoc := range langData.sourceMap.ToServiceLocation(uint32(pos)) {
		return int(serviceLoc.Offset)
	}
	return pos
}

var GolarExtCallbacks = &golarext.GolarCallbacks{
	// WrapDiagnostics: wrapDiagnostics,
	PositionToService:       positionToService,
	WrapCompilerHost:        wrapCompilerHost,
	ParseSourceFile:         parseFile,
}

func WrapFS(fs vfs.FS) vfs.FS {
	return utils.NewOverlayVFS(fs, map[string]string{
		vue_codegen.GlobalTypesPath: vue_codegen.GlobalTypes,
	})
}

func WrapFourslashFS(globalOptions map[string]string, fs vfs.FS) vfs.FS {
	overlay := map[string]string{
		vue_codegen.GlobalTypesPath: vue_codegen.GlobalTypes,
	}
	if extraFiles := globalOptions["golarextrafiles"]; extraFiles != "" {
		for pair := range strings.SplitSeq(extraFiles, "\x1f") {
			if pair == "" {
				continue
			}
			parsedPair := strings.Split(pair, "\x1e")
			realPath := parsedPair[0]
			virtualPath := parsedPair[1]
			bytes, err := os.ReadFile(realPath)
			if err != nil {
				panic(fmt.Sprintf("error reading %v: %v", realPath, err))
			}
			overlay[virtualPath] = string(bytes)
		}
	}
	return utils.NewOverlayVFS(fs, overlay)
}
