/// <reference types="../../../../../../.reference/language-tools/packages/language-core/types/template-helpers.d.ts" />
/// <reference types="../../../../../../.reference/language-tools/packages/language-core/types/props-fallback.d.ts" />

import { ref, computed, watch, onMounted } from 'vue'
import type { Ref } from 'vue'

interface BreadcrumbItem {
  label: string
  path: string
}

type __VLS_Props = {
  readonly title: string
  readonly itemId: string
  readonly variant?: 'block' | 'curved' | 'round'
};
const props = defineProps<__VLS_Props>()

type __VLS_Emit = {
  delete: [id: string]
  rename: [id: string, newName: string]
};
const emit = defineEmits<__VLS_Emit>()

const showMenu = ref(false)
const showDeleteConfirm = ref(false)
const deleteError = ref<string | null>(null)
const deleteChildrenCount = ref(0)
const searchQuery = ref('')
const breadcrumbContainer = ref<HTMLDivElement | null>(null)

const breadcrumbs = computed<BreadcrumbItem[]>(() => {
  if (!props.title) {
    return []
  }
  return props.title.split('/').map(part => ({
    label: part.trim(),
    path: `/${part.trim().toLowerCase()}`
  }))
})

const filteredBreadcrumbs = computed(() => {
  if (!searchQuery.value) {
    return breadcrumbs.value
  }
  return breadcrumbs.value.filter(b =>
    b.label.toLowerCase().includes(searchQuery.value.toLowerCase())
  )
})

const isGlobal = computed(() => {
  return props.title === 'Global'
})

const dynamicWidth = computed(() => {
  const count = filteredBreadcrumbs.value.length
  return Math.max(50, 300 / Math.max(count, 1))
})

watch(
  () => props.itemId,
  async (newId) => {
    if (newId) {
      showMenu.value = false
      showDeleteConfirm.value = false
    }
  },
  { immediate: true }
)

onMounted(() => {
  console.log('Component mounted:', props.title)
})

function handleDelete() {
  deleteError.value = null
  emit('delete', props.itemId)
  showDeleteConfirm.value = false
  showMenu.value = false
  deleteChildrenCount.value = 0
}

function handleRenameFromMenu() {
  showMenu.value = false
  emit('rename', props.itemId, props.title)
}
// @ts-ignore
declare const { defineProps, defineSlots, defineEmits, defineExpose, defineModel, defineOptions, withDefaults, }: typeof import('vue');
type __VLS_PublicProps = __VLS_Props;
type __VLS_SetupExposed = import('vue').ShallowUnwrapRef<{
filteredBreadcrumbs: typeof filteredBreadcrumbs;
dynamicWidth: typeof dynamicWidth;
isGlobal: typeof isGlobal;
showMenu: typeof showMenu;
handleRenameFromMenu: typeof handleRenameFromMenu;
deleteChildrenCount: typeof deleteChildrenCount;
showDeleteConfirm: typeof showDeleteConfirm;
deleteError: typeof deleteError;
handleDelete: typeof handleDelete;
}>;
type __VLS_EmitProps = __VLS_EmitsToProps<__VLS_NormalizeEmits<typeof emit>>;
const __VLS_ctx = {
...{} as import('vue').ComponentPublicInstance,
...{} as { $emit: typeof emit },
...{} as { $props: typeof props & __VLS_EmitProps },
...{} as typeof props & __VLS_EmitProps,
...{} as __VLS_SetupExposed,
};
type __VLS_LocalComponents = __VLS_SetupExposed;
type __VLS_GlobalComponents = import('vue').GlobalComponents;
let __VLS_components!: __VLS_LocalComponents & __VLS_GlobalComponents;
let __VLS_intrinsics!: import('vue/jsx-runtime').JSX.IntrinsicElements;
type __VLS_LocalDirectives = __VLS_SetupExposed;
let __VLS_directives!: __VLS_LocalDirectives & import('vue').GlobalDirectives;
type __VLS_StyleScopedClasses = {}
 & { 'item-header': boolean };
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
...{ class: "item-header" },
'data-id': (__VLS_ctx.itemId),
});
/** @type {__VLS_StyleScopedClasses['item-header']} */;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
...{ class: "breadcrumbs" },
ref: "breadcrumbContainer",
});
/** @type {__VLS_StyleScopedClasses['breadcrumbs']} */;
for (const [crumb, idx] of __VLS_vFor((__VLS_ctx.filteredBreadcrumbs)!)) {
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
key: (idx),
title: (crumb.label),
...{ class: "breadcrumb-item" },
...{ style: ({ maxWidth: `${__VLS_ctx.dynamicWidth}px` }) },
});
/** @type {__VLS_StyleScopedClasses['breadcrumb-item']} */;
__VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
});
( crumb.label );
if (idx < __VLS_ctx.filteredBreadcrumbs.length - 1) {
__VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
...{ class: "separator" },
});
/** @type {__VLS_StyleScopedClasses['separator']} */;
}
// @ts-ignore
[itemId,filteredBreadcrumbs,filteredBreadcrumbs,dynamicWidth,];
}
__VLS_asFunctionalElement1(__VLS_intrinsics.h2, __VLS_intrinsics.h2)({
...{ class: "item-title" },
});
/** @type {__VLS_StyleScopedClasses['item-title']} */;
( __VLS_ctx.title );
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
...{ class: "actions" },
});
/** @type {__VLS_StyleScopedClasses['actions']} */;
if (__VLS_ctx.isGlobal) {
__VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
...{ class: "global-badge" },
});
/** @type {__VLS_StyleScopedClasses['global-badge']} */;
}
else {
__VLS_asFunctionalElement1(__VLS_intrinsics.button, __VLS_intrinsics.button)({
...{ onClick: (...[$event]) => {
if (!!(__VLS_ctx.isGlobal)) return;
__VLS_ctx.showMenu = !__VLS_ctx.showMenu;
// @ts-ignore
[title,isGlobal,showMenu,showMenu,];
}},
...{ class: "menu-btn" },
});
/** @type {__VLS_StyleScopedClasses['menu-btn']} */;
}
var __VLS_0 = {
};
if (__VLS_ctx.showMenu && !__VLS_ctx.isGlobal) {
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
...{ onClick: () => {}},
...{ class: "menu-popover" },
});
/** @type {__VLS_StyleScopedClasses['menu-popover']} */;
__VLS_asFunctionalElement1(__VLS_intrinsics.button, __VLS_intrinsics.button)({
...{ onClick: (__VLS_ctx.handleRenameFromMenu)},
});
__VLS_asFunctionalElement1(__VLS_intrinsics.button, __VLS_intrinsics.button)({
...{ onClick: (...[$event]) => {
if (!(__VLS_ctx.showMenu && !__VLS_ctx.isGlobal)) return;

          __VLS_ctx.deleteChildrenCount = 0;
          __VLS_ctx.showDeleteConfirm = true;
        ;
// @ts-ignore
[isGlobal,showMenu,handleRenameFromMenu,deleteChildrenCount,showDeleteConfirm,];
}},
});
}
if (__VLS_ctx.showDeleteConfirm) {
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
...{ class: "delete-dialog" },
});
/** @type {__VLS_StyleScopedClasses['delete-dialog']} */;
__VLS_asFunctionalElement1(__VLS_intrinsics.p, __VLS_intrinsics.p)({
});
( __VLS_ctx.title );
if (__VLS_ctx.deleteChildrenCount > 0) {
__VLS_asFunctionalElement1(__VLS_intrinsics.p, __VLS_intrinsics.p)({
});
( __VLS_ctx.deleteChildrenCount );
}
if (__VLS_ctx.deleteError) {
__VLS_asFunctionalElement1(__VLS_intrinsics.p, __VLS_intrinsics.p)({
...{ class: "error" },
});
/** @type {__VLS_StyleScopedClasses['error']} */;
( __VLS_ctx.deleteError );
}
__VLS_asFunctionalElement1(__VLS_intrinsics.button, __VLS_intrinsics.button)({
...{ onClick: (__VLS_ctx.handleDelete)},
});
__VLS_asFunctionalElement1(__VLS_intrinsics.button, __VLS_intrinsics.button)({
...{ onClick: (...[$event]) => {
if (!(__VLS_ctx.showDeleteConfirm)) return;

          __VLS_ctx.showDeleteConfirm = false;
          __VLS_ctx.showMenu = false;
          __VLS_ctx.deleteChildrenCount = 0;
        ;
// @ts-ignore
[title,showMenu,deleteChildrenCount,deleteChildrenCount,deleteChildrenCount,showDeleteConfirm,showDeleteConfirm,deleteError,deleteError,handleDelete,];
}},
});
}
// @ts-ignore
var __VLS_1 = __VLS_0, ;
type __VLS_Slots = {}
& { 'extra-actions'?: (props: typeof __VLS_1) => any };
type __VLS_TemplateRefs = {}
& { breadcrumbContainer: __VLS_Elements['div'] };
type __VLS_RootEl = 
| __VLS_Elements['div'];
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
__typeEmits: {} as __VLS_Emit,
__typeProps: {} as __VLS_PublicProps,
});
const __VLS_export = {} as __VLS_WithSlots<typeof __VLS_base, __VLS_Slots>;
export default {} as typeof __VLS_export;
type __VLS_WithSlots<T, S> = T & {
	new(): {
		$slots: S;
	}
};

