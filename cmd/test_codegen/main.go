package main

import (
	"fmt"
	"os"

	vue_codegen "github.com/auvred/golar/internal/vue/codegen"
	vue_parser "github.com/auvred/golar/internal/vue/parser"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: test_codegen <vue-file> [--service]")
		os.Exit(1)
	}
	
	filePath := os.Args[1]
	outputService := len(os.Args) > 2 && os.Args[2] == "--service"
	
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading file:", err)
		os.Exit(1)
	}
	
	ast := vue_parser.Parse(string(content))
	serviceCode, _, _ := vue_codegen.Codegen(string(content), ast)
	
	if outputService {
		fmt.Print(serviceCode)
	} else {
		// Default: show AST info
		for i, child := range ast.Children {
			if child.Kind == 1 { // KindElement
				el := child.AsElement()
				fmt.Printf("Child %d: Tag=%s, Children=%d, InnerLoc=%d-%d\n", 
					i, el.Tag, len(el.Children), el.InnerLoc.Pos(), el.InnerLoc.End())
				if el.Tag == "template" {
					for j, tc := range el.Children {
						fmt.Printf("  Template child %d: Kind=%d, Loc=%d-%d\n", j, tc.Kind, tc.Loc.Pos(), tc.Loc.End())
					}
				}
			}
		}
	}
}
