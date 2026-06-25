package workspace

import (
	"context"
	"encoding/binary"
	"fmt"
	"slices"
	"sync"

	"github.com/auvred/golar/internal/linter"
	"github.com/auvred/golar/internal/typeencoder"
	"github.com/auvred/golar/internal/utils"
	"github.com/microsoft/typescript-go/pkg/api/encoder"
	"github.com/microsoft/typescript-go/pkg/ast"
	"github.com/microsoft/typescript-go/pkg/bundled"
	"github.com/microsoft/typescript-go/pkg/checker"
	"github.com/microsoft/typescript-go/pkg/collections"
	"github.com/microsoft/typescript-go/pkg/compiler"
	"github.com/microsoft/typescript-go/pkg/core"
	"github.com/microsoft/typescript-go/pkg/execute/tsc"
	"github.com/microsoft/typescript-go/pkg/tspath"
	"github.com/microsoft/typescript-go/pkg/vfs/cachedvfs"
	"github.com/microsoft/typescript-go/pkg/vfs/osvfs"
)

type program struct {
	program   *compiler.Program
	checker   *checker.Checker
	encoderMu sync.Mutex
	encoders  []*typeencoder.Encoder
}

const maxThreads = 64

func (p *program) encoderForThread(threadId uint32) *typeencoder.Encoder {
	p.encoderMu.Lock()
	defer p.encoderMu.Unlock()

	if int(threadId) >= len(p.encoders) {
		panic(fmt.Errorf("invalid thread id %d", threadId))
	}

	encoder := p.encoders[threadId]
	if encoder != nil {
		return encoder
	}

	encoder = typeencoder.New()
	p.encoders[threadId] = encoder
	return encoder
}

type Workspace struct {
	// input index -> source file + program
	RequestedFiles []linter.ProgramSourceFile
	// source file id -> source file
	// used only for encoding purposes
	FilesById []*ast.SourceFile
	// program id -> program
	Programs          []*program
	reportBaseDir     string
	pathCaseSensitive bool
	reportMu          sync.Mutex
}

func New(cwd string, filenames []string) *Workspace {
	fs := bundled.WrapFS(cachedvfs.From(osvfs.FS()))
	results, sourcePrograms := linter.CreatePrograms(filenames, linter.CreateProgramsOptions{
		FS:                 fs,
		CurrentDirectory:   cwd,
		DefaultLibraryPath: bundled.LibPath(),
	})

	programs := make([]*program, len(sourcePrograms))
	requestedFiles := make([]linter.ProgramSourceFile, len(filenames))
	var maxSourceFileID uint32

	for i, sourceProgram := range sourcePrograms {
		// TODO: think about checker disposal
		checker, _ := sourceProgram.GetTypeChecker(context.Background())
		programs[i] = &program{
			program:  sourceProgram,
			checker:  checker,
			encoders: make([]*typeencoder.Encoder, maxThreads),
		}
		for _, sourceFile := range sourceProgram.GetSourceFiles() {
			maxSourceFileID = max(maxSourceFileID, sourceFile.Id)
		}
	}

	filesById := make([]*ast.SourceFile, maxSourceFileID+1)

	for _, sourceProgram := range sourcePrograms {
		for _, sourceFile := range sourceProgram.GetSourceFiles() {
			if filesById[sourceFile.Id] != nil {
				continue
			}
			filesById[sourceFile.Id] = sourceFile
		}
	}

	for i, filename := range filenames {
		requestedFiles[i] = results[filename]
	}

	workspace := &Workspace{
		RequestedFiles:    requestedFiles,
		FilesById:         filesById,
		Programs:          programs,
		reportBaseDir:     tspath.NormalizePath(cwd),
		pathCaseSensitive: fs.UseCaseSensitiveFileNames(),
	}

	return workspace
}

func (w *Workspace) ReadRequestedFileAt(index uint32) (linter.ProgramSourceFile, []byte) {
	file := w.RequestedFiles[index]

	encodedFile, _, err := encoder.EncodeSourceFile(file.SourceFile)
	if err != nil {
		panic(err)
	}
	return file, encodedFile
}

func (w *Workspace) ReadFileById(id uint32) []byte {
	encodedFile, _, err := encoder.EncodeSourceFile(w.FilesById[id])
	if err != nil {
		panic(err)
	}

	return encodedFile
}

func (w *Workspace) GetTypeAtLocation(buf []byte, threadId uint32, programId uint32, node *ast.Node) {
	program := w.Programs[programId]
	t := program.checker.GetTypeAtLocation(node)

	program.encoderForThread(threadId).EncodeType(buf[:0], t)
}

func (w *Workspace) TypeCheck(buf []byte) int {
	ctx := context.Background()
	offset := 0
	filesCount := binary.LittleEndian.Uint32(buf[offset:])
	offset += 4

	programOrder := make([]uint32, 0, filesCount)
	requestedFilesByProgram := make(map[uint32][]*ast.SourceFile, filesCount)
	var seenPrograms collections.Set[uint32]

	for range filesCount {
		fileIdx := binary.LittleEndian.Uint32(buf[offset:])
		offset += 4

		if int(fileIdx) >= len(w.RequestedFiles) {
			panic(fmt.Errorf("invalid typecheck file id %d", fileIdx))
		}

		file := w.RequestedFiles[fileIdx]

		if seenPrograms.AddIfAbsent(file.ProgramId) {
			programOrder = append(programOrder, file.ProgramId)
		}
		requestedFilesByProgram[file.ProgramId] = append(requestedFilesByProgram[file.ProgramId], file.SourceFile)
	}

	allDiagnostics := make([]*ast.Diagnostic, 0)
	for _, programID := range programOrder {
		program := w.Programs[programID].program
		allDiagnostics = append(allDiagnostics, collectTypeCheckDiagnostics(ctx, program, requestedFilesByProgram[programID])...)
	}

	allDiagnostics = compiler.SortAndDeduplicateDiagnostics(allDiagnostics)

	sys := utils.NewOsSystem()
	reportProgram := w.Programs[programOrder[0]].program
	reportDiagnostic := tsc.CreateDiagnosticReporter(sys, sys.Writer(), reportProgram.CommandLine().Locale(), reportProgram.Options())
	reportErrorSummary := tsc.CreateReportErrorSummary(sys, reportProgram.CommandLine().Locale(), reportProgram.Options())
	for _, diagnostic := range allDiagnostics {
		reportDiagnostic(diagnostic)
	}
	reportErrorSummary(allDiagnostics)

	if len(allDiagnostics) > 0 {
		return 1
	}
	return 0
}

// Mirrored from GetDiagnosticsOfAnyProgram
func collectTypeCheckDiagnostics(ctx context.Context, program *compiler.Program, requestedFiles []*ast.SourceFile) []*ast.Diagnostic {
	allDiagnostics := slices.Clip(program.GetConfigFileParsingDiagnostics())
	configFileParsingDiagnosticsLength := len(allDiagnostics)

	allDiagnostics = append(allDiagnostics, program.GetSyntacticDiagnostics(ctx, nil)...)
	allDiagnostics = append(allDiagnostics, program.GetProgramDiagnostics()...)

	if len(allDiagnostics) == configFileParsingDiagnosticsLength {
		program.GetBindDiagnostics(ctx, nil)

		if program.Options().ListFilesOnly.IsFalseOrUnknown() {
			allDiagnostics = append(allDiagnostics, program.GetGlobalDiagnostics(ctx)...)

			if len(allDiagnostics) == configFileParsingDiagnosticsLength {
				allDiagnostics = append(allDiagnostics, collectRequestedFileDiagnostics(program, requestedFiles, func(file *ast.SourceFile) []*ast.Diagnostic {
					return program.GetSemanticDiagnostics(ctx, file)
				})...)
			}

			if program.Options().GetEmitDeclarations() && len(allDiagnostics) == configFileParsingDiagnosticsLength {
				allDiagnostics = append(allDiagnostics, collectRequestedFileDiagnostics(program, requestedFiles, func(file *ast.SourceFile) []*ast.Diagnostic {
					return program.GetDeclarationDiagnostics(ctx, file)
				})...)
			}
		}
	}

	return allDiagnostics
}

func collectRequestedFileDiagnostics(program *compiler.Program, requestedFiles []*ast.SourceFile, collect func(file *ast.SourceFile) []*ast.Diagnostic) []*ast.Diagnostic {
	if len(requestedFiles) == 0 {
		return nil
	}

	diagnostics := make([][]*ast.Diagnostic, len(requestedFiles))
	wg := core.NewWorkGroup(program.SingleThreaded())
	for i, file := range requestedFiles {
		wg.Queue(func() {
			diagnostics[i] = collect(file)
		})
	}
	wg.RunAndWait()

	return slices.Concat(diagnostics...)
}
