package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/auvred/golar/internal/golar"
	"github.com/auvred/golar/internal/linter"
	"github.com/auvred/golar/internal/linter/rule"
	"github.com/microsoft/TypeScript/tsc/pkg/ast"
	"github.com/microsoft/TypeScript/tsc/pkg/core"
	"github.com/microsoft/TypeScript/tsc/pkg/scanner"
	"github.com/microsoft/TypeScript/tsc/pkg/tspath"
)

type LintConfigRule struct {
	Name    string          `json:"name"`
	Options json.RawMessage `json:"options"`
}

type LintConfigFile struct {
	File        uint32   `json:"file"`
	RuleIndexes []uint32 `json:"ruleIndexes"`
}

// TODO: binary encoding
type LintConfig struct {
	Rules []LintConfigRule `json:"rules"`
	Files []LintConfigFile `json:"files"`
}

func (w *Workspace) reportFileName(file *ast.SourceFile) string {
	if w.reportBaseDir == "" {
		return file.FileName().AsString()
	}

	relative := tspath.GetRelativePathFromDirectory(w.reportBaseDir, file.FileName().AsString(), w.pathCaseSensitivity)
	if relative == "" {
		return file.FileName().AsString()
	}

	return relative
}

func (w *Workspace) report(file *ast.SourceFile, ruleName string, report rule.Report) {
	mapped, ok := golar.ServiceToSource(file, report.Range)
	if !ok {
		return
	}
	report.Range = mapped
	w.print(golar.ReportingSourceFile(file), ruleName, report)
}

func (w *Workspace) print(reportFile *ast.SourceFile, ruleName string, report rule.Report) {
	line, column := scanner.GetECMALineAndUTF16CharacterOfPosition(reportFile, report.Range.Pos())
	prefix := w.reportFileName(reportFile) + ":" + strconv.Itoa(line+1) + ":" + strconv.Itoa(int(column)+1) + ": " + ruleName + ": "
	indent := strings.Repeat(" ", len(prefix))
	messageLines := strings.Split(report.Message, "\n")

	var output strings.Builder
	for i, messageLine := range messageLines {
		if i == 0 {
			output.WriteString(prefix)
		} else {
			output.WriteByte('\n')
			output.WriteString(indent)
		}
		output.WriteString(messageLine)
	}
	output.WriteByte('\n')

	w.reportMu.Lock()
	defer w.reportMu.Unlock()
	fmt.Print(output.String())
}

func (w *Workspace) ReportRequestedFile(fileIdx uint32, ruleName string, report rule.Report) {
	w.report(w.RequestedFiles[fileIdx].SourceFile, ruleName, report)
}

// ReportRequestedFileSourceRange reports a range of the original file text,
// in UTF-16 offsets, without mapping it from service code.
func (w *Workspace) ReportRequestedFileSourceRange(fileIdx uint32, ruleName string, report rule.Report) {
	reportFile := golar.ReportingSourceFile(w.RequestedFiles[fileIdx].SourceFile)
	positions := reportFile.GetPositionMap()
	report.Range = core.NewTextRange(positions.UTF16ToUTF8(report.Range.Pos()), positions.UTF16ToUTF8(report.Range.End()))
	w.print(reportFile, ruleName, report)
}

func (w *Workspace) Lint(configRaw []byte) {
	var config LintConfig
	if err := json.Unmarshal(configRaw, &config); err != nil {
		panic(err)
	}

	for _, fileConfig := range config.Files {
		if int(fileConfig.File) >= len(w.RequestedFiles) {
			panic(fmt.Errorf("invalid lint file id %d", fileConfig.File))
		}

		file := w.RequestedFiles[fileConfig.File]
		typeChecker, done := file.Program.GetTypeChecker(context.Background())

		for _, ruleID := range fileConfig.RuleIndexes {
			if int(ruleID) >= len(config.Rules) {
				panic(fmt.Errorf("invalid lint rule id %d", ruleID))
			}

			ruleConfig := config.Rules[ruleID]
			listeners := linter.ConfigureRule(ruleConfig.Name, ruleConfig.Options, file.Program, typeChecker, file.SourceFile, func(report rule.Report) {
				w.report(file.SourceFile, ruleConfig.Name, report)
			})
			linter.VisitSourceFile(file.SourceFile, func(kind ast.Kind, node *ast.Node) {
				if listener, ok := listeners[kind]; ok {
					listener(node)
				}
			})
		}

		done()
	}
}
