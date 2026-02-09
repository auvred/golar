package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/auvred/golar/plugin"

	"github.com/withastro/compiler/shim"
	"github.com/withastro/compiler/shim/handler"
	"github.com/withastro/compiler/shim/printer"
	"github.com/withastro/compiler/shim/transform"
)

func getAstroInstallation() (string, string, error) {
	dir, _ := os.Getwd()

	var packageJson struct { Version string }
	var astroDir string
	for {
		content, err := os.ReadFile(filepath.Join(dir, "node_modules", "astro", "package.json"))
		if err == nil {
			if err := json.Unmarshal(content, &packageJson); err != nil {
				panic(err)
			}
			astroDir = filepath.Join(dir, "node_modules", "astro")
			break
		}
		parentDir := filepath.Dir(dir)
		if dir == parentDir {
			return "", "", errors.New("Cannot find 'astro' package.")
		}
		dir = parentDir
	}


	return astroDir, packageJson.Version, nil
}

func main() {
	astroDir, _, err := getAstroInstallation()
	if err != nil {
		panic(err)
	}

	astroFooter := []byte("\n\nimport '" + filepath.Join(astroDir, "env.d.ts") + "'\nimport '" + filepath.Join(astroDir, "env.d.ts") + "'\n")

	plugin.Run(plugin.PluginOptions{
		Input: os.Stdin,
		Output: os.Stdout,
		ExtraExtensions: []string{".astro"},
		CreateServiceCodeWithSourceMap: func (fileName string, sourceText string) *plugin.ServiceCodeWithSourceMap {
			transformOptions := transform.TransformOptions{
				Scope: "xxxxxx",
				Filename: fileName,
				NormalizedFilename: fileName,
			}
			h := handler.NewHandler(sourceText, transformOptions.Filename)

			var doc *astro.Node
			doc, err := astro.ParseWithOptions(strings.NewReader(sourceText), astro.ParseOptionWithHandler(h), astro.ParseOptionEnableLiteral(true))
			if err != nil {
				h.AppendError(err)
			}

			tsxOptions := printer.TSXOptions{}

			printed := printer.PrintToTSX(sourceText, doc, tsxOptions, transformOptions, h)

			// AFTER printing, exec transformations to pickup any errors/warnings
			transform.Transform(doc, transformOptions, h)

			printed.Output = append(printed.Output, astroFooter...)

			result := &plugin.ServiceCodeWithSourceMap{
				ScriptKind: plugin.ScriptKindTSX,
				ServiceText: printed.Output,
				Mappings: printed.SourceMapChunk.Buffer,
			}


			return result
		},
	})
}
