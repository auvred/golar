package vue_codegen

import (
	"strconv"
	"strings"
	"testing"

	"github.com/auvred/golar/internal/vue/ast"
	"github.com/auvred/golar/internal/vue/parser"
	"github.com/microsoft/typescript-go/shim/core"

	// "gotest.tools/v3/assert"
	"github.com/google/go-cmp/cmp"
)

func TestExpressionMapper(t *testing.T) {
	t.Run("non-binding position", func(t *testing.T) {
		cases := []struct {
			sourceText  string
			serviceText string
		}{
			{
				"hello",
				"__VLS_Ctx.hello",
			},
			{
				"hello.world",
				"__VLS_Ctx.hello.world",
			},
			{
				"hello[world]",
				"__VLS_Ctx.hello[__VLS_Ctx.world]",
			},
			{
				"() => { const foo: SomeType = bar }",
				"() => { const foo: SomeType = __VLS_Ctx.bar }",
			},
			{
				"() => { return foo }",
				"() => { return __VLS_Ctx.foo }",
			},
			{
				"{ a: a }",
				"{ a: __VLS_Ctx.a }",
			},
			{
				"/*syntax errors*/{ a:  }",
				"/*syntax errors*/{ a:  __VLS_Ctx.}",
			},
			{
				"{ a }",
				"{a: __VLS_Ctx. a }",
			},
			{
				"{ [a]: a }",
				"{ [__VLS_Ctx.a]: __VLS_Ctx.a }",
			},
			{
				"() => { class foo { bar: Foo } }",
				"() => { class foo { bar: Foo } }",
			},
			{
				"() => { interface foo { hello: world } }",
				"() => { interface foo { hello: world } }",
			},
			{
				"() => { interface foo<T extends Foo> {} }",
				"() => { interface foo<T extends Foo> {} }",
			},
			{
				"() => { interface foo extends bar {} }",
				"() => { interface foo extends bar {} }",
			},
			{
				"() => { foo: while (1) break foo }",
				"() => { foo: while (1) break foo }",
			},
			{
				"() => { foo: while (1) continue foo }",
				"() => { foo: while (1) continue foo }",
			},
			{
				"() => { type foo = bar }",
				"() => { type foo = bar }",
			},
			{
				"() => { enum foo { a = hello, b }}",
				"() => { enum foo { a = __VLS_Ctx.hello, b }}",
			},
			{
				"() => { const [, value] = foo }",
				"() => { const [, value] = __VLS_Ctx.foo }",
			},
			{
				"() => { const { [foo]: bar = baz } = qux }",
				"() => { const { [__VLS_Ctx.foo]: bar = __VLS_Ctx.baz } = __VLS_Ctx.qux }",
			},
			{
				"{ ...foo }",
				"{ ...__VLS_Ctx.foo }",
			},
			{
				"() => { module foo { bar } }",
				"() => { module foo { bar } }",
			},
			{
				"() => { namespace foo { bar } }",
				"() => { namespace foo { bar } }",
			},
			{
				"() => { import foo from 'bar' }",
				"() => { import foo from 'bar' }",
			},
			{
				"() => { import * as foo from 'bar' }",
				"() => { import * as foo from 'bar' }",
			},
			{
				"() => { import(foo) }",
				"() => { import(__VLS_Ctx.foo) }",
			},
			{
				"[foo]",
				"[__VLS_Ctx.foo]",
			},
			{
				"() => { class foo { constructor(readonly foo) { foo } } }",
				"() => { class foo { constructor(readonly foo) { foo } } }",
			},
			{
				"foo()",
				"__VLS_Ctx.foo()",
			},
			{
				"new foo()",
				"new __VLS_Ctx.foo()",
			},
			{
				"() => { class foo { hello = bar } }",
				"() => { class foo { hello = __VLS_Ctx.bar } }",
			},
			{
				"() => { class foo { [hello] = bar } }",
				"() => { class foo { [__VLS_Ctx.hello] = __VLS_Ctx.bar } }",
			},
			{
				"() => { class foo { accessor hello: string  } }",
				"() => { class foo { accessor hello: string  } }",
			},
			{
				"() => { class foo { get hello<T extends Foo>() {}  } }",
				"() => { class foo { get hello<T extends Foo>() {}  } }",
			},
			{
				"() => { class foo { get hello(): Foo {}  } }",
				"() => { class foo { get hello(): Foo {}  } }",
			},
			{
				"() => { class foo { get [hello]() {}  } }",
				"() => { class foo { get [__VLS_Ctx.hello]() {}  } }",
			},
			{
				"() => { class foo { set hello<T extends Foo>() {}  } }",
				"() => { class foo { set hello<T extends Foo>() {}  } }",
			},
			{
				"() => { class foo { set hello(): Foo {}  } }",
				"() => { class foo { set hello(): Foo {}  } }",
			},
			{
				"() => { class foo { set [hello]() {}  } }",
				"() => { class foo { set [__VLS_Ctx.hello]() {}  } }",
			},
			{
				"() => { class foo<T extends Foo = Bar> {} }",
				"() => { class foo<T extends Foo = Bar> {} }",
			},
			{
				"() => { class foo {}; class bar extends foo {} }",
				"() => { class foo {}; class bar extends foo {} }",
			},
			{
				"() => { class foo {}; class bar extends baz {} }",
				"() => { class foo {}; class bar extends __VLS_Ctx.baz {} }",
			},
			{
				"() => { interface foo {}; class bar implements foo, baz {} }",
				"() => { interface foo {}; class bar implements foo, baz {} }",
			},
			{
				"() => { class foo { constructor<T extends Foo>() {} } }",
				"() => { class foo { constructor<T extends Foo>() {} } }",
			},
			{
				"() => { class foo { constructor(): Foo {} } }",
				"() => { class foo { constructor(): Foo {} } }",
			},
			{
				"() => { class foo { method<T extends Foo>() {} } }",
				"() => { class foo { method<T extends Foo>() {} } }",
			},
			{
				"() => { class foo { method(): Foo {} } }",
				"() => { class foo { method(): Foo {} } }",
			},
			{
				"() => { class foo { [method]() { bar } } }",
				"() => { class foo { [__VLS_Ctx.method]() { __VLS_Ctx.bar } } }",
			},
			{
				"() => { function foo(bar: Foo = hello) { bar } }",
				"() => { function foo(bar: Foo = __VLS_Ctx.hello) { bar } }",
			},
			{
				"() => { function foo<T extends Foo>() {} }",
				"() => { function foo<T extends Foo>() {} }",
			},
			{
				"() => { function foo(): Foo {} }",
				"() => { function foo(): Foo {} }",
			},
			{
				"() => { function foo(bar: string | number): asserts bar is string {} }",
				"() => { function foo(bar: string | number): asserts bar is string {} }",
			},
			{
				"<T extends Foo>() => {}",
				"<T extends Foo>() => {}",
			},
			{
				"(): Foo => {}",
				"(): Foo => {}",
			},
			{
				"function <T extends Foo>() {}",
				"function <T extends Foo>() {}",
			},
			{
				"function (): Foo {}",
				"function (): Foo {}",
			},
			{
				"foo<T>",
				"__VLS_Ctx.foo<T>",
			},
			{
				"foo as Bar",
				"__VLS_Ctx.foo as Bar",
			},
			{
				"foo<T>()",
				"__VLS_Ctx.foo<T>()",
			},
			{
				"() => { const foo: Foo = '' }",
				"() => { const foo: Foo = '' }",
			},
			{
				"() => { type foo = typeof bar }",
				"() => { type foo = typeof __VLS_Ctx.bar }",
			},
			{
				"() => { type foo = typeof bar[baz] }",
				"() => { type foo = typeof __VLS_Ctx.bar[baz] }",
			},
			{
				"() => { type foo = typeof bar<T> }",
				"() => { type foo = typeof __VLS_Ctx.bar<T> }",
			},
			{
				"() => { type foo = typeof bar.baz }",
				"() => { type foo = typeof __VLS_Ctx.bar.baz }",
			},
			{
				"() => { type foo = typeof bar['baz'] }",
				"() => { type foo = typeof __VLS_Ctx.bar['baz'] }",
			},
			{
				"() => { const foo = 1; type foo = typeof foo }",
				"() => { const foo = 1; type foo = typeof foo }",
			},
		}

		for i, c := range cases {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				base := newCodegenCtx(nil, c.sourceText, VueOptions{Version: 3_006_000})
				ctx := newTemplateCodegenCtx(&base)

				tsAst := vue_parser.ParseTsAst("(" + c.sourceText + ")")
				diagnostics := tsAst.Diagnostics()
				if strings.Contains(c.sourceText, "/*syntax errors*/") {
					if len(diagnostics) == 0 {
						t.Fatalf("expected to contain syntax errors: %v", diagnostics)
					}
				} else if len(diagnostics) > 0 {
					t.Fatalf("esyntax errors: %v", diagnostics)
				}
				expr := vue_ast.NewSimpleExpressionNode(tsAst, core.NewTextRange(0, len(c.sourceText)), 1, 1)
				ctx.mapExpressionInNonBindingPosition(expr)

				diff := cmp.Diff(c.serviceText, ctx.serviceText.String())
				if diff != "" {
					t.Fatal(diff)
				}
			})
		}
	})

	t.Run("binding position", func(t *testing.T) {
		cases := []struct {
			sourceText  string
			serviceText string
		}{
			{
				"hello",
				"hello",
			},
			{
				"[hello]",
				"[hello]",
			},
			{
				"{ hello: bar }",
				"{ hello: bar }",
			},
			{
				"{ hello: bar }",
				"{ hello: bar }",
			},
			{
				"{ [hello]: bar }",
				"{ [__VLS_Ctx.hello]: bar }",
			},
			{
				"{ hello: bar = foo }",
				"{ hello: bar = __VLS_Ctx.foo }",
			},
		}

		for i, c := range cases {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				base := newCodegenCtx(nil, c.sourceText, VueOptions{Version: 3_006_000})
				ctx := newTemplateCodegenCtx(&base)

				tsAst := vue_parser.ParseTsAst("(" + c.sourceText + ")=>{}")
				expr := vue_ast.NewSimpleExpressionNode(tsAst, core.NewTextRange(0, len(c.sourceText)), 1, 5)
				ctx.mapExpressionInBindingPosition(expr)

				diff := cmp.Diff(c.serviceText, ctx.serviceText.String())
				if diff != "" {
					t.Fatal(diff)
				}
			})
		}
	})
}
