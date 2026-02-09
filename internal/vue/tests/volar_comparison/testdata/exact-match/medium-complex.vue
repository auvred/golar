<script lang="ts" setup>
import { ref, computed, watch, onMounted } from 'vue'
import type { Ref } from 'vue'

interface BreadcrumbItem {
  label: string
  path: string
}

const props = defineProps<{
  readonly title: string
  readonly itemId: string
  readonly variant?: 'block' | 'curved' | 'round'
}>()

const emit = defineEmits<{
  delete: [id: string]
  rename: [id: string, newName: string]
}>()

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
</script>

<template>
  <div class="item-header" :data-id="itemId">
    <div class="breadcrumbs" ref="breadcrumbContainer">
      <div
        v-for="(crumb, idx) in filteredBreadcrumbs"
        :key="idx"
        :title="crumb.label"
        class="breadcrumb-item"
        :style="{ maxWidth: `${dynamicWidth}px` }"
      >
        <span>{{ crumb.label }}</span>
        <span v-if="idx < filteredBreadcrumbs.length - 1" class="separator">/</span>
      </div>
    </div>

    <h2 class="item-title">{{ title }}</h2>

    <div class="actions">
      <template v-if="isGlobal">
        <span class="global-badge">Global</span>
      </template>
      <template v-else>
        <button
          class="menu-btn"
          @click="showMenu = !showMenu"
        >
          Menu
        </button>
      </template>
      <slot name="extra-actions" />
    </div>

    <div
      v-if="showMenu && !isGlobal"
      class="menu-popover"
      @click.stop
    >
      <button @click="handleRenameFromMenu">Rename</button>
      <button
        @click="
          deleteChildrenCount = 0;
          showDeleteConfirm = true;
        "
      >
        Delete
      </button>
    </div>

    <div v-if="showDeleteConfirm" class="delete-dialog">
      <p>Are you sure you want to delete "{{ title }}"?</p>
      <p v-if="deleteChildrenCount > 0">
        This will also delete {{ deleteChildrenCount }} child items.
      </p>
      <p v-if="deleteError" class="error">{{ deleteError }}</p>
      <button @click="handleDelete">Confirm</button>
      <button
        @click="
          showDeleteConfirm = false;
          showMenu = false;
          deleteChildrenCount = 0;
        "
      >
        Cancel
      </button>
    </div>
  </div>
</template>

<style scoped>
.item-header {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
