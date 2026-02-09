/// <reference types="../../../../../../.reference/language-tools/packages/language-core/types/template-helpers.d.ts" />
/// <reference types="../../../../../../.reference/language-tools/packages/language-core/types/props-fallback.d.ts" />

import { exactType } from '../shared';

let Foo: new () => { $props: { foo: (_: string) => void; }; };
let Bar: new () => { $props: { bar: (_: number) => void; }; };
// @ts-ignore
declare const { defineProps, defineSlots, defineEmits, defineExpose, defineModel, defineOptions, withDefaults, }: typeof import('vue');
type __VLS_SetupExposed = import('vue').ShallowUnwrapRef<{
Foo: typeof Foo;
exactType: typeof exactType;
Bar: typeof Bar;
}>;
const __VLS_ctx = {
...{} as import('vue').ComponentPublicInstance,
...{} as __VLS_SetupExposed,
};
type __VLS_LocalComponents = __VLS_SetupExposed;
type __VLS_GlobalComponents = import('vue').GlobalComponents;
let __VLS_components!: __VLS_LocalComponents & __VLS_GlobalComponents;
let __VLS_intrinsics!: import('vue/jsx-runtime').JSX.IntrinsicElements;
type __VLS_LocalDirectives = __VLS_SetupExposed;
let __VLS_directives!: __VLS_LocalDirectives & import('vue').GlobalDirectives;
const __VLS_0 = (__VLS_ctx.Foo);
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
foo: (e => __VLS_ctx.exactType(e, {} as string)),
}));
const __VLS_2 = __VLS_1({
foo: (e => __VLS_ctx.exactType(e, {} as string)),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
const __VLS_5 = (Math.random() > 0.5 ? __VLS_ctx.Foo : __VLS_ctx.Bar);
// @ts-ignore
const __VLS_6 = __VLS_asFunctionalComponent1(__VLS_5, new __VLS_5({
foo: (e => __VLS_ctx.exactType(e, {} as string)),
}));
const __VLS_7 = __VLS_6({
foo: (e => __VLS_ctx.exactType(e, {} as string)),
}, ...__VLS_functionalComponentArgsRest(__VLS_6));
const __VLS_10 = (Math.random() > 0.5 ? __VLS_ctx.Foo : __VLS_ctx.Bar);
// @ts-ignore
const __VLS_11 = __VLS_asFunctionalComponent1(__VLS_10, new __VLS_10({
bar: (e => __VLS_ctx.exactType(e, {} as number)),
}));
const __VLS_12 = __VLS_11({
bar: (e => __VLS_ctx.exactType(e, {} as number)),
}, ...__VLS_functionalComponentArgsRest(__VLS_11));
// @ts-ignore
[Foo,Foo,Foo,exactType,exactType,exactType,Bar,Bar,];
const __VLS_export = (await import('vue')).defineComponent({
});
export default {} as typeof __VLS_export;

