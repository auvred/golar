/// <reference types="../../../../../../.reference/language-tools/packages/language-core/types/template-helpers.d.ts" />
/// <reference types="../../../../../../.reference/language-tools/packages/language-core/types/props-fallback.d.ts" />

const type = ref<'a' | 'b' | 'c'>('a')
const showExtra = ref(false)
// @ts-ignore
declare const { defineProps, defineSlots, defineEmits, defineExpose, defineModel, defineOptions, withDefaults, }: typeof import('vue');
type __VLS_SetupExposed = import('vue').ShallowUnwrapRef<{
type: typeof type;
showExtra: typeof showExtra;
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
if (__VLS_ctx.type === 'a') {
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
});
}
else if (__VLS_ctx.type === 'b') {
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
});
}
else {
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
});
( __VLS_ctx.type );
}
if (__VLS_ctx.showExtra) {
__VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
});
}
// @ts-ignore
[type,type,type,showExtra,];
const __VLS_export = (await import('vue')).defineComponent({
});
export default {} as typeof __VLS_export;

