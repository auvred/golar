<script setup lang="ts">
import {
	DOCUMENT_STATES,
	DOCUMENT_STATES_PRETTY,
	DOCUMENT_TYPES,
	type DocumentState
} from "@nonfx/foundation/documents";
import type { CommonDocument } from "./DocumentList.vue";
import { useDocumentMetadataStore } from "~/store/document-metadata-store";
import { useDocumentMappingStore } from "~/store/document-mapping-store";
import { useUserStore } from "~/store/user-store";
import Fuse from "fuse.js";
import { getFlagEmoji } from "~/lib";
import type { DBDocumentSchema } from "@nonfx/stance-schema";
import { useFeatureFlagStore } from "@nonfx/feature-flags";
import { formatStancePath } from "~/utils";

const props = defineProps({
	mergedDocuments: {
		type: Array as PropType<Array<CommonDocument>>,
		required: true
	},

	searchTerm: {
		type: String,
		default: () => ""
	}
});

const featureFlagStore = useFeatureFlagStore();
const metadataStore = useDocumentMetadataStore();
const documentStore = useDocumentMappingStore();
const userStore = useUserStore();
const { mergedDocuments, searchTerm } = toRefs(props);
const regionNames = new Intl.DisplayNames(["en"], { type: "region" });
const sortBy = useRouteQuery<SortFields>("sortBy", "title");
const sortDirection = useRouteQuery<"asc" | "desc">("sortDirection", "asc");
const editedDocumentMetaId = ref<string | null>(null);

// State for inline actions (only needed for confirmation popovers)
const confirmDeleteDraft = ref(false);
const confirmDecommissionDocument = ref(false);
const draftDocumentDropdown = ref<DBDocumentSchema | null>(null);
const publishedDocumentDropdown = ref<DBDocumentSchema | null>(null);
const stancesUsingDocument = ref<
	Array<{ id: string; title: string; departmentPath: string | null; regionPath: string | null }>
>([]);

const emit = defineEmits<{
	"document-click": [param: { id: string; isDraft: boolean }];
}>();

// Computed permissions
function canDecommissionDocument(doc: CommonDocument) {
	return (
		userStore.permissions.includes("documents-decommission") &&
		doc.state === DOCUMENT_STATES.PUBLISHED
	);
}

const canUpdateDocument = computed(() => {
	return userStore.permissions.includes("documents-update");
});

const fuse = computed(() => {
	return new Fuse(mergedDocuments.value, {
		keys: ["displayName", "publisher", "type", "country", "owner"],
		threshold: 0.4,
		minMatchCharLength: 2,
		ignoreLocation: true,
		includeScore: true,
		shouldSort: true
	});
});

const statusFilter = ref<Array<string>>([]);
const typeFilter = ref<Array<string>>([]);
const regionFilter = ref<Array<string>>([]);
const publisherFilter = ref<Array<string>>([]);
const ownerFilter = ref<Array<string>>([]);

function isValueFiltered(value: string | null | undefined, filterList: Array<string>): boolean {
	if (filterList.length === 0) {
		return false;
	}
	const normalizedValue = value ?? "";
	const isEmptyValue = !normalizedValue;
	return filterList.includes(normalizedValue) || (isEmptyValue && filterList.includes("__EMPTY__"));
}

const searchedDocuments = computed(() => {
	let filteredDocs = mergedDocuments.value;

	// Apply search filter
	if (searchTerm.value.trim() !== "") {
		const searchResults = fuse.value.search(searchTerm.value);
		filteredDocs = searchResults.map(result => result.item);
	}

	// Apply column filters
	filteredDocs = filteredDocs.filter(document => {
		// Status filter
		if (isValueFiltered(document.state, statusFilter.value)) {
			return false;
		}

		// Type filter
		if (isValueFiltered(document.type, typeFilter.value)) {
			return false;
		}

		// Region filter
		if (isValueFiltered(document.country, regionFilter.value)) {
			return false;
		}

		// Publisher filter
		if (isValueFiltered(document.publisher, publisherFilter.value)) {
			return false;
		}

		// Owner filter
		if (isValueFiltered(document.owner, ownerFilter.value)) {
			return false;
		}

		return true;
	});

	return filteredDocs;
});

// Adjust display data
const documents = computed(() => {
	const searchResults = searchedDocuments.value.map(document => {
		const { type, state, publisher } = document;
		const documentTypeDisplay = DOCUMENT_TYPES.find(doctype => doctype.id === type)?.name ?? type;
		const isPublished = state === DOCUMENT_STATES.PUBLISHED;
		const publisherObj = metadataStore.publishers.find(o => o.id === publisher);

		return {
			...document,
			documentTypeDisplay,
			isPublished,
			publisherName: publisherObj?.name ?? "Unknown Publisher"
		};
	});

	// If we are searching, return the search results directly without sorting
	if (searchTerm.value.trim() !== "") {
		return searchResults;
	}

	return searchResults.sort((a, b) => {
		const valA = getSortValue(a, sortBy.value);
		const valB = getSortValue(b, sortBy.value);

		if (typeof valA === "string" || typeof valB === "string") {
			return sortDirection.value === "asc"
				? String(valA).localeCompare(String(valB))
				: String(valB).localeCompare(String(valA));
		}

		return sortDirection.value === "asc" ? valA - valB : valB - valA;
	});
});

const statusFilterOptions = computed(() => {
	return getUniqueDocumentFilterOptions(mergedDocuments.value, "state").map(filterOption => {
		const docType = filterOption.value.trim();

		// Handle the special "__EMPTY__" value
		if (docType === "__EMPTY__") {
			return {
				label: "Unknown",
				value: "__EMPTY__",
				indent: 0,
				metadataId: ""
			};
		}

		const label = DOCUMENT_STATES_PRETTY[docType as DocumentState];
		return {
			label,
			value: docType,
			indent: 0,
			metadataId: ""
		};
	});
});

const typeFilterOptions = computed(() => {
	return getUniqueDocumentFilterOptions(mergedDocuments.value, "documentType").map(filterOption => {
		const docType = filterOption.value.trim();

		// Handle the special "__EMPTY__" value
		if (docType === "__EMPTY__") {
			return {
				label: "Unknown",
				value: "__EMPTY__",
				indent: 0,
				metadataId: ""
			};
		}

		const label = DOCUMENT_TYPES.find(o => o.id === docType)?.name ?? docType;
		return {
			label,
			value: docType,
			indent: 0,
			metadataId: ""
		};
	});
});

const regionFilterOptions = computed(() => {
	return [
		...getUniqueDocumentFilterOptions(mergedDocuments.value, "country").map(filterOption => {
			const region = filterOption.value.trim();

			// Handle the special "__EMPTY__" value
			if (region === "__EMPTY__") {
				return {
					label: "Unknown",
					value: "__EMPTY__",
					indent: 0,
					metadataId: ""
				};
			}

			let regionName = region.trim();

			try {
				regionName = regionNames.of(region.trim()) ?? regionName;
			} catch {
				// regionNames throws error for invalid region codes
			}

			const label = region === "001" ? "🌐 Earth" : `${getFlagEmoji(region.trim())} ${regionName}`;

			return {
				label,
				value: region,
				indent: 0,
				metadataId: ""
			};
		})
	];
});

const publisherFilterOptions = computed(() => {
	return getUniqueDocumentFilterOptions(mergedDocuments.value, "documentPublisher")
		.map(filterOption => {
			const publisherValue = filterOption.value.trim();

			// Handle the special "__EMPTY__" value
			if (publisherValue === "__EMPTY__") {
				return {
					label: "Unknown",
					value: "__EMPTY__",
					indent: 0,
					metadataId: ""
				};
			}

			const publisher = metadataStore.documentPublisherOptions.find(
				o => o.data.id === publisherValue
			);

			return {
				label: publisher?.title ?? "Unknown publisher",
				value: publisherValue,
				indent: 0,
				metadataId: "",
				documentPublisher: publisher?.data.id
			};
		})
		.sort((a, b) => a.label.localeCompare(b.label));
});

const ownerFilterOptions = computed(() => {
	return getUniqueDocumentFilterOptions(mergedDocuments.value, "owner").map(filterOption => {
		const ownerValue = filterOption.value.trim();

		// Handle the special "__EMPTY__" value
		if (ownerValue === "__EMPTY__") {
			return {
				label: "Unknown",
				value: "__EMPTY__",
				indent: 0,
				metadataId: ""
			};
		}

		return {
			label: ownerValue || "Unassigned",
			value: ownerValue,
			indent: 0,
			metadataId: ""
		};
	});
});

function getDocumentLink({ id: documentId, isDraft }: { id: string; isDraft: boolean }) {
	if (isDraft) {
		return `/library/document/draft/${documentId}`;
	} else {
		return `/library/document/view/${documentId}`;
	}
}

function getSortValue(document: EnrichedDocument, field: SortFields) {
	switch (field) {
		case "title":
			return document.displayName.toLowerCase().trim();
		case "type":
			return document.type.toLowerCase();
		case "version":
			return document.version;
		case "region":
			return document.country?.toLowerCase() ?? "";
		case "status":
			return document.state;
		case "publisher":
			return document.publisherName.toLowerCase();
		case "updated":
			return new Date(document.updatedAt).getTime();
		default:
			return "";
	}
}

function handleSort(field: SortFields) {
	sortBy.value = field;
	sortDirection.value = sortDirection.value === "asc" ? "desc" : "asc";
}

function handleStatusFilter(deselectedOptions: Array<string>) {
	statusFilter.value = deselectedOptions;
}

function handleTypeFilter(deselectedOptions: Array<string>) {
	typeFilter.value = deselectedOptions;
}

function handleRegionFilter(deselectedOptions: Array<string>) {
	regionFilter.value = deselectedOptions;
}

function handleOwnerFilter(deselectedOptions: Array<string>) {
	ownerFilter.value = deselectedOptions;
}

function handlePublisherFilter(deselectedOptions: Array<string>) {
	publisherFilter.value = deselectedOptions;
}

function handleEditDraft(event: MouseEvent, doc: EnrichedDocument) {
	const editDraftLink = `/library/document/draft/${doc.id}`;
	if (event.ctrlKey || event.metaKey) {
		window.open(editDraftLink);
	} else {
		navigateTo(editDraftLink);
	}
}

function handleEditMetaPublished(doc: EnrichedDocument) {
	editedDocumentMetaId.value = doc.id;
}

async function handleEditPublished(event: MouseEvent, doc: EnrichedDocument) {
	const newDocId = await documentStore.EDIT_PUBLISHED_DOCUMENT(doc.id);
	const draftLink = `/library/document/draft/${newDocId}`;

	if (event.ctrlKey || event.metaKey) {
		window.open(draftLink);
	} else {
		await navigateTo(draftLink);
	}
}

async function handleDeleteDraft() {
	if (!draftDocumentDropdown.value) {
		return;
	}
	await documentStore.DELETE_DRAFT_DOCUMENT(draftDocumentDropdown.value.id);
	await documentStore.FETCH_DOCUMENTS();
	draftDocumentDropdown.value = null;
	confirmDeleteDraft.value = false;
}

async function handleDecommissionDocument() {
	if (!publishedDocumentDropdown.value) {
		return;
	}

	if (publishedDocumentDropdown.value.state === DOCUMENT_STATES.PUBLISHED) {
		const response = await $fetch<Array<DBDocumentSchema>>("/api/documents", {
			method: "DELETE",
			body: JSON.stringify({
				id: publishedDocumentDropdown.value.id,
				force: stancesUsingDocument.value.length > 0
			}),
			credentials: "include"
		});

		// If response is an array, document is used in stances
		if (Array.isArray(response)) {
			stancesUsingDocument.value = response;
			return;
		}
	}

	await documentStore.FETCH_DOCUMENTS();
	publishedDocumentDropdown.value = null;
	confirmDecommissionDocument.value = false;
	stancesUsingDocument.value = [];
}

type SortFields =
	| "title"
	| "type"
	| "version"
	| "region"
	| "status"
	| "publisher"
	| "updated"
	| "owner";

type EnrichedDocument = CommonDocument & {
	documentTypeDisplay: string;
	isPublished: boolean;
	publisherName: string;
};
</script>

<template>
	<FDiv overflow="scroll" show-scrollbar align="top-center" direction="column">
		<table class="w-full border-collapse text-left text-sm">
			<thead class="sticky top-0 z-10">
				<tr class="border-b border-gray-200 bg-gray-50">
					<th class="w-[40%] bg-gray-100 px-4 py-3 font-medium text-gray-500">
						<div class="flex items-center gap-1.5">
							<span class="text-xs tracking-wide uppercase">Title</span>
							<DocumentListColumnFilter
								title="Title"
								:options="[]"
								:sort-direction="sortBy === 'title' ? sortDirection : 'none'"
								@sort="handleSort('title')"
							/>
						</div>
					</th>
					<th class="bg-gray-100 px-4 py-3 font-medium text-gray-500">
						<span class="text-xs tracking-wide uppercase">Version</span>
					</th>
					<th class="bg-gray-100 px-4 py-3 font-medium text-gray-500">
						<div class="flex items-center gap-1.5">
							<span class="text-xs tracking-wide uppercase">Geo</span>
							<DocumentListColumnFilter
								title="Region"
								:options="regionFilterOptions"
								:sort-direction="sortBy === 'region' ? sortDirection : 'none'"
								@sort="handleSort('region')"
								@filter="handleRegionFilter"
							/>
						</div>
					</th>
					<th class="bg-gray-100 px-4 py-3 font-medium text-gray-500">
						<div class="flex items-center gap-1.5">
							<span class="text-xs tracking-wide uppercase">Type</span>
							<DocumentListColumnFilter
								title="Type"
								:options="typeFilterOptions"
								:sort-direction="sortBy === 'type' ? sortDirection : 'none'"
								@sort="handleSort('type')"
								@filter="handleTypeFilter"
							/>
						</div>
					</th>
					<th class="bg-gray-100 px-4 py-3 font-medium text-gray-500">
						<div class="flex items-center gap-1.5">
							<span class="text-xs tracking-wide uppercase">Owner</span>
							<DocumentListColumnFilter
								title="Owner"
								:options="ownerFilterOptions"
								:sort-direction="sortBy === 'owner' ? sortDirection : 'none'"
								@sort="handleSort('owner')"
								@filter="handleOwnerFilter"
							/>
						</div>
					</th>
					<th class="bg-gray-100 px-4 py-3 font-medium text-gray-500">
						<div class="flex items-center gap-1.5">
							<span class="text-xs tracking-wide uppercase">Publisher</span>
							<DocumentListColumnFilter
								title="Publisher"
								:options="publisherFilterOptions"
								:sort-direction="sortBy === 'publisher' ? sortDirection : 'none'"
								@sort="handleSort('publisher')"
								@filter="handlePublisherFilter"
							/>
						</div>
					</th>
					<th class="bg-gray-100 px-4 py-3 font-medium text-gray-500">
						<div class="flex items-center gap-1.5">
							<span class="text-xs tracking-wide uppercase">Status</span>
							<DocumentListColumnFilter
								title="Status"
								:options="statusFilterOptions"
								:sort-direction="sortBy === 'status' ? sortDirection : 'none'"
								@sort="handleSort('status')"
								@filter="handleStatusFilter"
							/>
						</div>
					</th>
					<th class="bg-gray-100 px-4 py-3 text-right font-medium text-gray-500">
						<span class="text-xs tracking-wide uppercase">Actions</span>
					</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-gray-100">
				<tr
					v-for="doc in documents"
					:key="doc.id"
					class="group transition-colors hover:bg-gray-50/50"
				>
					<!-- Title -->
					<td class="px-4 py-3">
						<NuxtLink
							:to="getDocumentLink({ id: doc.id, isDraft: doc.state === 'DRAFT' })"
							class="f-div"
							direction="row"
							width="fill-container"
							padding="none none"
						>
							<FDiv
								align="middle-left"
								padding="none none none small"
								gap="medium"
								height="hug-content"
								width="fill-container"
							>
								<DocumentIcon
									:publisher="doc.publisher"
									size="large"
									:is-published="doc.state === 'PUBLISHED'"
								/>
								<FText :data-qa="doc.displayName" weight="medium" size="medium"
									>{{ doc.displayName }} -
									<FText inline size="x-small" state="subtle">{{
										doc.documentVersion
									}}</FText></FText
								>
							</FDiv>
						</NuxtLink>
					</td>

					<!-- Internal Version -->
					<td class="px-4 py-3">
						<DocumentVersionSelector
							v-if="doc.state !== 'DRAFT'"
							:document-id="doc.id"
							:version="doc.version"
							@select-document="
								documentId =>
									emit('document-click', { id: documentId, isDraft: doc.state === 'DRAFT' })
							"
						/>
					</td>

					<!-- Geo/Region -->
					<td class="px-4 py-3 text-center">
						<span v-if="doc.country === 'Earth' || !doc.country" class="text-lg">🌐</span>
						<span v-else class="text-lg">{{ getFlagEmoji(doc.country || "") }}</span>
					</td>

					<!-- Type -->
					<td class="px-4 py-3">
						<span class="text-gray-600">{{ doc.documentTypeDisplay }}</span>
					</td>

					<!-- Owner -->
					<td class="px-4 py-3">
						<span class="text-gray-600">{{ doc.owner }}</span>
					</td>

					<!-- Publisher -->
					<td class="px-4 py-3">
						<span class="text-gray-600">{{ doc.publisherName }}</span>
					</td>

					<!-- Status -->
					<td class="px-4 py-3 text-center">
						<FIcon
							:source="
								doc.state === 'PUBLISHED'
									? 'i-tick-fill'
									: doc.state === 'DECOMMISSIONED'
										? 'i-info-fill'
										: 'i-doc-draft-line'
							"
							size="small"
							:state="
								doc.state === 'PUBLISHED'
									? 'success'
									: doc.state === 'DECOMMISSIONED'
										? 'warning'
										: 'secondary'
							"
						>
						</FIcon>
					</td>

					<!-- Actions -->
					<td class="px-4 py-3">
						<div
							class="flex items-center justify-end gap-1 opacity-0 transition-opacity group-hover:opacity-100"
						>
							<!-- Draft document actions -->
							<template v-if="doc.state === 'DRAFT'">
								<FIconButton
									v-tooltip="`Edit Draft Document`"
									size="small"
									category="packed"
									icon="i-edit-line"
									@click.stop="handleEditDraft($event, doc)"
								/>
								<FIconButton
									v-tooltip="`Delete Draft Document`"
									size="small"
									category="packed"
									state="danger"
									icon="i-delete-line"
									@click.stop="
										confirmDeleteDraft = true;
										draftDocumentDropdown = doc.originalObject;
									"
								/>
							</template>

							<!-- Published document actions -->
							<template v-else>
								<!-- @todo - in progress -->
								<FDiv
									width="hug-content"
									gap="small"
									padding="none small none none"
									align="middle-left"
								>
									<FIconButton
										v-if="false && canUpdateDocument && featureFlagStore.featureMap.ADMIN_MODE"
										v-tooltip="`Edit Document Meta`"
										size="x-small"
										category="packed"
										icon="i-table-edit"
										@click.stop="handleEditMetaPublished(doc)"
									/>
									<FIconButton
										v-if="canUpdateDocument"
										v-tooltip="`Edit Document`"
										size="small"
										category="packed"
										icon="i-edit-line"
										@click.stop="handleEditPublished($event, doc)"
									/>
									<FIconButton
										v-if="canDecommissionDocument(doc)"
										v-tooltip="`Decomission Document`"
										size="small"
										category="packed"
										state="danger"
										icon="i-delete-line"
										@click.stop="
											confirmDecommissionDocument = true;
											publishedDocumentDropdown = doc.originalObject;
										"
									/>
								</FDiv>
							</template>
						</div>
					</td>
				</tr>
			</tbody>
		</table>

		<!-- Empty state -->
		<FDiv v-if="documents.length === 0" align="middle-center" gap="large" direction="column">
			<FIcon source="i-document" size="large" class="mb-3 text-gray-300" />
			<p class="text-sm text-gray-500">No documents found</p>
		</FDiv>
	</FDiv>

	<!-- Delete Draft Confirmation Popover -->
	<f-popover
		v-if="confirmDeleteDraft && draftDocumentDropdown"
		open
		size="custom(480px,auto)"
		:overlay="true"
		@overlay-click="confirmDeleteDraft = false"
		@esc="confirmDeleteDraft = false"
		@close="confirmDeleteDraft = false"
	>
		<FDiv state="default" direction="column" padding="medium" gap="x-large" height="hug-content">
			<FText size="medium"
				>Are you sure you want to delete the draft document
				<FText weight="bold" inline>{{ draftDocumentDropdown?.displayName }}</FText
				>?</FText
			>
			<FDiv gap="medium" height="hug-content">
				<FButton label="Delete" state="danger" @click="handleDeleteDraft"></FButton>
				<FButton
					label="Cancel"
					state="neutral"
					category="outline"
					@click="confirmDeleteDraft = false"
				></FButton>
			</FDiv>
		</FDiv>
	</f-popover>

	<!-- Decommission Document Confirmation Popover -->
	<f-popover
		v-if="confirmDecommissionDocument && publishedDocumentDropdown"
		open
		size="custom(600px,auto)"
		:overlay="true"
		@overlay-click="
			confirmDecommissionDocument = false;
			stancesUsingDocument = [];
		"
		@esc="
			confirmDecommissionDocument = false;
			stancesUsingDocument = [];
		"
		@close="
			confirmDecommissionDocument = false;
			stancesUsingDocument = [];
		"
	>
		<FDiv state="default" direction="column" padding="none">
			<FDiv border="small solid subtle bottom" padding="large" align="middle-left">
				<FDiv><FText state="secondary">Decommission Document</FText></FDiv>
				<!-- <FDiv width="hug-content">
						<FIcon source="i-close-line"  size="x-small" state="subtle" />
					</FDiv> -->
			</FDiv>
			<FDiv padding="large" height="hug-content" align="top-left">
				<FText size="medium">
					Please confirm if you’d like to decommission
					<FText weight="bold" inline>{{ publishedDocumentDropdown?.name }}</FText
					>?
				</FText>
			</FDiv>

			<!-- Show stance warning if document is used in stances -->
			<FDiv v-if="stancesUsingDocument?.length > 0" direction="column" gap="medium" padding="small">
				<FText size="medium" state="danger">
					This document is used in {{ stancesUsingDocument.length }} stance{{
						stancesUsingDocument.length > 1 ? "s" : ""
					}}. Decommissioning will remove it from these stances:
				</FText>

				<FDiv
					direction="column"
					gap="small"
					padding="small"
					variant="curved"
					height="hug-content"
					max-height="400px"
					overflow="scroll"
				>
					<FDiv
						v-for="stance in stancesUsingDocument"
						:key="stance.id"
						gap="small"
						align="middle-left"
						height="hug-content"
						padding="small"
						clickable
						@click="navigateTo(`/stances?stance=${stance.id}`)"
					>
						<FIcon source="i-stance-solo-line" size="small" state="secondary" />
						<FDiv direction="column" width="fill-container">
							<FText size="small" weight="medium">{{ stance.title }}</FText>
							<FText size="x-small" state="subtle">
								{{
									formatStancePath(
										stance.departmentPath ?? stance.regionPath ?? "",
										$t("org.org_name")
									)
								}}
							</FText>
						</FDiv>
						<FIcon source="i-external-link" size="small" state="subtle" />
					</FDiv>
				</FDiv>
			</FDiv>

			<FDiv gap="medium" padding="large" align="middle-right" height="hug-content">
				<FButton
					label="Cancel"
					state="primary"
					category="transparent"
					@click="confirmDecommissionDocument = false"
				></FButton>
				<FButton
					:label="stancesUsingDocument.length > 0 ? 'Proceed with Decommission' : 'Decommission'"
					state="danger"
					@click="handleDecommissionDocument"
				></FButton>
			</FDiv>
		</FDiv>
	</f-popover>
</template>
