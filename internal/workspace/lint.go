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
	reportFile := golar.ReportingSourceFile(file)

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
	w.reportCount++
	fmt.Print(output.String())
}

// ReportCount returns how many lint reports were printed so far.
func (w *Workspace) ReportCount() uint32 {
	w.reportMu.Lock()
	defer w.reportMu.Unlock()
	return w.reportCount
}

func (w *Workspace) ReportRequestedFile(fileIdx uint32, ruleName string, report rule.Report) {
	w.report(w.RequestedFiles[fileIdx].SourceFile, ruleName, report)
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
