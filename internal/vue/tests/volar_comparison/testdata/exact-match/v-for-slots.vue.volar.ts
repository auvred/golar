/// <reference types="../../../../../../.reference/language-tools/packages/language-core/types/template-helpers.d.ts" />
/// <reference types="../../../../../../.reference/language-tools/packages/language-core/types/props-fallback.d.ts" />

import { ref } from 'vue'

const items = ref([
  { id: 1, name: 'foo' },
  { id: 2, name: 'bar' },
])
// @ts-ignore
declare const { defineProps, defineSlots, defineEmits, defineExpose, defineModel, defineOptions, withDefaults, }: typeof import('vue');
type __VLS_SetupExposed = import('vue').ShallowUnwrapRef<{
items: typeof items;
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
__VLS_asFunctionalElement1(__VLS_intrinsics.ul, __VLS_intrinsics.ul)({
});
for (const [{ id, name }] of __VLS_vFor((__VLS_ctx.items)!)) {
__VLS_asFunctionalElement1(__VLS_intrinsics.li, __VLS_intrinsics.li)({
key: (id),
});
( name );
( id );
// @ts-ignore
[items,];
}
var __VLS_0 = {
count: (__VLS_ctx.items.length),
};
// @ts-ignore
var __VLS_1 = __VLS_0, ;
type __VLS_Slots = {}
& { footer?: (props: typeof __VLS_1) => any };
// @ts-ignore
[items,];
const __VLS_base = (await import('vue')).defineComponent({
});
const __VLS_export = {} as __VLS_WithSlots<typeof __VLS_base, __VLS_Slots>;
export default {} as typeof __VLS_export;
type __VLS_WithSlots<T, S> = T & {
	new(): {
		$slots: S;
	}
};

