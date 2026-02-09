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

// TestVolarComparison runs comparison tests between Golar and Volar codegen.
//
// Prerequisites:
//   - Run: ./scripts/setup-volar-reference.sh
//
// Test cases are in testdata/ directory, copied from Volar's test-workspace.
// Each test case has:
//   - main.vue: The Vue component to test
//   - *.vue: Additional Vue files if needed
//   - tsconfig.json: TypeScript configuration
//
// To sync test cases from Volar:
//   - Run: ./scripts/sync-volar-tests.sh
func TestVolarComparison(t *testing.T) {
	projectRoot := getProjectRoot(t)
	refDir := filepath.Join(projectRoot, ".reference", "language-tools")

	// Skip if reference not set up
	if _, err := os.Stat(refDir); os.IsNotExist(err) {
		t.Skip("Skipping: .reference/language-tools not found. Run: ./scripts/setup-volar-reference.sh")
	}

	if _, err := exec.LookPath("bun"); err != nil {
		t.Skip("Skipping: bun not found in PATH")
	}

	// Check if bun install was run
	nodeModules := filepath.Join(refDir, "node_modules")
	if _, err := os.Stat(nodeModules); os.IsNotExist(err) {
		t.Skip("Skipping: run './scripts/setup-volar-reference.sh' first")
	}

	testdataDir := filepath.Join(projectRoot, "internal/vue/tests/volar_comparison/testdata")
	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		t.Fatalf("Failed to read testdata directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		testName := entry.Name()
		testDir := filepath.Join(testdataDir, testName)

		// Find main.vue or any .vue file
		vueFile := findVueFile(t, testDir)
		if vueFile == "" {
			continue
		}

		t.Run(testName, func(t *testing.T) {
			content, err := os.ReadFile(vueFile)
			if err != nil {
				t.Fatalf("Failed to read %s: %v", vueFile, err)
			}

			golarOutput := getGolarOutput(t, string(content))
			volarOutput := getVolarOutput(t, vueFile, projectRoot)

			// Basic sanity checks
			if len(golarOutput) == 0 {
				t.Error("Golar produced empty output")
			}
			if len(volarOutput) == 0 {
				t.Error("Volar produced empty output")
			}

			// Log outputs for debugging (only shown on failure or -v)
			t.Logf("\n=== Source ===\n%s", string(content))
			t.Logf("\n=== Golar Output ===\n%s", golarOutput)
			t.Logf("\n=== Volar Output ===\n%s", volarOutput)

			// Check semantic equivalence
			checkSemanticEquivalence(t, string(content), golarOutput, volarOutput)
		})
	}
}

// TestInlineComparison tests specific inline Vue code snippets
func TestInlineComparison(t *testing.T) {
	projectRoot := getProjectRoot(t)
	refDir := filepath.Join(projectRoot, ".reference", "language-tools")

	if _, err := os.Stat(refDir); os.IsNotExist(err) {
		t.Skip("Skipping: run './scripts/setup-volar-reference.sh' first")
	}

	if _, err := exec.LookPath("bun"); err != nil {
		t.Skip("Skipping: bun not found in PATH")
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
			name: "event_handler_simple",
			content: `<script setup lang="ts">
const handleClick = () => console.log('clicked')
</script>
<template>
  <button @click="handleClick">Click me</button>
</template>`,
		},
		{
			name: "event_handler_compound",
			content: `<script setup lang="ts">
let count = 0
</script>
<template>
  <button @click="count++">{{ count }}</button>
</template>`,
		},
		{
			name: "v_if_with_logical_and",
			content: `<script setup lang="ts">
const a = true
const b = false
</script>
<template>
  <div v-if="a && b">Both true</div>
</template>`,
		},
		{
			name: "v_for_with_destructuring",
			content: `<script setup lang="ts">
const items = [{ id: 1, name: 'foo' }, { id: 2, name: 'bar' }]
</script>
<template>
  <div v-for="{ id, name } in items" :key="id">{{ name }}</div>
</template>`,
		},
		{
			name: "v_if_else_chain",
			content: `<script setup lang="ts">
const type = 'a'
</script>
<template>
  <div v-if="type === 'a'">A</div>
  <div v-else-if="type === 'b'">B</div>
  <div v-else>Other</div>
</template>`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create temp file for Volar
			tmpFile, err := os.CreateTemp("", "*.vue")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(tc.content); err != nil {
				t.Fatalf("Failed to write temp file: %v", err)
			}
			tmpFile.Close()

			golarOutput := getGolarOutput(t, tc.content)
			volarOutput := getVolarOutput(t, tmpFile.Name(), projectRoot)

			if len(golarOutput) == 0 {
				t.Error("Golar produced empty output")
			}
			if len(volarOutput) == 0 {
				t.Error("Volar produced empty output")
			}

			t.Logf("\n=== Source ===\n%s", tc.content)
			t.Logf("\n=== Golar Output ===\n%s", golarOutput)
			t.Logf("\n=== Volar Output ===\n%s", volarOutput)

			checkSemanticEquivalence(t, tc.content, golarOutput, volarOutput)
		})
	}
}

func findVueFile(t *testing.T, dir string) string {
	// Prefer main.vue
	mainVue := filepath.Join(dir, "main.vue")
	if _, err := os.Stat(mainVue); err == nil {
		return mainVue
	}

	// Fall back to any .vue file
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".vue") {
			return filepath.Join(dir, entry.Name())
		}
	}

	return ""
}

func getGolarOutput(t *testing.T, content string) string {
	ast, _ := vue_parser.Parse(content)
	serviceCode, _, _, _, _ := vue_codegen.Codegen(content, ast, vue_codegen.VueOptions{})
	return serviceCode
}

func getVolarOutput(t *testing.T, filePath, projectRoot string) string {
	scriptPath := filepath.Join(projectRoot, "tools/volar/generate_volar.ts")

	// Check if the generator script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Fatalf("Generator script not found at %s", scriptPath)
	}

	cmd := exec.Command("bun", "run", scriptPath, filePath)
	cmd.Dir = projectRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to run Volar generator: %v\nStderr: %s", err, stderr.String())
	}

	return stdout.String()
}

func getProjectRoot(t *testing.T) string {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	for {
		goMod := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goMod); err == nil {
			content, _ := os.ReadFile(goMod)
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

// checkSemanticEquivalence verifies key semantic elements are present in both outputs
func checkSemanticEquivalence(t *testing.T, source, golarOutput, volarOutput string) {
	t.Helper()

	// Both should have __VLS_ctx for template bindings
	if !strings.Contains(golarOutput, "__VLS_ctx") {
		t.Error("Golar output missing __VLS_ctx")
	}
	if !strings.Contains(volarOutput, "__VLS_ctx") {
		t.Error("Volar output missing __VLS_ctx")
	}

	// Check v-for generates iteration
	if strings.Contains(source, "v-for") {
		if !strings.Contains(golarOutput, "__VLS_vFor") {
			t.Error("Golar output missing __VLS_vFor for v-for directive")
		}
		if !strings.Contains(volarOutput, "__VLS_vFor") {
			t.Error("Volar output missing __VLS_vFor for v-for directive")
		}
	}

	// Check v-if generates conditional
	if strings.Contains(source, "v-if") {
		if !strings.Contains(golarOutput, "if (") && !strings.Contains(golarOutput, "if(") {
			t.Error("Golar output missing if statement for v-if directive")
		}
		if !strings.Contains(volarOutput, "if (") && !strings.Contains(volarOutput, "if(") {
			t.Error("Volar output missing if statement for v-if directive")
		}
	}

	// Check v-else generates else block
	if strings.Contains(source, "v-else") {
		if !strings.Contains(golarOutput, "else") {
			t.Error("Golar output missing else for v-else directive")
		}
		if !strings.Contains(volarOutput, "else") {
			t.Error("Volar output missing else for v-else directive")
		}
	}

	// Check event handlers
	if strings.Contains(source, "@click") {
		// Should have onClick or click in the output
		hasClick := strings.Contains(golarOutput, "onClick") ||
			strings.Contains(golarOutput, "Click") ||
			strings.Contains(golarOutput, "click")
		if !hasClick {
			t.Error("Golar output missing click handler")
		}
	}

	// Check intrinsic elements
	if strings.Contains(source, "<div") || strings.Contains(source, "<button") {
		if !strings.Contains(golarOutput, "__VLS_intrinsics") {
			t.Error("Golar output missing __VLS_intrinsics for intrinsic elements")
		}
	}
}
