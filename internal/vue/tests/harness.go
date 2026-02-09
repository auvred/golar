package vue_tests

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/testutil"
	"gotest.tools/v3/assert"
)

func ptrTo[T any](v T) *T {
	return &v
}

type vueVersion string

const (
	vue_3_2 vueVersion = "vue-3.2"
	vue_3_3 vueVersion = "vue-3.3"
	vue_3_4 vueVersion = "vue-3.4"
	vue_3_5 vueVersion = "vue-3.5"
	vue_3_6 vueVersion = "vue-3.6"
)

func withVueNodeModules(t *testing.T, version vueVersion, content string) string {
	_, filename, _, _ := runtime.Caller(1)
	dirname := filepath.Join(filepath.Dir(filename), string(version))
	var extraFilesBuilder strings.Builder
	extraFilesBuilder.WriteString("// @golarExtraFiles: ")

	err := filepath.Walk(filepath.Join(dirname, "node_modules"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && (strings.HasSuffix(path, ".d.ts") || strings.HasSuffix(path, ".d.mts") || strings.HasSuffix(path, ".d.cts") || filepath.Base(path) == "package.json") {
			p, err := filepath.Rel(dirname, path)
			if err != nil {
				return err
			}
			virtualPath := filepath.Join("/", p)

			// https://en.wikipedia.org/wiki/Delimiter#Control_characters
			extraFilesBuilder.WriteString(path)
			extraFilesBuilder.WriteByte('\x1e')
			extraFilesBuilder.WriteString(virtualPath)
			extraFilesBuilder.WriteByte('\x1f')
		}
		return nil
	})
	assert.NilError(t, err)
	extraFilesBuilder.WriteByte('\n')

	return extraFilesBuilder.String() + content
}

func runFourslashTest(t *testing.T, content string, run func(t *testing.T, f *fourslash.FourslashTest, version vueVersion)) {
	t.Parallel()
	for _, version := range []vueVersion{vue_3_2, vue_3_3, vue_3_4, vue_3_5, vue_3_6} {
		t.Run(string(version), func(t *testing.T) {
			defer testutil.RecoverAndFail(t, "Panic on fourslash test")

			contentWithNodeModules := withVueNodeModules(t, version, content)
			f, done := fourslash.NewFourslash(t, nil, contentWithNodeModules)
			defer done()
			run(t, f, version)
		})
	}
}
