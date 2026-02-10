package volar_comparison

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	vue_codegen "github.com/auvred/golar/internal/vue/codegen"
	vue_parser "github.com/auvred/golar/internal/vue/parser"
)

// TestExactVolarMatch tests that Golar's codegen output is byte-for-byte
// identical to Volar's output. This is the primary test for ensuring 1:1
// parity between the two implementations.
//
// Test fixtures are in testdata/exact-match/*.vue
//
// Prerequisites:
//   - Run: ./scripts/setup-volar-reference.sh
//
// To update golden files from Volar:
//   UPDATE_GOLDEN=1 go test ./internal/vue/tests/volar_comparison/... -run TestExactVolarMatch -v
func TestExactVolarMatch(t *testing.T) {
	projectRoot := getProjectRoot(t)
	requireVolarSetup(t, projectRoot)

	updateGolden := os.Getenv("UPDATE_GOLDEN") != ""

	testDir := filepath.Join(projectRoot, "internal/vue/tests/volar_comparison/testdata/exact-match")
	entries, err := os.ReadDir(testDir)
	if err != nil {
		t.Fatalf("Failed to read exact-match testdata: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".vue") {
			continue
		}

		testName := strings.TrimSuffix(entry.Name(), ".vue")
		vueFilePath := filepath.Join(testDir, entry.Name())

		t.Run(testName, func(t *testing.T) {
			content, err := os.ReadFile(vueFilePath)
			if err != nil {
				t.Fatalf("Failed to read %s: %v", vueFilePath, err)
			}

			volarOutput := normalizeReferenceTypes(getVolarOutput(t, vueFilePath, projectRoot))
			golarOutput := getGolarCodegen(t, string(content))

			// Write golden file if requested
			goldenPath := vueFilePath + ".volar.ts"
			if updateGolden {
				if err := os.WriteFile(goldenPath, []byte(volarOutput), 0o644); err != nil {
					t.Fatalf("Failed to write golden file: %v", err)
				}
				t.Logf("Updated golden file: %s", goldenPath)
			}

			if golarOutput == volarOutput {
				t.Logf("PASS: Exact match for %s", entry.Name())
				return
			}

			// Show contextual diff (only lines around changes)
			volarLines := strings.Split(volarOutput, "\n")
			golarLines := strings.Split(golarOutput, "\n")

			t.Errorf("Golar output does not match Volar for %s", entry.Name())
			t.Logf("\n%s", formatContextDiff(volarLines, golarLines, 2))

			// Write outputs to temp files for external diff
			tmpDir := t.TempDir()
			volarFile := filepath.Join(tmpDir, "volar.ts")
			golarFile := filepath.Join(tmpDir, "golar.ts")
			os.WriteFile(volarFile, []byte(volarOutput), 0o644)
			os.WriteFile(golarFile, []byte(golarOutput), 0o644)
			t.Logf("diff -u %s %s", volarFile, golarFile)
		})
	}
}

func requireVolarSetup(t *testing.T, projectRoot string) {
	t.Helper()

	refDir := filepath.Join(projectRoot, ".reference", "language-tools")
	if _, err := os.Stat(refDir); os.IsNotExist(err) {
		t.Skip("Skipping: .reference/language-tools not found. Run: ./scripts/setup-volar-reference.sh")
	}

	if _, err := exec.LookPath("bun"); err != nil {
		t.Skip("Skipping: bun not found in PATH")
	}

	nodeModules := filepath.Join(refDir, "node_modules")
	if _, err := os.Stat(nodeModules); os.IsNotExist(err) {
		t.Skip("Skipping: run './scripts/setup-volar-reference.sh' first")
	}
}

func getGolarCodegen(t *testing.T, content string) string {
	t.Helper()
	ast, _ := vue_parser.Parse(content)
	serviceCode, _, _, _, _ := vue_codegen.Codegen(content, ast, vue_codegen.VueOptions{})
	return normalizeReferenceTypes(serviceCode)
}

// normalizeReferenceTypes normalizes /// <reference types="..." /> paths
// so they can be compared between Volar and Golar outputs.
// Both use different base paths but the same file names (template-helpers.d.ts, props-fallback.d.ts).
var referenceTypeRe = regexp.MustCompile(`/// <reference types="[^"]*/(template-helpers\.d\.ts|props-fallback\.d\.ts)" />`)

func normalizeReferenceTypes(s string) string {
	return referenceTypeRe.ReplaceAllString(s, `/// <reference types="$1" />`)
}

// formatContextDiff shows only changed lines with surrounding context.
func formatContextDiff(expected, actual []string, contextLines int) string {
	var sb strings.Builder
	sb.WriteString("--- Volar (expected)\n")
	sb.WriteString("+++ Golar (actual)\n\n")

	maxLen := len(expected)
	if len(actual) > maxLen {
		maxLen = len(actual)
	}

	// First pass: identify which lines differ
	type diffLine struct {
		lineNum  int
		expLine  string
		actLine  string
		isDiff   bool
		expOnly  bool // line only exists in expected
		actOnly  bool // line only exists in actual
	}

	var diffs []diffLine
	for i := 0; i < maxLen; i++ {
		d := diffLine{lineNum: i + 1}
		if i < len(expected) {
			d.expLine = expected[i]
		}
		if i < len(actual) {
			d.actLine = actual[i]
		}

		if i >= len(expected) {
			d.isDiff = true
			d.actOnly = true
		} else if i >= len(actual) {
			d.isDiff = true
			d.expOnly = true
		} else if d.expLine != d.actLine {
			d.isDiff = true
		}
		diffs = append(diffs, d)
	}

	// Second pass: print with context, grouping nearby changes into hunks
	matchCount := 0
	diffCount := 0
	inHunk := false
	lastDiff := -1

	for i, d := range diffs {
		if d.isDiff {
			diffCount++
			// Print context before this diff if not already in a hunk
			if !inHunk || i-lastDiff > contextLines*2+1 {
				if inHunk {
					sb.WriteString("  ...\n\n")
				}
				// Print preceding context
				for j := max(0, i-contextLines); j < i; j++ {
					if !diffs[j].isDiff {
						sb.WriteString(fmt.Sprintf(" %4d | %s\n", diffs[j].lineNum, diffs[j].expLine))
					}
				}
			}
			inHunk = true
			lastDiff = i

			// Print the diff
			if d.expOnly {
				sb.WriteString(fmt.Sprintf("-%4d | %s\n", d.lineNum, d.expLine))
			} else if d.actOnly {
				sb.WriteString(fmt.Sprintf("+%4d | %s\n", d.lineNum, d.actLine))
			} else {
				sb.WriteString(fmt.Sprintf("-%4d | %s\n", d.lineNum, d.expLine))
				sb.WriteString(fmt.Sprintf("+%4d | %s\n", d.lineNum, d.actLine))
			}
		} else {
			matchCount++
			// Print trailing context after last diff
			if inHunk && i-lastDiff <= contextLines {
				sb.WriteString(fmt.Sprintf(" %4d | %s\n", d.lineNum, d.expLine))
			}
		}
	}
	if inHunk {
		sb.WriteString("  ...\n")
	}

	sb.WriteString(fmt.Sprintf("\nSummary: %d/%d lines match, %d differ (volar: %d lines, golar: %d lines)\n",
		matchCount, maxLen, diffCount, len(expected), len(actual)))

	return sb.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
