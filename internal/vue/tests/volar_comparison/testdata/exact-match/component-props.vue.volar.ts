/// <reference types="../../../../../../.reference/language-tools/packages/language-core/types/template-helpers.d.ts" />
/// <reference types="../../../../../../.reference/language-tools/packages/language-core/types/props-fallback.d.ts" />

import ChildComp from './child.vue'

const name = ref('hello')
// @ts-ignore
declare const { defineProps, defineSlots, defineEmits, defineExpose, defineModel, defineOptions, withDefaults, }: typeof import('vue');
type __VLS_SetupExposed = import('vue').ShallowUnwrapRef<{
name: typeof name;
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
const __VLS_0 = ChildComp;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
foo: (__VLS_ctx.name),
bar: "static",
}));
const __VLS_2 = __VLS_1({
foo: (__VLS_ctx.name),
bar: "static",
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
const __VLS_5 = ChildComp;
// @ts-ignore
const __VLS_6 = __VLS_asFunctionalComponent1(__VLS_5, new __VLS_5({
foo: "missing-bar",
}));
const __VLS_7 = __VLS_6({
foo: "missing-bar",
}, ...__VLS_functionalComponentArgsRest(__VLS_6));
// @ts-ignore
[name,];
const __VLS_export = (await import('vue')).defineComponent({
});
export default {} as typeof __VLS_export;

