package vue_codegen

import (
	"strconv"
	"strings"

	"github.com/auvred/golar/internal/mapping"
	"github.com/auvred/golar/internal/vue/ast"
	"github.com/auvred/golar/internal/vue/diagnostics"
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/core"
	"github.com/microsoft/typescript-go/shim/diagnostics"
)

// TODO: relative to cwd or executable location, so vue import works
// const GlobalTypesPath = utils.GolarVirtualScheme + "vue-global-types.d.ts"
const GlobalTypesPath = "/" + "vue-global-types.d.ts"
const globalTypesReference = `/// <reference types="` + GlobalTypesPath + `" />
`

// Iterable requires es2015.iterable
const GlobalTypes = `/// <reference lib="es2015" />
export {}

declare global {
	function __VLS_vFor<T>(source: T): T extends number
		? [number, number]
		: T extends string
			? [string, number]
			: T extends any[]
				? [T[number], number]
				: T extends Iterable<infer V>
					? [V, number]
					: [T[keyof T], ` + "`${keyof T & string}`" + `, number]

	type __VLS_FunctionalComponent<T> = (props: (T extends { $props: infer Props } ? Props : {}), ctx?: any) => {
		__ctx?: {
			attrs?: any;
			slots?: T extends { $slots: infer Slots } ? Slots : Record<string, any>;
			emit?: T extends { $emit: infer Emit } ? Emit : {};
			props?: typeof props;
			expose?: (exposed: T) => void;
		};
	};

	function __VLS_AsFunctionalComponent<T, K = T extends new (...args: any) => any ? InstanceType<T> : unknown>(t: T, instance?: K):
		T extends new (...args: any) => any
			? __VLS_FunctionalComponent<K>
			: T extends () => any
				? (props: {}, ctx?: any) => ReturnType<T>
				: T extends (...args: any) => any
					? T
					: __VLS_FunctionalComponent<{}>;

	// TODO: pre 3.5?
	type __VLS_GlobalComponents = import('vue').GlobalComponents

	type __VLS_ExtractComponentType<N0 extends string, LocalComponents, Self, N1 extends string, N2 extends string = N1, N3 extends string = N1> =
		N1 extends keyof LocalComponents
			? { [K in N0]: LocalComponents[N1] }
			: N2 extends keyof LocalComponents
				? { [K in N0]: LocalComponents[N2] }
				: N3 extends keyof LocalComponents
					? { [K in N0]: LocalComponents[N3] }
					: Self extends object
						? { [K in N0]: Self }
						: N1 extends keyof __VLS_GlobalComponents
							? { [K in N0]: __VLS_GlobalComponents[N1] }
							: N2 extends keyof __VLS_GlobalComponents
								? { [K in N0]: __VLS_GlobalComponents[N2] }
								: N3 extends keyof __VLS_GlobalComponents
									? { [K in N0]: __VLS_GlobalComponents[N3] }
									: {};

	type __VLS_IsAny<T> = 0 extends 1 & T ? true : false;

	type __VLS_PickNotAny<A, B> = __VLS_IsAny<A> extends true ? B : A;

	type __VLS_SpreadMerge<A, B> = Omit<A, keyof B> & B;

	type __VLS_PrettifyGlobal<T> = (T extends any ? { [K in keyof T]: T[K] } : { [K in keyof T as K]: T[K] }) & {};

	type __VLS_UnionToIntersection<U> = (U extends unknown ? (arg: U) => unknown : never) extends ((arg: infer P) => unknown)
		? P
		: never;

	type __VLS_OverloadUnionInner<T, U = unknown> = U & T extends (...args: infer A) => infer R
		? U extends T
			? never
			: __VLS_OverloadUnionInner<T, Pick<T, keyof T> & U & ((...args: A) => R)> | ((...args: A) => R)
		: never;
	type __VLS_OverloadUnion<T> = Exclude<
		__VLS_OverloadUnionInner<(() => never) & T>,
		T extends () => never ? never : () => never
	>;

	type __VLS_ConstructorOverloads<T> = __VLS_OverloadUnion<T> extends infer F
		? F extends (event: infer E, ...args: infer A) => any
			? { [K in E & string]: (...args: A) => void }
			: never
		: never;

	type __VLS_IsFunction<T, K> = K extends keyof T
		? __VLS_IsAny<T[K]> extends false
			? unknown extends T[K]
				? false
				: true
			: false
		: false;

	type __VLS_NormalizeComponentEvent<
		Props,
		Emits,
		onEvent extends keyof Props,
		Event extends keyof Emits,
		CamelizedEvent extends keyof Emits,
	> = __VLS_IsFunction<Props, onEvent> extends true
		? Props
		: __VLS_IsFunction<Emits, Event> extends true
			? { [K in onEvent]?: Emits[Event] }
			: __VLS_IsFunction<Emits, CamelizedEvent> extends true
				? { [K in onEvent]?: Emits[CamelizedEvent] }
				: Props;

	type __VLS_FunctionalComponentProps<T, K> = '__ctx' extends keyof __VLS_PickNotAny<K, {}>
		? K extends { __ctx?: { props?: infer P } }
			? NonNullable<P>
			: never
		: T extends (props: infer P, ...args: any) => any
			? P
			: {};

	type __VLS_FunctionalComponentCtx<T, K> = __VLS_PickNotAny<
		'__ctx' extends keyof __VLS_PickNotAny<K, {}>
			? K extends { __ctx?: infer Ctx }
				? NonNullable<Ctx>
				: never
			: any,
		T extends (props: any, ctx: infer Ctx) => any
			? Ctx
			: any
	>;

	type __VLS_NormalizeEmits<T> = __VLS_PrettifyGlobal<
		__VLS_UnionToIntersection<
			__VLS_ConstructorOverloads<T> & {
				[K in keyof T]: T[K] extends any[] ? { (...args: T[K]): void } : never
			}
		>
	>;

	type __VLS_ShortEmitsToObject<E> = E extends Record<string, any[]>
		? { [K in keyof E]: (...args: E[K]) => any }
		: E;

	type __VLS_ResolveEmits<
		Comp,
		Emits,
		TypeEmits = Comp extends { __typeEmits?: infer T }
			? unknown extends T
				? {}
				: __VLS_ShortEmitsToObject<T>
			: {},
		NormalizedEmits = __VLS_NormalizeEmits<Emits> extends infer E
			? string extends keyof E
				? {}
				: E
			: never,
	> = __VLS_SpreadMerge<NormalizedEmits, TypeEmits>;

	type __VLS_WithSlots<T, S> = T extends abstract new (...args: any) => any
		? T & {
				new(...args: ConstructorParameters<T>): {
					$slots: S;
				}
			}
		: any;

	function __VLS_vSlot<S, D extends S>(slot: S, decl?: D): D extends (...args: infer P) => any ? P : any[];

	type __VLS_TypePropsToOption<T> = {
		[K in keyof T]-?: {} extends Pick<T, K>
			? { type: import('vue').PropType<Required<T>[K]> }
			: { type: import('vue').PropType<T[K]>, required: true }
	};

	type __VLS_EmitsToProps<T> = __VLS_PrettifyGlobal<
		{
			[K in string & keyof T as ` + "`" + `on${Capitalize<K>}` + "`" + `]?: (
				...args: T[K] extends (...args: infer P) => any ? P : T[K] extends null ? any[] : never
			) => any;
		}
	>;

	function __VLS_DefineExpose<Exposed extends Record<string, any> = Record<string, any>>(exposed?: Exposed): Exposed;

	function __VLS_AsFunctionalElement<T>(tag: T, endTag?: T): (attrs: T) => void;
}

// pre 3.5 shims
// https://github.com/vuejs/core/pull/3399
declare module 'vue' {
	export interface GlobalComponents {}
	export interface GlobalDirectives {}
}
`

func Codegen(sourceText string, root *vue_ast.RootNode, options VueOptions) (string, []mapping.Mapping, []mapping.IgnoreDirectiveMapping, []mapping.ExpectErrorDirectiveMapping, []*ast.Diagnostic) {
	ctx := newCodegenCtx(root, sourceText, options)
	ctx.serviceText.WriteString(globalTypesReference)
	ctx.serviceText.WriteString("declare const __VLS_Intrinsics: ")
	if options.Version.hasJsxRuntimeTypes() {
		ctx.serviceText.WriteString("import('vue/jsx-runtime').JSX.IntrinsicElements\n")
	} else {
		ctx.serviceText.WriteString("globalThis.JSX.IntrinsicElements")
	}

	var scriptEl *vue_ast.ElementNode
	var scriptSetupEl *vue_ast.ElementNode
	var templateEl *vue_ast.ElementNode

RootChild:
	for _, child := range root.Children {
		if child.Kind != vue_ast.KindElement {
			continue
		}

		el := child.AsElement()

		if el.Tag == "script" {
			for _, prop := range el.Props {
				if prop.Kind == vue_ast.KindAttribute {
					attr := prop.AsAttribute()
					if attr.Name == "setup" {
						if scriptSetupEl != nil {
							ctx.reportDiagnostic(el.Loc.WithEnd(el.InnerLoc.Pos()), vue_diagnostics.Single_file_component_can_contain_only_one_script_setup_element)
						} else {
							scriptSetupEl = el
						}
						continue RootChild
					}
				}
			}

			if scriptEl != nil {
				ctx.reportDiagnostic(el.Loc.WithEnd(el.InnerLoc.Pos()), vue_diagnostics.Single_file_component_can_contain_only_one_script_element)
			} else {
				scriptEl = el
			}
			continue RootChild
		}

		if el.Tag == "template" {
			if templateEl != nil {
				ctx.reportDiagnostic(el.Loc.WithEnd(el.InnerLoc.Pos()), vue_diagnostics.Single_file_component_can_contain_only_one_template_element)
				continue
			}
			templateEl = el
		}
	}

	// https://github.com/volarjs/volar.js/discussions/188
	lineStart := 0
	for {
		idx := strings.IndexByte(sourceText[lineStart:], '\n')
		if idx == -1 {
			for range len(sourceText) - lineStart {
				ctx.serviceText.WriteByte(' ')
			}
			break
		}
		idx += lineStart
		for range idx - lineStart {
			ctx.serviceText.WriteByte(' ')
		}
		ctx.serviceText.WriteByte('\n')
		lineStart = idx + 1
	}

	// {
	// 	c := newCodegenCtx(root, sourceText)
	// 	generateScript(&c, scriptSetupEl, scriptEl, templateEl)
	// 	newMappingsStart := len(ctx.mappings)
	// 	ctx.mappings = append(ctx.mappings, c.mappings...)
	// 	for i := newMappingsStart; i < len(ctx.mappings); i++ {
	// 		ctx.mappings[i].ServiceOffset += ctx.serviceText.Len()
	// 		// TODO: range mappings?
	// 	}
	// 	ctx.serviceText.Write([]byte(c.serviceText.String()))
	// 	ctx.diagnostics = append(ctx.diagnostics, c.diagnostics...)
	// }
	generateScript(&ctx, scriptSetupEl, scriptEl, templateEl)

	return ctx.serviceText.String(), ctx.mappings, ctx.ignoreDirectives, ctx.expectErrorDirectives, ctx.diagnostics
}

type codegenCtx struct {
	ast                     *vue_ast.RootNode
	sourceText              string
	serviceText             strings.Builder
	mappings                []mapping.Mapping
	ignoreDirectives        []mapping.IgnoreDirectiveMapping
	expectErrorDirectives   []mapping.ExpectErrorDirectiveMapping
	diagnostics             []*ast.Diagnostic
	internalVariableCounter int
	options                 VueOptions
}

type VueVersion int
type VueOptions struct {
	// major * 1_000_000 + minor * 1_000 + patch
	Version VueVersion
}

func NewVueVersionFromSemver(major, minor, patch int) VueVersion {
	return VueVersion(major*1_000_000 + minor*1_000 + patch)
}

// https://github.com/vuejs/core/pull/10801
func (v VueVersion) supportsTypeProps() bool {
	return v >= NewVueVersionFromSemver(3, 5, 0)
}
func (v VueVersion) supportsTypeEmits() bool {
	return v.supportsTypeProps()
}

func (v VueVersion) supportsDefineSlots() bool {
	return v >= NewVueVersionFromSemver(3, 3, 0)
}

func (v VueVersion) supportsDefineModel() bool {
	return v >= NewVueVersionFromSemver(3, 4, 0)
}

// https://github.com/vuejs/core/pull/11699
func (v VueVersion) modelRefHasGetterAndSetter() bool {
	return v >= NewVueVersionFromSemver(3, 5, 0)
}

func (v VueVersion) hasPublicPropsType() bool {
	return v >= NewVueVersionFromSemver(3, 4, 0)
}

func (v VueVersion) hasJsxRuntimeTypes() bool {
	return v >= NewVueVersionFromSemver(3, 3, 0)
}

func newCodegenCtx(root *vue_ast.RootNode, sourceText string, options VueOptions) codegenCtx {
	return codegenCtx{
		ast:         root,
		sourceText:  sourceText,
		serviceText: strings.Builder{},
		mappings:    []mapping.Mapping{},
		diagnostics: []*ast.Diagnostic{},
		options:     options,
	}
}

func (c *codegenCtx) reportDiagnostic(loc core.TextRange, message *diagnostics.Message, args ...any) {
	c.diagnostics = append(c.diagnostics, ast.NewDiagnostic(nil, loc, message, args...))
}

func (c *codegenCtx) mapText(from, to int) {
	serviceOffset := c.serviceText.Len()
	c.serviceText.WriteString(c.sourceText[from:to])
	c.mappings = append(c.mappings, mapping.Mapping{
		SourceOffset:  uint32(from),
		ServiceOffset: uint32(serviceOffset),
		SourceLength:  uint32(to - from),
	})
}

func (c *codegenCtx) mapRange(sourceStart, sourceEnd, serviceStart, serviceEnd int) {
	c.mappings = append(c.mappings, mapping.Mapping{
		SourceOffset:  uint32(sourceStart),
		ServiceOffset: uint32(serviceStart),
		SourceLength:  uint32(0),
	}, mapping.Mapping{
		SourceOffset:  uint32(sourceEnd),
		ServiceOffset: uint32(serviceEnd),
		SourceLength:  uint32(0),
	})
}

func (c *codegenCtx) mapIgnoreDirective(serviceStart, serviceEnd int) {
	c.ignoreDirectives = append(c.ignoreDirectives, mapping.IgnoreDirectiveMapping{
		ServiceOffset: uint32(serviceStart),
		ServiceLength: uint32(serviceEnd - serviceStart),
	})
}

func (c *codegenCtx) mapExpectErrorDirective(sourceStart, sourceEnd, serviceStart, serviceEnd int) {
	c.expectErrorDirectives = append(c.expectErrorDirectives, mapping.ExpectErrorDirectiveMapping{
		SourceOffset:  uint32(sourceStart),
		ServiceOffset: uint32(serviceStart),
		SourceLength:  uint32(sourceEnd - sourceStart),
		ServiceLength: uint32(serviceEnd - serviceStart),
	})
}

func (c *codegenCtx) newInternalVariable() string {
	c.internalVariableCounter++
	// TODO: maybe something more performant?
	return "__VLS_Var_" + strconv.Itoa(c.internalVariableCounter)
}
