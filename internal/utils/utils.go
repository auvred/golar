package utils

import (
	"unicode"
	"unicode/utf8"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/core"
	"github.com/microsoft/typescript-go/shim/scanner"
)

// https://tc39.es/ecma262/2025/multipage/ecmascript-language-lexical-grammar.html#sec-white-space
// https://tc39.es/ecma262/2025/multipage/ecmascript-language-lexical-grammar.html#sec-line-terminators
func IsWhiteSpaceOrLineTerminator(r rune) bool {
	switch r {
	// LineTerminator
	case '\n', '\r', 0x2028, 0x2029:
		return true
	// WhiteSpace
	case '\t', '\v', '\f', 0xFEFF:
		return true
	}

	// WhiteSpace
	return unicode.Is(unicode.Zs, r)
}

func TrimWhiteSpaceOrLineTerminator(str string) (string, int, int) {
	var trimmedLeft, trimmedRight int
	for len(str) > 0 {
		r, n := utf8.DecodeRuneInString(str)
		if !IsWhiteSpaceOrLineTerminator(r) {
			break
		}
		str = str[n:]
		trimmedLeft += n
	}

	for len(str) > 0 {
		r, n := utf8.DecodeLastRuneInString(str)
		if !IsWhiteSpaceOrLineTerminator(r) {
			break
		}
		str = str[:len(str)-n]
		trimmedRight += n
	}

	return str, trimmedLeft, trimmedRight
}

func TrimNodeTextRange(sourceFile *ast.SourceFile, node *ast.Node) core.TextRange {
	pos := scanner.SkipTrivia(sourceFile.Text(), node.Pos())
	return core.NewTextRange(pos, max(pos, node.End()))
	// return scanner.GetRangeOfTokenAtPosition(sourceFile, node.Pos()).WithEnd(node.End())
}

func MoveTextRange(loc core.TextRange, number int) core.TextRange {
	return core.NewTextRange(number+loc.Pos(), number+loc.End())
}
