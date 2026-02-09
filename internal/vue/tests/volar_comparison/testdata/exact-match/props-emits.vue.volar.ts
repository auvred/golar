/// <reference types="../../../../../../.reference/language-tools/packages/language-core/types/template-helpers.d.ts" />
/// <reference types="../../../../../../.reference/language-tools/packages/language-core/types/props-fallback.d.ts" />

type __VLS_Props = {
	foo: string;
};
const props = defineProps<__VLS_Props>();

type __VLS_Emit = {
	bar: [value: number];
};
const emit = defineEmits<__VLS_Emit>();
// @ts-ignore
declare const { defineProps, defineSlots, defineEmits, defineExpose, defineModel, defineOptions, withDefaults, }: typeof import('vue');
type __VLS_PublicProps = __VLS_Props;
type __VLS_EmitProps = __VLS_EmitsToProps<__VLS_NormalizeEmits<typeof emit>>;
const __VLS_ctx = {
...{} as import('vue').ComponentPublicInstance,
...{} as { $emit: typeof emit },
...{} as { $props: typeof props & __VLS_EmitProps },
...{} as typeof props & __VLS_EmitProps,
};
type __VLS_LocalComponents = {};
type __VLS_GlobalComponents = import('vue').GlobalComponents;
let __VLS_components!: __VLS_LocalComponents & __VLS_GlobalComponents;
let __VLS_intrinsics!: import('vue/jsx-runtime').JSX.IntrinsicElements;
type __VLS_LocalDirectives = {};
let __VLS_directives!: __VLS_LocalDirectives & import('vue').GlobalDirectives;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
});
( __VLS_ctx.foo );
type __VLS_RootEl = 
| __VLS_Elements['div'];
// @ts-ignore
[foo,];
const __VLS_export = (await import('vue')).defineComponent({
__typeEmits: {} as __VLS_Emit,
__typeProps: {} as __VLS_PublicProps,
});
export default {} as typeof __VLS_export;

