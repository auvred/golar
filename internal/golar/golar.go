package golar

import (
	"path/filepath"
	"slices"

	"github.com/auvred/golar/internal/mapping"
	"github.com/auvred/golar/internal/tscodegenplugin"
	"github.com/auvred/golar/internal/utils"
	"github.com/zeebo/xxh3"

	"github.com/microsoft/typescript-go/pkg/ast"
	"github.com/microsoft/typescript-go/pkg/collections"
	"github.com/microsoft/typescript-go/pkg/compiler"
	"github.com/microsoft/typescript-go/pkg/core"
	"github.com/microsoft/typescript-go/pkg/diagnostics"
	"github.com/microsoft/typescript-go/pkg/parser"
	"github.com/microsoft/typescript-go/pkg/tspath"
)

var pluginErrorDiagnostic = diagnostics.NewMessage(1_000_000, diagnostics.CategoryError, "plugin_error_diagnostic", "{0}")

type compilerHostProxy struct {
	compiler.CompilerHost
	configName      string
	sourceFileCache *collections.SyncMap[xxh3.Uint128, *ast.SourceFile]
}

type languageData struct {
	sourceText            string
	sourceMap             *mapping.SourceMap
	ignoreDirectives      []mapping.IgnoreDirectiveMapping
	expectErrorDirectives []mapping.ExpectErrorDirectiveMapping
}

func bool2byte(x bool) byte {
	var res byte
	if x {
		res = 1
	}
	return res
}

func (h *compilerHostProxy) GetSourceFile(opts ast.SourceFileParseOptions) *ast.SourceFile {
	// FS is cached in single run
	sourceText, ok := h.CompilerHost.FS().ReadFile(opts.FileName)
	if !ok {
		return nil
	}

	var hash xxh3.Hasher128
	hash.WriteString(opts.FileName)
	hash.WriteString(string(opts.Path))
	hash.Write([]byte{bool2byte(opts.ExternalModuleIndicatorOptions.JSX)<<1 | bool2byte(opts.ExternalModuleIndicatorOptions.Force)})
	hash.WriteString(h.configName)

	key := hash.Sum128()

	if h.sourceFileCache != nil {
		if cached, ok := h.sourceFileCache.Load(key); ok {
			return cached
		}
	}

	sourceFile := h.parseFile(opts, sourceText, core.GetScriptKindFromFileName(opts.FileName))
	if sourceFile == nil {
		return h.CompilerHost.GetSourceFile(opts)
	}
	if h.sourceFileCache != nil {
		sourceFile, _ = h.sourceFileCache.LoadOrStore(key, sourceFile)
	}
	return sourceFile
}

func NewCompilerHost(base compiler.CompilerHost, configName string, sourceFileCache *collections.SyncMap[xxh3.Uint128, *ast.SourceFile]) compiler.CompilerHost {
	return &compilerHostProxy{
		CompilerHost:    base,
		configName:      configName,
		sourceFileCache: sourceFileCache,
	}
}

var pluginByExtension = map[string]tscodegenplugin.Plugin{}

func RegisterCodegenPlugin(plugin tscodegenplugin.Plugin) {
	for _, ext := range plugin.Extensions() {
		tspath.RegisterSupportedExtension(ext.Extension)
		pluginByExtension[ext.Extension] = plugin
		if ext.AllowExtensionlessImports {
			tspath.RegisterSupportedExtensionless(ext.Extension)
		}
		if ext.StripFromDeclarationFileName {
			tspath.RegisterExtensionToRemove(ext.Extension)
		}
	}
}

func (h *compilerHostProxy) parseFile(opts ast.SourceFileParseOptions, sourceText string, scriptKind core.ScriptKind) *ast.SourceFile {
	ext := filepath.Ext(opts.FileName)
	if plugin, ok := pluginByExtension[ext]; ok {
		resp := plugin.CreateServiceCode(tscodegenplugin.CreateServiceCodeRequest{
			Cwd:            h.GetCurrentDirectory(),
			ConfigFileName: h.configName,
			FileName:       opts.FileName,
			SourceText:     sourceText,
		})
		if resp.Errors != nil {
			file := ast.NewNode(ast.KindSourceFile, &ast.SourceFile{}).AsSourceFile()
			file.Statements = &ast.NodeList{Nodes: []*ast.Node{}}
			file.SetText(sourceText)
			file.SetParseOptions(opts)
			diags := make([]*ast.Diagnostic, len(resp.Errors))
			for i, err := range resp.Errors {
				diags[i] = ast.NewDiagnostic(file, err.Loc, pluginErrorDiagnostic, err.Message)
			}
			file.SetDiagnostics(diags)
			return file
		}

		file := parser.ParseSourceFile(opts, resp.ServiceText, resp.ScriptKind)
		file.IsDeclarationFile = resp.DeclarationFile
		file.WrapDiagnostics = func(diags []*ast.Diagnostic) []*ast.Diagnostic {
			newFile := ReportingSourceFile(file)
			return wrapDiagnostics(newFile, diags, false, resp.IgnoreNotMappedDiagnostics)
		}
		file.WrapSemanticDiagnostics = func(diags []*ast.Diagnostic) []*ast.Diagnostic {
			newFile := ReportingSourceFile(file)
			return wrapDiagnostics(newFile, diags, true, resp.IgnoreNotMappedDiagnostics)
		}
		langData := languageData{
			sourceText: sourceText,
		}
		langData.sourceMap = mapping.NewSourceMap(resp.Mappings)
		langData.ignoreDirectives = resp.IgnoreMappings
		langData.expectErrorDirectives = resp.ExpectErrorMappings
		file.GolarLanguageData = langData

		return file
	}

	return nil
}

func ServiceToSource(file *ast.SourceFile, loc core.TextRange) (core.TextRange, bool) {
	if file.GolarLanguageData == nil {
		return loc, true
	}
	langData := file.GolarLanguageData.(languageData)
	for _, sourceRange := range langData.sourceMap.ToSourceRange(uint32(loc.Pos()), uint32(loc.End()), true) {
		return core.NewTextRange(int(sourceRange.MappedStart), int(sourceRange.MappedEnd)), true
	}
	return loc, false
}

func ReportingSourceFile(file *ast.SourceFile) *ast.SourceFile {
	if file.GolarLanguageData == nil {
		return file
	}

	langData := file.GolarLanguageData.(languageData)
	newFile := ast.NewNode(ast.KindSourceFile, &ast.SourceFile{}).AsSourceFile()
	newFile.Statements = &ast.NodeList{Nodes: []*ast.Node{}}
	newFile.GolarLanguageData = langData
	newFile.SetText(langData.sourceText)
	newFile.SetParseOptions(file.ParseOptions())

	return newFile
}

func adjustDiagnostic(file *ast.SourceFile, diagnostic *ast.Diagnostic, dropUnmatched bool) *ast.Diagnostic {
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
	MessageChain:
		for _, d := range diagnostic.MessageChain() {
			for _, sourceRange := range langData.sourceMap.ToSourceRange(uint32(d.Pos()), uint32(d.End()), true) {
				d.SetLocation(core.NewTextRange(int(sourceRange.MappedStart), int(sourceRange.MappedEnd)))
				continue MessageChain
			}
			d.SetLocation(core.NewTextRange(0, 0))
		}
	RelatedInformation:
		for _, d := range diagnostic.RelatedInformation() {
			for _, sourceRange := range langData.sourceMap.ToSourceRange(uint32(d.Pos()), uint32(d.End()), true) {
				d.SetLocation(core.NewTextRange(int(sourceRange.MappedStart), int(sourceRange.MappedEnd)))
				continue RelatedInformation
			}
			d.SetLocation(core.NewTextRange(0, 0))
		}
		return diagnostic
	}

	if dropUnmatched {
		return nil
	}

	diagnostic.SetLocation(core.NewTextRange(0, 0))

	return diagnostic
}

func wrapDiagnostics(file *ast.SourceFile, diags []*ast.Diagnostic, collectUnused bool, dropUnmatched bool) []*ast.Diagnostic {
	type expectErrorMarker struct {
		ownerRange   core.TextRange
		serviceRange core.TextRange
	}

	res := []*ast.Diagnostic{}
	if file.GolarLanguageData == nil {
		return nil
	}
	langData := file.GolarLanguageData.(languageData)
	directiveMap := mapping.NewDirectiveMap(langData.ignoreDirectives, langData.expectErrorDirectives)
	markers := []expectErrorMarker{}

Diagnostics:
	for _, diag := range diags {
		if diag.Code() == 2578 {
			markerRanges := langData.sourceMap.ToSourceRangeFiltered(
				uint32(diag.Pos()),
				uint32(diag.End()),
				true,
				func(m *mapping.Mapping) bool {
					return slices.Contains(m.SuppressedDiagnostics, uint32(2578))
				},
			)
			for _, sourceRange := range markerRanges {
				markers = append(markers, expectErrorMarker{
					ownerRange:   core.NewTextRange(int(sourceRange.MappedStart), int(sourceRange.MappedEnd)),
					serviceRange: diag.Loc(),
				})
				continue Diagnostics
			}
		}
		if langData.sourceMap.AnySourceRangeMatch(
			uint32(diag.Pos()),
			uint32(diag.End()),
			true,
			func(m *mapping.Mapping) bool {
				return slices.Contains(m.SuppressedDiagnostics, uint32(diag.Code()))
			},
		) {
			continue
		}
		if directiveMap.IsServiceRangeIgnored(diag.Loc()) {
			continue
		}
		adjusted := adjustDiagnostic(file, diag, dropUnmatched)
		if adjusted != nil {
			res = append(res, adjusted)
		}
	}
	slices.SortFunc(markers, func(a, b expectErrorMarker) int {
		if diff := a.serviceRange.Pos() - b.serviceRange.Pos(); diff != 0 {
			return diff
		}
		return a.serviceRange.End() - b.serviceRange.End()
	})
	for _, marker := range markers {
		directiveMap.ProcessExpectErrorMarker(marker.ownerRange, marker.serviceRange)
	}
	if !collectUnused {
		return res
	}
	for _, loc := range directiveMap.CollectUnused() {
		res = append(res, ast.NewDiagnostic(file, loc, diagnostics.Unused_ts_expect_error_directive))
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

func init() {
	compiler.GolarExt.WrapCompilerHost = func(base compiler.CompilerHost, configName string) compiler.CompilerHost {
		if _, ok := base.(*utils.IndexedCompilerHost); ok {
			return base
		}
		return utils.NewIndexedCompilerHost(NewCompilerHost(base, configName, nil))
	}
	compiler.GolarExt.ReportingSourceFile = ReportingSourceFile
}
