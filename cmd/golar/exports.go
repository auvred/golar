package main

/*
#include <stdint.h>
#include <stdlib.h>

void napi_call_threadsafe_function_any(uintptr_t func, uintptr_t data, size_t is_blocking);

#ifdef _WIN32
#define GOLAR_EXPORT
#else
#define GOLAR_EXPORT __attribute__((visibility("default")))
#endif

typedef struct {
	uintptr_t program;
	uintptr_t source_file;
} golar_file_with_program;

extern golar_file_with_program golar_workspace_get_requested_file(uintptr_t workspacePtr, uint32_t fileIdx) GOLAR_EXPORT;
extern uintptr_t golar_program_get_type_at_location(uintptr_t programPtr, uintptr_t nodePtr) GOLAR_EXPORT;
extern void golar_workspace_report(uintptr_t workspacePtr, uint32_t fileIdx, int32_t start, int32_t end, char *ruleNameData, size_t ruleNameLen, char *messageData, size_t messageLen) GOLAR_EXPORT;
*/
import "C"

import (
	"context"
	"encoding/binary"
	"runtime"
	"slices"
	"unsafe"

	"github.com/auvred/golar/internal/linter"
	"github.com/auvred/golar/internal/linter/rule"
	"github.com/auvred/golar/internal/utils"
	"github.com/auvred/golar/internal/workspace"
	"github.com/microsoft/typescript-go/pkg/ast"
	"github.com/microsoft/typescript-go/pkg/compiler"
	"github.com/microsoft/typescript-go/pkg/core"
	"github.com/microsoft/typescript-go/pkg/execute"
)

type SyncBuffer []byte

func (b SyncBuffer) readString(pos uint32, len uint32) string {
	return C.GoStringN((*C.char)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(b)), pos)), C.int(len))
}

func (b SyncBuffer) readStringsList(offset uint32) ([]string, uint32) {
	count := binary.LittleEndian.Uint32(b[offset:])
	offset += 4
	strings := make([]string, count)
	for i := range count {
		argLen := binary.LittleEndian.Uint32(b[offset:])
		offset += 4
		strings[i] = b.readString(offset, argLen)
		offset += argLen
	}
	return strings, offset
}

const maxThreads = 64

var syncBuffers [maxThreads]SyncBuffer

//export golar_js_setSyncBuffer
func golar_js_setSyncBuffer(threadId C.uint32_t, bufPtr C.uintptr_t, bufLen C.size_t) {
	buf := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(bufPtr))), bufLen)
	syncBuffers[threadId] = buf
}

//export golar_js_linter_RuleTesterLint
func golar_js_linter_RuleTesterLint(files_data *C.char, files_len C.size_t, fileName_data *C.char, fileName_len C.size_t, ruleName_data *C.char, ruleName_len C.size_t, options_data *C.char, options_len C.size_t) (*C.char, C.size_t) {
	res := linter.RuleTesterLint(C.GoStringN(files_data, C.int(files_len)), C.GoStringN(fileName_data, C.int(fileName_len)), C.GoStringN(ruleName_data, C.int(ruleName_len)), C.GoStringN(options_data, C.int(options_len)))
	return C.CString(res), C.size_t(len(res))
}

var pinner runtime.Pinner

func castWorkspace(ptr uint64) *workspace.Workspace {
	return (*workspace.Workspace)(unsafe.Pointer(uintptr(ptr)))
}
func castNode(ptr uint64) *ast.Node {
	return (*ast.Node)(unsafe.Pointer(uintptr(ptr)))
}

//export golar_js_workspace_New
func golar_js_workspace_New(cbPtr C.uintptr_t) {
	buf := syncBuffers[0]

	var offset uint32
	cwdLen := binary.LittleEndian.Uint32(buf[offset:])
	offset += 4
	cwd := buf.readString(offset, cwdLen)
	offset += cwdLen
	filenameCount := binary.LittleEndian.Uint32(buf[offset:])
	offset += 4

	filenames := make([]string, 0, filenameCount)
	for range filenameCount {
		filenameLen := binary.LittleEndian.Uint32(buf[offset:])
		offset += 4
		filenames = append(filenames, buf.readString(offset, filenameLen))
		offset += filenameLen
	}

	go func() {
		w := workspace.New(cwd, filenames)
		// TODO: unpin & destroy
		pinner.Pin(w)
		offset = 0
		binary.LittleEndian.PutUint64(buf[offset:], uint64(uintptr(unsafe.Pointer(w))))
		offset += 8
		binary.LittleEndian.PutUint32(buf[offset:], uint32(len(w.Programs)))
		offset += 4
		binary.LittleEndian.PutUint32(buf[offset:], uint32(len(w.FilesById)))
		C.napi_call_threadsafe_function_any(cbPtr, 0, 0)
	}()
}

//export golar_js_workspace_ReadRequestedFileAt
func golar_js_workspace_ReadRequestedFileAt(threadId C.uint32_t) {
	buf := syncBuffers[threadId]

	offset := 0
	workspacePtr := binary.LittleEndian.Uint64(buf[offset:])
	offset += 8
	index := binary.LittleEndian.Uint32(buf[offset:])
	offset += 4
	file, encodedFile := castWorkspace(workspacePtr).ReadRequestedFileAt(index)

	offset = 0
	binary.LittleEndian.PutUint32(buf[offset:], file.ProgramId)
	offset += 4
	binary.LittleEndian.PutUint32(buf[offset:], file.SourceFile.Id)
	offset += 4
	binary.LittleEndian.PutUint32(buf[offset:], uint32(len(encodedFile)))
	offset += 4
	copy(buf[offset:], encodedFile)
}

//export golar_js_workspace_ReadFileById
func golar_js_workspace_ReadFileById(threadId C.uint32_t) {
	buf := syncBuffers[threadId]

	offset := 0
	workspacePtr := binary.LittleEndian.Uint64(buf[offset:])
	offset += 8
	id := binary.LittleEndian.Uint32(buf[offset:])
	encodedFile := castWorkspace(workspacePtr).ReadFileById(id)

	offset = 0
	binary.LittleEndian.PutUint32(buf[offset:], uint32(len(encodedFile)))
	offset += 4
	copy(buf[offset:], encodedFile)
}

//export golar_js_workspace_GetTypeAtLocation
func golar_js_workspace_GetTypeAtLocation(threadId C.uint32_t) {
	buf := syncBuffers[threadId]

	offset := 0
	workspacePtr := binary.LittleEndian.Uint64(buf[offset:])
	offset += 8
	programId := binary.LittleEndian.Uint32(buf[offset:])
	offset += 4
	nodePtr := binary.LittleEndian.Uint64(buf[offset:])
	offset += 8
	castWorkspace(workspacePtr).GetTypeAtLocation(buf, uint32(threadId), programId, castNode(nodePtr))
}

//export golar_js_workspace_Lint
func golar_js_workspace_Lint() {
	buf := syncBuffers[0]

	var offset uint32
	workspacePtr := binary.LittleEndian.Uint64(buf[offset:])
	offset += 8
	payloadLen := binary.LittleEndian.Uint32(buf[offset:])
	offset += 4
	payload := slices.Clone(buf[offset : offset+payloadLen])
	castWorkspace(workspacePtr).Lint(payload)
}

//export golar_js_workspace_Report
func golar_js_workspace_Report(threadId C.uint32_t) {
	buf := syncBuffers[threadId]

	offset := uint32(0)
	workspacePtr := binary.LittleEndian.Uint64(buf[offset:])
	offset += 8
	fileIdx := binary.LittleEndian.Uint32(buf[offset:])
	offset += 4
	start := binary.LittleEndian.Uint32(buf[offset:])
	offset += 4
	end := binary.LittleEndian.Uint32(buf[offset:])
	offset += 4
	ruleNameLen := binary.LittleEndian.Uint32(buf[offset:])
	offset += 4
	ruleName := buf.readString(offset, ruleNameLen)
	offset += ruleNameLen
	messageLen := binary.LittleEndian.Uint32(buf[offset:])
	offset += 4
	message := buf.readString(offset, messageLen)

	castWorkspace(workspacePtr).ReportRequestedFile(fileIdx, ruleName, rule.Report{
		Range:   core.NewTextRange(int(start), int(end)),
		Message: message,
	})
}

//export golar_js_tsc
func golar_js_tsc(threadId C.uint32_t, doneCbTsfn C.uintptr_t) {
	buf := syncBuffers[threadId]

	argv, _ := buf.readStringsList(0)

	go func() {
		result := execute.CommandLine(context.Background(), utils.NewOsSystem(), argv, nil)
		C.napi_call_threadsafe_function_any(doneCbTsfn, C.uintptr_t(result.Status), 0)
	}()
}

//export golar_js_workspace_TypeCheck
func golar_js_workspace_TypeCheck() C.uint32_t {
	buf := syncBuffers[0]

	offset := uint32(0)
	workspacePtr := binary.LittleEndian.Uint64(buf[offset:])
	offset += 8

	return C.uint32_t(castWorkspace(workspacePtr).TypeCheck(buf[offset:]))
}

//export golar_workspace_get_requested_file
func golar_workspace_get_requested_file(workspacePtr C.uintptr_t, fileIdx C.uint32_t) C.golar_file_with_program {
	w := castWorkspace(uint64(workspacePtr))
	file := w.RequestedFiles[fileIdx]

	return C.golar_file_with_program{
		program:     C.uintptr_t(uintptr(unsafe.Pointer(file.Program))),
		source_file: C.uintptr_t(uintptr(unsafe.Pointer(file.SourceFile))),
	}
}

//export golar_program_get_type_at_location
func golar_program_get_type_at_location(programPtr C.uintptr_t, nodePtr C.uintptr_t) C.uintptr_t {
	program := (*compiler.Program)(unsafe.Pointer(uintptr(programPtr)))
	node := (*ast.Node)(unsafe.Pointer(uintptr(nodePtr)))

	typeChecker, done := program.GetTypeChecker(context.Background())
	defer done()

	t := typeChecker.GetTypeAtLocation(node)
	if t == nil {
		return 0
	}
	return C.uintptr_t(uintptr(unsafe.Pointer(t)))
}

//export golar_workspace_report
func golar_workspace_report(workspacePtr C.uintptr_t, fileIdx C.uint32_t, start C.int32_t, end C.int32_t, ruleNameData *C.char, ruleNameLen C.size_t, messageData *C.char, messageLen C.size_t) {
	castWorkspace(uint64(workspacePtr)).ReportRequestedFile(uint32(fileIdx), C.GoStringN(ruleNameData, C.int(ruleNameLen)), rule.Report{
		Range:   core.NewTextRange(int(start), int(end)),
		Message: C.GoStringN(messageData, C.int(messageLen)),
	})
}
