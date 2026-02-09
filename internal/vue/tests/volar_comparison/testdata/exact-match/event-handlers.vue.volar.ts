/// <reference types="../../../../../../.reference/language-tools/packages/language-core/types/template-helpers.d.ts" />
/// <reference types="../../../../../../.reference/language-tools/packages/language-core/types/props-fallback.d.ts" />

import { ref } from 'vue'

const count = ref(0)
const showMenu = ref(false)
const showConfirm = ref(false)

function handleClick() {
  console.log('clicked')
}
// @ts-ignore
declare const { defineProps, defineSlots, defineEmits, defineExpose, defineModel, defineOptions, withDefaults, }: typeof import('vue');
type __VLS_SetupExposed = import('vue').ShallowUnwrapRef<{
handleClick: typeof handleClick;
count: typeof count;
showConfirm: typeof showConfirm;
showMenu: typeof showMenu;
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
__VLS_asFunctionalElement1(__VLS_intrinsics.button, __VLS_intrinsics.button)({
...{ onClick: (__VLS_ctx.handleClick)},
});
__VLS_asFunctionalElement1(__VLS_intrinsics.button, __VLS_intrinsics.button)({
...{ onClick: (...[$event]) => {
__VLS_ctx.count++;
// @ts-ignore
[handleClick,count,];
}},
});
__VLS_asFunctionalElement1(__VLS_intrinsics.button, __VLS_intrinsics.button)({
...{ onClick: (...[$event]) => {

    __VLS_ctx.showConfirm = false;
    __VLS_ctx.showMenu = false;
    __VLS_ctx.count = 0;
  ;
// @ts-ignore
[count,showConfirm,showMenu,];
}},
});
__VLS_asFunctionalElement1(__VLS_intrinsics.button, __VLS_intrinsics.button)({
...{ onClick: (__VLS_ctx.handleClick)},
});
// @ts-ignore
[handleClick,];
const __VLS_export = (await import('vue')).defineComponent({
});
export default {} as typeof __VLS_export;

