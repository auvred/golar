<script setup lang="ts">
import { logger } from "@nonfx/logger";

// Page meta
definePageMeta({
	title: "Manage Codegen Tests"
});

// State
const isLoading = ref(false);
const codegenTests = ref<Array<CodegenTest>>([]);
const selectedIds = ref<Set<string>>(new Set());
const totalTests = ref(0);
const showDeleteDialog = ref(false);
const isDeleting = ref(false);
const testToDelete = ref<string | null>(null);

// Code viewer state
const showCodeViewer = ref(false);
const selectedTest = ref<CodegenTestDetails | null>(null);
const selectedCodeTab = ref<"code" | "passingCode" | "failingCode">("code");

// Fetch codegen tests
async function fetchCodegenTests() {
	isLoading.value = true;
	try {
		const response: CodegenResponse = await $fetch("/api/admin/codegen", {
			method: "GET"
		});

		codegenTests.value = response.tests;
		totalTests.value = response.total;
		selectedIds.value.clear();
	} catch (error) {
		logger.error("Failed to fetch codegen tests:", error);
	} finally {
		isLoading.value = false;
	}
}

// View code for a test
async function viewCode(test: CodegenTest) {
	try {
		const response: CodegenDetailsResponse = await $fetch(`/api/admin/codegen/${test.id}`, {
			method: "GET"
		});
		selectedTest.value = response.test;
		selectedCodeTab.value = "code";
		showCodeViewer.value = true;
	} catch (error) {
		logger.error(`Failed to fetch details for test ${test.id}:`, error);
	}
}

// Delete single test
function deleteTest(id: string) {
	testToDelete.value = id;
	showDeleteDialog.value = true;
}

// Delete selected tests
function deleteSelectedTests() {
	if (selectedIds.value.size === 0) {
		return;
	}
	testToDelete.value = null;
	showDeleteDialog.value = true;
}

// Confirm deletion
async function confirmDelete() {
	isDeleting.value = true;
	try {
		const idsToDelete = testToDelete.value ? [testToDelete.value] : Array.from(selectedIds.value);

		await $fetch("/api/admin/codegen", {
			method: "DELETE",
			body: testToDelete.value ? { id: testToDelete.value } : { ids: idsToDelete }
		});

		// Remove from local state
		codegenTests.value = codegenTests.value.filter(test => !idsToDelete.includes(test.id));
		totalTests.value = codegenTests.value.length;

		// Clear selections
		idsToDelete.forEach(id => {
			selectedIds.value.delete(id);
		});

		if (showCodeViewer.value && selectedTest.value && idsToDelete.includes(selectedTest.value.id)) {
			showCodeViewer.value = false;
			selectedTest.value = null;
		}

		showDeleteDialog.value = false;
		testToDelete.value = null;
	} catch (error) {
		logger.error("Failed to delete test(s):", error);
	} finally {
		isDeleting.value = false;
	}
}

// Toggle selection
function toggleSelection(id: string) {
	if (selectedIds.value.has(id)) {
		selectedIds.value.delete(id);
	} else {
		selectedIds.value.add(id);
	}
}

// Toggle document selection
function toggleDocumentSelection(docName: string) {
	const grouped = groupedTests();
	const docTests = grouped.find(([name]) => name === docName)?.[1] ?? [];
	const testIds = docTests.map(t => t.id);

	const allSelected = testIds.every(id => selectedIds.value.has(id));

	if (allSelected) {
		testIds.forEach(id => selectedIds.value.delete(id));
	} else {
		testIds.forEach(id => selectedIds.value.add(id));
	}
}

// Check if all tests in document are selected
function isDocumentFullySelected(docName: string): boolean {
	const grouped = groupedTests();
	const docTests = grouped.find(([name]) => name === docName)?.[1] ?? [];
	const testIds = docTests.map(t => t.id);

	return testIds.length > 0 && testIds.every(id => selectedIds.value.has(id));
}

// Check if some tests in document are selected
function isDocumentPartiallySelected(docName: string): boolean {
	const grouped = groupedTests();
	const docTests = grouped.find(([name]) => name === docName)?.[1] ?? [];
	const testIds = docTests.map(t => t.id);

	const selectedCount = testIds.filter(id => selectedIds.value.has(id)).length;
	return selectedCount > 0 && selectedCount < testIds.length;
}

// Select all tests
function toggleSelectAll() {
	if (selectedIds.value.size === codegenTests.value.length) {
		selectedIds.value.clear();
	} else {
		selectedIds.value = new Set(codegenTests.value.map(test => test.id));
	}
}

// Group tests by document
function groupedTests() {
	const groups = new Map<string, Array<CodegenTest>>();

	for (const test of codegenTests.value) {
		const docKey = test.docName ?? "No Document";
		if (!groups.has(docKey)) {
			groups.set(docKey, []);
		}
		groups.get(docKey)!.push(test);
	}

	return Array.from(groups.entries());
}

// Format date
function formatDate(dateString: string): string {
	return new Date(dateString).toLocaleString();
}

// Computed
const allSelected = computed(
	() => codegenTests.value.length > 0 && selectedIds.value.size === codegenTests.value.length
);

const someSelected = computed(
	() => selectedIds.value.size > 0 && selectedIds.value.size < codegenTests.value.length
);

const currentCodeContent = computed(() => {
	if (!selectedTest.value) {
		return "";
	}
	return selectedTest.value.generated_code[selectedCodeTab.value];
});

const deleteDialogMessage = computed(() => {
	if (testToDelete.value) {
		return "Are you sure you want to delete this codegen test? This action cannot be undone.";
	}
	return `Are you sure you want to delete ${selectedIds.value.size} codegen tests? This action cannot be undone.`;
});

// Initial load
onMounted(() => {
	fetchCodegenTests();
});

// Types
type CodegenTest = {
	id: string;
	type: string;
	ruleTitle: string;
	ruleDescription: string;
	cloudProvider: string | null;
	docId: string | null;
	statementId: string | null;
	createdAt: string;
	updatedAt: string;
	docName: string | null;
	docVersion: string | null;
	docPublisher: string | null;
	docState: string | null;
	statementTitle: string | null;
	statementDisplayId: string | null;
};

type CodegenTestDetails = CodegenTest & {
	generated_code: {
		code: string;
		passingCode: string;
		failingCode: string;
	};
};

type CodegenResponse = {
	success: boolean;
	tests: Array<CodegenTest>;
	total: number;
};

type CodegenDetailsResponse = {
	success: boolean;
	test: CodegenTestDetails;
};
</script>

<template>
	<div class="flex h-screen w-screen overflow-hidden bg-gray-50 dark:bg-gray-900">
		<!-- Main content -->
		<div class="flex flex-1 flex-col overflow-hidden">
			<!-- Header -->
			<div
				class="sticky top-0 z-40 border-b border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800"
			>
				<div class="px-4 py-4 sm:px-6 lg:px-8">
					<div class="flex items-center justify-between">
						<h1 class="text-xl font-bold text-gray-900 dark:text-white">Codegen Tests</h1>
						<div class="flex items-center gap-2">
							<button
								class="rounded border px-2.5 py-1.5 text-xs font-medium transition-colors"
								:class="
									allSelected || someSelected
										? 'border-blue-600 text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/20'
										: 'border-gray-300 text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700'
								"
								@click="toggleSelectAll"
							>
								{{ allSelected ? "Deselect All" : "Select All" }}
							</button>
							<button
								class="rounded bg-blue-600 px-2.5 py-1.5 text-xs font-medium text-white transition-colors hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
								:disabled="isLoading"
								@click="fetchCodegenTests"
							>
								<span v-if="isLoading">Refreshing...</span>
								<span v-else>Refresh</span>
							</button>
							<button
								v-if="selectedIds.size > 0"
								class="rounded bg-red-600 px-2.5 py-1.5 text-xs font-medium text-white transition-colors hover:bg-red-700"
								@click="deleteSelectedTests"
							>
								Delete ({{ selectedIds.size }})
							</button>
							<span class="ml-2 text-xs text-gray-500 dark:text-gray-400">
								{{ totalTests }} tests
							</span>
						</div>
					</div>
				</div>
			</div>

			<!-- Loading State -->
			<div v-if="isLoading" class="flex flex-1 flex-col items-center justify-center">
				<div class="mb-3 h-10 w-10 animate-spin rounded-full border-b-2 border-blue-600"></div>
				<p class="text-sm text-gray-600 dark:text-gray-400">Loading codegen tests...</p>
			</div>

			<!-- Empty State -->
			<div
				v-else-if="codegenTests.length === 0"
				class="flex flex-1 flex-col items-center justify-center"
			>
				<div class="mb-2 text-4xl">📝</div>
				<h2 class="mb-1 text-sm font-semibold text-gray-900 dark:text-white">No tests found</h2>
				<p class="text-xs text-gray-600 dark:text-gray-400">
					No codegen tests have been generated yet
				</p>
			</div>

			<!-- Grouped Tests List -->
			<div v-else class="flex-1 overflow-y-auto">
				<div class="space-y-4 px-4 py-4 sm:px-6 lg:px-8">
					<div v-for="[docName, tests] in groupedTests()" :key="docName" class="space-y-2">
						<!-- Document group header -->
						<div class="sticky top-0 flex items-center gap-2 bg-gray-50 px-3 py-2 dark:bg-gray-900">
							<input
								type="checkbox"
								class="h-4 w-4 cursor-pointer rounded border-gray-300 accent-blue-600"
								:checked="isDocumentFullySelected(docName)"
								:indeterminate="isDocumentPartiallySelected(docName)"
								@change="toggleDocumentSelection(docName)"
							/>
							<h3
								class="text-xs font-semibold tracking-wider text-gray-600 uppercase dark:text-gray-400"
							>
								{{ docName }}
							</h3>
						</div>

						<!-- Tests in group -->
						<div class="space-y-1">
							<div
								v-for="test in tests"
								:key="test.id"
								class="flex flex-col gap-2 rounded border border-gray-200 bg-white p-3 transition-all hover:border-gray-300 dark:border-gray-700 dark:bg-gray-800 dark:hover:border-gray-600"
								:class="
									selectedIds.has(test.id)
										? 'border-blue-500 bg-blue-50 shadow-md dark:bg-blue-900/20'
										: ''
								"
							>
								<!-- Top row: checkbox and content -->
								<div class="flex min-w-0 items-start gap-3">
									<!-- Checkbox -->
									<input
										type="checkbox"
										:checked="selectedIds.has(test.id)"
										class="mt-0.5 h-4 w-4 shrink-0 cursor-pointer rounded border-gray-300 text-blue-600 focus:ring-blue-500"
										@click.stop="toggleSelection(test.id)"
									/>

									<!-- Content -->
									<div class="min-w-0 flex-1">
										<h4 class="truncate text-sm font-medium text-gray-900 dark:text-white">
											{{ test.ruleTitle }}
										</h4>
										<p class="truncate text-xs text-gray-500 dark:text-gray-400">
											{{ test.ruleDescription }}
										</p>
									</div>

									<!-- Actions -->
									<div class="flex shrink-0 items-center gap-1">
										<button
											class="rounded p-1.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-blue-600 dark:hover:bg-gray-700 dark:hover:text-blue-400"
											title="View code"
											@click.stop="viewCode(test)"
										>
											<svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<path
													stroke-linecap="round"
													stroke-linejoin="round"
													stroke-width="2"
													d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"
												/>
											</svg>
										</button>
										<button
											class="rounded p-1.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-red-600 dark:hover:bg-gray-700 dark:hover:text-red-400"
											title="Delete test"
											@click.stop="deleteTest(test.id)"
										>
											<svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<path
													stroke-linecap="round"
													stroke-linejoin="round"
													stroke-width="2"
													d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
												/>
											</svg>
										</button>
									</div>
								</div>

								<!-- Metadata badges - wraps on small screens -->
								<div class="flex flex-wrap gap-2 text-xs text-gray-500 dark:text-gray-400">
									<span class="truncate rounded bg-gray-100 px-1.5 py-0.5 dark:bg-gray-700">
										{{ test.type }}
									</span>
									<span
										v-if="test.cloudProvider"
										class="truncate rounded bg-purple-100 px-1.5 py-0.5 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400"
									>
										{{ test.cloudProvider }}
									</span>
									<span
										v-if="test.statementDisplayId"
										class="truncate rounded bg-amber-100 px-1.5 py-0.5 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400"
									>
										{{ test.statementDisplayId }}
									</span>
									<span class="truncate text-gray-400 dark:text-gray-500">
										{{ formatDate(test.updatedAt) }}
									</span>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>
		</div>

		<!-- Code Viewer Sidebar -->
		<div
			v-if="showCodeViewer && selectedTest"
			class="flex w-96 shrink-0 flex-col overflow-hidden border-l border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800"
		>
			<!-- Sidebar Header -->
			<div class="border-b border-gray-200 p-4 dark:border-gray-700">
				<div class="mb-2 flex items-start justify-between gap-3">
					<h3 class="truncate text-sm font-semibold text-gray-900 dark:text-white">
						{{ selectedTest.ruleTitle }}
					</h3>
					<button
						class="shrink-0 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
						@click="showCodeViewer = false"
					>
						<svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M6 18L18 6M6 6l12 12"
							/>
						</svg>
					</button>
				</div>
				<p class="truncate text-xs text-gray-600 dark:text-gray-400">
					{{ selectedTest.ruleDescription }}
				</p>
			</div>

			<!-- Code Tabs -->
			<div class="flex gap-1 border-b border-gray-200 px-4 pt-3 dark:border-gray-700">
				<button
					v-for="tab in ['code', 'passingCode', 'failingCode'] as const"
					:key="tab"
					class="rounded-t border-b-2 px-2 py-1.5 text-xs font-medium transition-colors"
					:class="
						selectedCodeTab === tab
							? 'border-blue-600 text-blue-600'
							: 'border-transparent text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-300'
					"
					@click="selectedCodeTab = tab"
				>
					{{ tab === "code" ? "Generated" : tab === "passingCode" ? "Passing" : "Failing" }}
				</button>
			</div>

			<!-- Code Content -->
			<div class="flex-1 overflow-auto p-4">
				<pre
					class="break-word overflow-auto rounded bg-gray-900 p-3 font-mono text-xs whitespace-pre-wrap text-gray-100"
				><code>{{ currentCodeContent }}</code></pre>
			</div>

			<!-- Sidebar Footer -->
			<div
				class="space-y-2 border-t border-gray-200 p-3 text-xs text-gray-600 dark:border-gray-700 dark:text-gray-400"
			>
				<div class="flex items-center justify-between">
					<span>Type:</span>
					<span class="font-mono">{{ selectedTest.type }}</span>
				</div>
				<div v-if="selectedTest.cloudProvider" class="flex items-center justify-between">
					<span>Provider:</span>
					<span class="font-mono">{{ selectedTest.cloudProvider }}</span>
				</div>
				<div class="flex items-center justify-between">
					<span>Updated:</span>
					<span class="font-mono">{{ formatDate(selectedTest.updatedAt) }}</span>
				</div>
			</div>
		</div>

		<!-- Delete Confirmation Dialog -->
		<div
			v-if="showDeleteDialog"
			class="bg-opacity-50 fixed inset-0 z-50 flex items-center justify-center bg-black p-4"
			@click.self="showDeleteDialog = false"
		>
			<div class="w-full max-w-md rounded-lg bg-white p-6 shadow-xl dark:bg-gray-800">
				<h3 class="mb-2 text-lg font-semibold text-gray-900 dark:text-white">Confirm Deletion</h3>
				<p class="mb-6 text-sm text-gray-600 dark:text-gray-400">
					{{ deleteDialogMessage }}
				</p>
				<div class="flex justify-end gap-3">
					<button
						class="rounded-lg border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600"
						:disabled="isDeleting"
						@click="showDeleteDialog = false"
					>
						Cancel
					</button>
					<button
						class="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-50"
						:disabled="isDeleting"
						@click="confirmDelete"
					>
						{{ isDeleting ? "Deleting..." : "Delete" }}
					</button>
				</div>
			</div>
		</div>
	</div>
</template>

<style scoped>
/* Custom scrollbar for code blocks */
pre {
	scrollbar-width: thin;
	scrollbar-color: rgba(156, 163, 175, 0.5) transparent;
}

pre::-webkit-scrollbar {
	width: 6px;
	height: 6px;
}

pre::-webkit-scrollbar-track {
	background: transparent;
}

pre::-webkit-scrollbar-thumb {
	background-color: rgba(156, 163, 175, 0.5);
	border-radius: 3px;
}

pre::-webkit-scrollbar-thumb:hover {
	background-color: rgba(156, 163, 175, 0.7);
}
</style>
