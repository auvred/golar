package volar_comparison

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	vue_codegen "github.com/auvred/golar/internal/vue/codegen"
	vue_parser "github.com/auvred/golar/internal/vue/parser"
)

// TestVolarComparison compares Golar's codegen output with Volar's reference implementation.
// This test requires the .reference/language-tools to be set up with dependencies installed.
//
// To update the reference baselines:
//  1. cd .reference/language-tools && bun install
//  2. Run tests with -update flag (not yet implemented)
//
// The test fixtures are stored in testdata/volar_comparison/
func TestVolarComparison(t *testing.T) {
	// Skip if .reference directory doesn't exist or bun is not available
	refDir := filepath.Join(getProjectRoot(t), ".reference", "language-tools")
	if _, err := os.Stat(refDir); os.IsNotExist(err) {
		t.Skip("Skipping Volar comparison: .reference/language-tools not found")
	}

	if _, err := exec.LookPath("bun"); err != nil {
		t.Skip("Skipping Volar comparison: bun not found in PATH")
	}

	// Check if node_modules exists
	nodeModules := filepath.Join(refDir, "node_modules")
	if _, err := os.Stat(nodeModules); os.IsNotExist(err) {
		t.Skip("Skipping Volar comparison: run 'bun install' in .reference/language-tools first")
	}

	testCases := []struct {
		name    string
		content string
	}{
		{
			name: "simple_interpolation",
			content: `<script setup lang="ts">
const msg = "hello"
</script>
<template>
  <div>{{ msg }}</div>
</template>`,
		},
		{
			name: "v_if_directive",
			content: `<script setup lang="ts">
const show = true
</script>
<template>
  <div v-if="show">Visible</div>
</template>`,
		},
		{
			name: "v_for_directive",
			content: `<script setup lang="ts">
const items = ['a', 'b', 'c']
</script>
<template>
  <div v-for="item in items" :key="item">{{ item }}</div>
</template>`,
		},
		{
			name: "event_handler",
			content: `<script setup lang="ts">
const handleClick = () => console.log('clicked')
</script>
<template>
  <button @click="handleClick">Click me</button>
</template>`,
		},
		{
			name: "compound_event_handler",
			content: `<script setup lang="ts">
let count = 0
</script>
<template>
  <button @click="count++">{{ count }}</button>
</template>`,
		},
		{
			name: "logical_and_in_directive",
			content: `<script setup lang="ts">
const a = true
const b = false
</script>
<template>
  <div v-if="a && b">Both true</div>
</template>`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create temp file
			tmpFile, err := os.CreateTemp("", "*.vue")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(tc.content); err != nil {
				t.Fatalf("Failed to write temp file: %v", err)
			}
			tmpFile.Close()

			// Get Golar output
			golarOutput := getGolarOutput(t, tc.content)

			// Get Volar output
			volarOutput := getVolarOutput(t, tmpFile.Name())

			// For now, just log the outputs for comparison
			// In the future, we can add specific assertions based on what should match
			t.Logf("=== Golar Output ===\n%s", golarOutput)
			t.Logf("=== Volar Output ===\n%s", volarOutput)

			// Basic sanity checks - both should produce non-empty output
			if len(golarOutput) == 0 {
				t.Error("Golar produced empty output")
			}
			if len(volarOutput) == 0 {
				t.Error("Volar produced empty output")
			}

			// Check that key patterns exist in both outputs
			// These are semantic checks, not exact string matches
			checkSemanticEquivalence(t, tc.name, golarOutput, volarOutput, tc.content)
		})
	}
}

func getGolarOutput(t *testing.T, content string) string {
	ast := vue_parser.Parse(content)
	serviceCode, _, _ := vue_codegen.Codegen(content, ast)
	return serviceCode
}

func getVolarOutput(t *testing.T, filePath string) string {
	projectRoot := getProjectRoot(t)
	refDir := filepath.Join(projectRoot, ".reference")
	scriptPath := filepath.Join(refDir, "generate_volar.ts")

	// Check if the generator script exists (created by setup-volar-reference.sh)
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Fatalf("Volar generator script not found. Run: ./scripts/setup-volar-reference.sh")
	}

	cmd := exec.Command("bun", "run", scriptPath, filePath)
	// Run from .reference directory where language-tools and node_modules are installed
	cmd.Dir = refDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to run Volar generator: %v\nStderr: %s\n\nTo set up Volar reference, run: ./scripts/setup-volar-reference.sh", err, stderr.String())
	}

	return stdout.String()
}

func getProjectRoot(t *testing.T) string {
	// Walk up from current directory to find go.mod
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			// Check if this is the golar root (not typescript-go)
			content, _ := os.ReadFile(filepath.Join(dir, "go.mod"))
			if strings.Contains(string(content), "github.com/auvred/golar") {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("Could not find project root")
		}
		dir = parent
	}
}

// checkSemanticEquivalence verifies that key semantic elements are present in both outputs
func checkSemanticEquivalence(t *testing.T, testName, golarOutput, volarOutput, sourceContent string) {
	t.Helper()

	// Extract variable names from script setup
	// Both outputs should reference these variables
	if strings.Contains(sourceContent, "const msg") {
		if !strings.Contains(golarOutput, "msg") {
			t.Error("Golar output missing 'msg' reference")
		}
		if !strings.Contains(volarOutput, "msg") {
			t.Error("Volar output missing 'msg' reference")
		}
	}

	if strings.Contains(sourceContent, "const show") {
		if !strings.Contains(golarOutput, "show") {
			t.Error("Golar output missing 'show' reference")
		}
		if !strings.Contains(volarOutput, "show") {
			t.Error("Volar output missing 'show' reference")
		}
	}

	if strings.Contains(sourceContent, "const items") {
		if !strings.Contains(golarOutput, "items") {
			t.Error("Golar output missing 'items' reference")
		}
		if !strings.Contains(volarOutput, "items") {
			t.Error("Volar output missing 'items' reference")
		}
	}

	// Check v-for generates iteration
	if strings.Contains(sourceContent, "v-for") {
		// Golar uses __VLS_vFor
		if !strings.Contains(golarOutput, "__VLS_vFor") && !strings.Contains(golarOutput, "for") {
			t.Error("Golar output missing v-for iteration construct")
		}
	}

	// Check v-if generates conditional
	if strings.Contains(sourceContent, "v-if") {
		if !strings.Contains(golarOutput, "if") {
			t.Error("Golar output missing v-if conditional")
		}
	}

	// Check event handlers are present
	if strings.Contains(sourceContent, "@click") {
		if !strings.Contains(golarOutput, "click") && !strings.Contains(golarOutput, "Click") {
			t.Error("Golar output missing click handler reference")
		}
	}
}
