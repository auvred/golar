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
	"github.com/microsoft/TypeScript/tsc/pkg/api/encoder"
	"github.com/microsoft/TypeScript/tsc/pkg/ast"
	"github.com/microsoft/TypeScript/tsc/pkg/bundled"
	"github.com/microsoft/TypeScript/tsc/pkg/checker"
	"github.com/microsoft/TypeScript/tsc/pkg/collections"
	"github.com/microsoft/TypeScript/tsc/pkg/compiler"
	"github.com/microsoft/TypeScript/tsc/pkg/core"
	"github.com/microsoft/TypeScript/tsc/pkg/execute/tsc"
	"github.com/microsoft/TypeScript/tsc/pkg/tspath"
	"github.com/microsoft/TypeScript/tsc/pkg/vfs/cachedvfs"
	"github.com/microsoft/TypeScript/tsc/pkg/vfs/osvfs"
)

type program struct {
	program  *compiler.Program
	threadMu sync.Mutex
	threads  []*programThread
}

// programThread is what one JS thread uses to query a program. A checker is
// not safe for concurrent use, and the JS side caches decoded types by the
// checker's type ids, so every JS thread gets a checker of its own and keeps
// it for the lifetime of the workspace.
type programThread struct {
	checker *checker.Checker
	encoder *typeencoder.Encoder
}

const maxThreads = 64

func (p *program) forThread(threadId uint32) *programThread {
	p.threadMu.Lock()
	defer p.threadMu.Unlock()

	if int(threadId) >= len(p.threads) {
		panic(fmt.Errorf("invalid thread id %d", threadId))
	}

	thread := p.threads[threadId]
	if thread != nil {
		return thread
	}

	c, _ := checker.NewChecker(p.program, nil)
	thread = &programThread{
		checker: c,
		encoder: typeencoder.New(),
	}
	p.threads[threadId] = thread
	return thread
}

type Workspace struct {
	// input index -> source file + program
	RequestedFiles []linter.ProgramSourceFile
	// source file id -> source file
	// used only for encoding purposes
	FilesById []*ast.SourceFile
	// program id -> program
	Programs            []*program
	reportBaseDir       string
	pathCaseSensitivity tspath.CaseSensitivity
	reportMu            sync.Mutex
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
		programs[i] = &program{
			program: sourceProgram,
			threads: make([]*programThread, maxThreads),
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
		RequestedFiles:      requestedFiles,
		FilesById:           filesById,
		Programs:            programs,
		reportBaseDir:       tspath.NormalizePath(cwd),
		pathCaseSensitivity: fs.CaseSensitivity(),
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
	thread := w.Programs[programId].forThread(threadId)
	t := thread.checker.GetTypeAtLocation(node)

	thread.encoder.EncodeType(buf[:0], t)
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
