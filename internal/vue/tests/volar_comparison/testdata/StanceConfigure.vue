<script setup lang="ts">
import { DOCUMENT_CATEGORIES } from "#layers/stance-base/data/stance-document-categories";
import {
	DOCUMENT_CLASS,
	DOCUMENT_TYPES,
	type DocumentClassType
} from "@nonfx/foundation/documents";
import { logger } from "@nonfx/logger";
import type { DBDocumentSchema, ResolvedStanceDocumentItem } from "@nonfx/stance-schema";

import { useApplicationStore } from "~/store/application-store";
import { useDocumentMappingStore } from "~/store/document-mapping-store";
import { useUserStore } from "~/store/user-store";
import { useStanceStore } from "~/store/stance-store";
import { getStanceCategoryIcon } from "~/utils";
import { Permissions } from "@nonfx/utils";
import { useStanceCoverage } from "~/composables/useStanceCoverage";

// Stores
const stanceStore = useStanceStore();
const documentStore = useDocumentMappingStore();
const applicationStore = useApplicationStore();
const userStore = useUserStore();

const { t } = useI18n();

// Coverage composable - initialization happens in stance-header
const { recalculateCoverageForCategory, getIsCalculating, getCoverageScore } = useStanceCoverage();

// References
const isAddDocumentsOpen = ref(false);
const selectedStanceCategory = ref<DocumentClassType | null>();

// State for document actions
const excludedDocumentList = ref<Array<DBDocumentSchema>>([]);
const shareWithOtherStances = ref<boolean>(true);

const currentStance = computed(() => {
	return stanceStore.currentStance ?? null;
});

const applicationIds = computed(() => {
	if (!currentStance.value?.evidencesData) {
		return [];
	}

	return Object.keys(currentStance.value.evidencesData.applications);
});

const platformIds = computed(() => {
	if (!currentStance.value?.evidencesData) {
		return [];
	}

	return Object.keys(currentStance.value.evidencesData.platforms);
});

const processIds = computed(() => {
	if (!currentStance.value?.evidencesData) {
		return [];
	}

	return Object.keys(currentStance.value.evidencesData.processes);
});

// watch currentStance and reset local state when it changes
watch(
	() => currentStance.value?.id,
	() => {
		if (currentStance.value?.id) {
			resetState();
			applicationStore.getApplications();
		}
	},
	{ immediate: true }
);

// Dynamic columns based on DOCUMENT_CATEGORIES
// Layout: 1 at top (OBLIGATION), 3 in middle (POLICY, STANDARD, CONTROL_CATALOG), 1 at bottom (TESTS_EVIDENCE)
const topColumn = computed((): Column => {
	const category = DOCUMENT_CATEGORIES[0]!; // OBLIGATION
	return {
		id: category,
		name: getCategoryDisplayName(category),
		icon: getStanceCategoryIcon(category),
		documents: documentGroupByCategory.value[category]
	};
});

const middleColumns = computed((): Array<Column> => {
	// POLICY, STANDARD, CONTROL_CATALOG
	return DOCUMENT_CATEGORIES.slice(1, 4).map(category => ({
		id: category,
		name: getCategoryDisplayName(category),
		icon: getStanceCategoryIcon(category),
		documents: documentGroupByCategory.value[category]
	}));
});

const bottomColumn = computed((): Column => {
	const category = DOCUMENT_CATEGORIES[4]!; // TESTS_EVIDENCE
	return {
		id: category,
		name: getCategoryDisplayName(category),
		icon: getStanceCategoryIcon(category),
		documents: documentGroupByCategory.value[category]
	};
});

// Legacy support for existing code
const verticalColumns = computed((): Array<Column> => {
	return [topColumn.value];
});

const horizontalColumns = computed((): Array<Column> => {
	return [...middleColumns.value, bottomColumn.value];
});

const documentGroupByCategory = computed(() => {
	const stanceValue = currentStance.value;
	if (!stanceValue) {
		return DOCUMENT_CATEGORIES.reduce(
			(acc, category) => {
				acc[category] = [];
				return acc;
			},
			{} as Record<DocumentClassType, Array<ResolvedStanceDocumentItem>>
		);
	}

	const docCategoryMap = DOCUMENT_CATEGORIES.reduce(
		(acc, category) => {
			acc[category] = [];
			return acc;
		},
		{} as Record<DocumentClassType, Array<ResolvedStanceDocumentItem>>
	);

	DOCUMENT_CATEGORIES.forEach(category => {
		const matchingDocuments = stanceValue.documentsData.documents.filter(
			doc => doc.documentClass === category
		);
		docCategoryMap[category] = matchingDocuments;

		matchingDocuments.sort((a, b) => {
			// First priority: isInherited true comes first
			if (a.isInherited && !b.isInherited) {
				return -1;
			}
			if (!a.isInherited && b.isInherited) {
				return 1;
			}

			// Second priority: isShared true comes before isShared false
			if (a.isShared && !b.isShared) {
				return -1;
			}
			if (!a.isShared && b.isShared) {
				return 1;
			}

			// If all conditions are equal, maintain original order
			return 0;
		});
	});

	return docCategoryMap;
});

const existingDocumentNames = computed(() => {
	return currentStance.value?.documentsData.documents.map(document => document.documentName) ?? [];
});

const canAddDocumentToStance = computed(() => {
	return userStore.permissions.includes(Permissions.STANCES_CREATE);
});

onBeforeUnmount(() => {
	resetState();
});

async function addDocumentToStance(documents: Array<DBDocumentSchema>) {
	try {
		if (!selectedStanceCategory.value) {
			logger.error("No document category selected for adding documents.");
			return;
		}

		await stanceStore.addDocumentsToStance({
			documents,
			documentCategory: selectedStanceCategory.value,
			isShared: shareWithOtherStances.value
		});

		// Recalculate relevant coverage based on what was added
		await recalculateCoverageForCategory(selectedStanceCategory.value);
	} catch (error) {
		logger.error("Error adding documents to folder:", error);
	}
}

// Helper function to get documents by category
function getDocumentByCategory(
	documentCategory: DocumentClassType
): Array<ResolvedStanceDocumentItem> {
	if (!currentStance.value) {
		return [];
	}
	return currentStance.value.documentsData.documents.filter(
		(doc: ResolvedStanceDocumentItem) => doc.documentClass === documentCategory
	);
}

// Handle document removal and recalculate coverage
async function handleDocumentRemoved(documentClass: DocumentClassType) {
	try {
		await stanceStore.refreshStances(true);
		await recalculateCoverageForCategory(documentClass);
	} catch (err) {
		logger.error("Error recalculating coverage:", err);
	}
}

function compareDocuments(categoryType: DocumentClassType) {
	const columnDocuments: Array<string> = getDocumentByCategory(categoryType).map(
		(doc: ResolvedStanceDocumentItem) => doc.id
	);
	let comparedDocuments: Array<string> = [];

	if (categoryType === DOCUMENT_CLASS.POLICY) {
		comparedDocuments = getDocumentByCategory(DOCUMENT_CLASS.OBLIGATION).map(
			(doc: ResolvedStanceDocumentItem) => doc.id
		);
	} else if (categoryType === DOCUMENT_CLASS.STANDARD) {
		comparedDocuments = getDocumentByCategory(DOCUMENT_CLASS.POLICY).map(
			(doc: ResolvedStanceDocumentItem) => doc.id
		);
	} else if (categoryType === DOCUMENT_CLASS.CONTROL_CATALOG) {
		comparedDocuments = getDocumentByCategory(DOCUMENT_CLASS.STANDARD).map(
			(doc: ResolvedStanceDocumentItem) => doc.id
		);
	}
	// We don't compare other categories
	else {
		return;
	}

	navigateTo({
		path: "/compare",
		query: {
			compare: comparedDocuments,
			against: columnDocuments
		}
	});
}

function openDesignPage(categoryType: DocumentClassType) {
	const rightSideStanceCategory: DocumentClassType = categoryType;
	let leftSideStanceCategory: DocumentClassType = "OBLIGATION";

	if (categoryType === DOCUMENT_CLASS.POLICY) {
		leftSideStanceCategory = DOCUMENT_CLASS.OBLIGATION;
	} else if (categoryType === DOCUMENT_CLASS.STANDARD) {
		leftSideStanceCategory = DOCUMENT_CLASS.POLICY;
	} else if (categoryType === DOCUMENT_CLASS.CONTROL_CATALOG) {
		leftSideStanceCategory = DOCUMENT_CLASS.STANDARD;
	} else {
		return;
	}

	navigateTo({
		path: "/design",
		query: {
			stance: currentStance.value?.id,
			leftSideStanceCategory,
			rightSideStanceCategory
		}
	});
}

function onShareWithOtherStancesChange(event: CustomEvent) {
	shareWithOtherStances.value = event.detail.value;
}

function navigateToOperate() {
	if (currentStance.value?.id) {
		navigateTo({
			path: "/operate",
			query: { stance: currentStance.value.id }
		});
	}
}

function toggleAddDocuments(columnId: DocumentClassType) {
	const column = [...horizontalColumns.value, ...verticalColumns.value].find(
		col => col.id === columnId
	);

	if (!column) {
		logger.error(`Column with ID ${columnId} not found.`);
		return;
	}

	const documentTypeToExclude = DOCUMENT_TYPES.filter(type => {
		return !type.documentClasses.some(docClass => docClass === columnId);
	}).map(type => type.id);

	const excludedDocs = documentStore.documentsMetaList.filter(
		doc =>
			existingDocumentNames.value.includes(doc.name) || documentTypeToExclude.includes(doc.type)
	);

	excludedDocumentList.value = excludedDocs;
	selectedStanceCategory.value = columnId;
	isAddDocumentsOpen.value = !isAddDocumentsOpen.value;
}

// Helper function to get display names for categories
function getCategoryDisplayName(category: DocumentClassType): string {
	switch (category) {
		case "OBLIGATION":
			return t("org.obligations_title");
		case "POLICY":
			return t("org.policies_title");
		case "STANDARD":
			return t("org.standards_title");
		case "CONTROL_CATALOG":
			return t("org.controls_title");
		case "TESTS_EVIDENCE":
			return t("org.evidence_title");
		default:
			return category;
	}
}

function resetState() {
	isAddDocumentsOpen.value = false;
	selectedStanceCategory.value = null;
	shareWithOtherStances.value = true;
	excludedDocumentList.value = [];
	// Coverage state is managed by stance-header component
}

type Column = {
	id: DocumentClassType;
	name: string;
	icon: string;
	documents?: Array<ResolvedStanceDocumentItem>;
};
</script>

<template>
	<!-- do not remove the key attribute, it is used to force re-rendering of the component when stance changes -->
	<!-- while switch between stances the documents and other data can we same having a key rendered will force the component to re-render and fetch the new data  -->
	<FDiv :key="currentStance?.id" direction="column" padding="none" gap="none">
		<StancesGlobalStanceAcknowledgementAlert />

		<!-- Top Column (Obligations) -->
		<FDiv
			state="secondary"
			:data-qa-id="`${topColumn.id}`"
			padding="small small x-small small"
			height="hug-content"
			width="fill-container"
		>
			<FDiv
				direction="column"
				state="default"
				style="border-radius: 4px"
				width="fill-container"
				max-height="250px"
				overflow="hidden"
			>
				<FDiv
					align="middle-left"
					state="default"
					padding="small small"
					gap="medium"
					width="fill-container"
					height="hug-content"
					border="small solid subtle bottom"
				>
					<FDiv align="middle-left" width="fill-container" height="fill-container" gap="small">
						<FIcon source="i-document-line" size="medium" state="secondary"></FIcon>
						<FText weight="medium" state="secondary" size="medium">{{ topColumn.name }}</FText>
					</FDiv>
					<FDiv
						v-if="canAddDocumentToStance"
						align="middle-center"
						height="hug-content"
						gap="x-small"
						padding="none none none none"
						direction="row"
						width="hug-content"
					>
						<FIconButton
							v-tooltip="`Add`"
							category="transparent"
							size="x-small"
							:data-qa-id="`add-document-btn-${topColumn.id}`"
							icon="i-plus-line"
							style="transform: scale(0.8); transform-origin: 50% 50%"
							@click="toggleAddDocuments(topColumn.id)"
						></FIconButton>
					</FDiv>
				</FDiv>
				<FDiv padding="x-small">
					<FDiv height="hug-content" direction="column" padding="none" gap="x-small">
						<FDiv direction="column" gap="x-small">
							<ScrollArea>
								<FDiv direction="row" width="100%" gap="small" padding="none" class="stance-grid">
									<StancesDocumentCard
										v-for="(document, idx) in topColumn.documents"
										:key="`${document.documentName}-${idx}-${document.isInherited}-${document.isShared}-${document.documentVersion}`"
										:document="document"
										@document-removed="handleDocumentRemoved"
									/>
									<StancesSelectDocumentCard
										:data-qa-id="`select-document-btn-${topColumn.id}`"
										@click="toggleAddDocuments(topColumn.id)"
									/>
								</FDiv>
							</ScrollArea>
						</FDiv>
					</FDiv>
				</FDiv>
			</FDiv>
		</FDiv>

		<!-- Middle Columns (Policies, Standards, Controls) -->
		<FDiv
			direction="row"
			gap="x-small"
			padding="none small"
			state="secondary"
			align="middle-center"
		>
			<FDiv
				v-for="column in middleColumns"
				:key="column.id"
				state="default"
				direction="row"
				width="fill-container"
				style="border-radius: 4px"
			>
				<FDiv
					direction="column"
					gap="none"
					height="100%"
					width="fill-container"
					overflow="scroll"
					padding="none"
				>
					<!-- Header -->
					<FDiv
						direction="row"
						sticky="top"
						border="small solid subtle bottom"
						gap="small"
						height="hug-content"
						padding="small small"
						width="fill-container"
						state="default"
					>
						<FDiv
							padding="none"
							gap="small"
							width="fill-container"
							align="middle-left"
							state="default"
						>
							<FDiv
								direction="row"
								gap="small"
								align="middle-left"
								width="fill-container"
								padding="none"
								state="default"
							>
								<FDiv align="middle-left" width="hug-content" gap="medium" height="fill-container">
									<FDiv align="middle-left" width="hug-content" height="fill-container" gap="small">
										<FIcon :source="column.icon" size="medium" state="secondary"></FIcon>
										<FText weight="medium" state="secondary" size="medium">{{ column.name }}</FText>
									</FDiv>
								</FDiv>
								<FDiv
									align="middle-left"
									padding="none none none small"
									direction="column"
									width="hug-content"
									height="hug-content"
								>
									<!-- <FText size="x-small" state="subtle" variant="para" weight="regular">
													COVERAGE
												</FText> -->
									<FIcon
										v-if="getIsCalculating(column.id)"
										source="i-loader"
										size="x-small"
										:loading="getIsCalculating(column.id)"
									></FIcon>
									<FText
										v-else
										size="small"
										variant="heading"
										weight="bold"
										:data-qa-id="`coverage-score-${column.id}`"
									>
										{{ getCoverageScore(column.id) }}%
										<!-- <span style="font-size: 11px; padding-left:0px">%</span> -->
									</FText>
								</FDiv>
								<FDiv
									tooltip="Coverage Score"
									width="fill-container"
									height="hug-content"
									align="middle-right"
									gap="small"
								>
									<FDiv
										direction="row"
										width="hug-content"
										height="hug-content"
										align="middle-right"
										padding="none"
									>
										<FDiv
											align="middle-center"
											height="hug-content"
											gap="x-small"
											padding="none none none none"
											direction="row"
										>
											<!-- //todo: @shubham: <task>Auto load the left side on the design tab</task> -->

											<FIconButton
												v-tooltip="`Design`"
												category="packed"
												size="small"
												icon="i-design-line"
												class="scale-click h-auto!"
												state="secondary"
												@click="openDesignPage(column.id)"
											></FIconButton>

											<FIconButton
												v-tooltip="`Compare`"
												category="packed"
												size="small"
												icon="i-compare-1-line"
												class="scale-click h-auto!"
												state="secondary"
												@click="compareDocuments(column.id)"
											></FIconButton>
											<FIconButton
												v-if="canAddDocumentToStance"
												v-tooltip="`Add`"
												category="transparent"
												size="x-small"
												icon="i-plus-line"
												style="transform: scale(0.8); transform-origin: 50% 50%"
												@click="toggleAddDocuments(column.id)"
											></FIconButton>
										</FDiv>
									</FDiv>
								</FDiv>
							</FDiv>
						</FDiv>
					</FDiv>

					<FDiv direction="column" gap="small" padding="x-small small" height="hug-content">
						<!-- Documents -->
						<FDiv height="hug-content" direction="column" padding="none" class="stance-grid">
							<StancesDocumentCard
								v-for="(document, idx) in column.documents"
								:key="`${document.id}-${idx}-${document.isInherited}-${document.isShared}`"
								:document="document"
								@document-removed="handleDocumentRemoved"
							/>
							<StancesSelectDocumentCard
								:data-qa-id="`select-document-btn-${column.id}`"
								@click="toggleAddDocuments(column.id)"
							/>
						</FDiv>
					</FDiv>
				</FDiv>
			</FDiv>
		</FDiv>

		<!-- Bottom Column (Evidence) -->
		<FDiv
			state="secondary"
			padding="x-small small x-small small"
			height="hug-content"
			width="fill-container"
		>
			<FDiv
				direction="column"
				state="default"
				style="border-radius: 4px"
				width="fill-container"
				overflow="hidden"
			>
				<FDiv
					align="middle-left"
					state="default"
					padding="small"
					gap="medium"
					width="fill-container"
					height="hug-content"
					border="small solid subtle bottom"
				>
					<FDiv align="middle-left" width="fill-container" height="fill-container" gap="small">
						<FDiv align="middle-left" width="hug-content" gap="medium" height="fill-container">
							<FDiv align="middle-left" width="hug-content" height="fill-container" gap="small">
								<FIcon :source="bottomColumn.icon" size="medium" state="secondary"></FIcon>
								<FText weight="medium" state="secondary" size="medium">{{
									bottomColumn.name
								}}</FText>
							</FDiv>
						</FDiv>
						<FDiv
							align="middle-left"
							padding="none none none small"
							direction="column"
							width="hug-content"
							height="hug-content"
						>
							<FIcon
								v-if="getIsCalculating('TESTS_EVIDENCE')"
								source="i-loader"
								size="x-small"
								:loading="getIsCalculating('TESTS_EVIDENCE')"
							></FIcon>
							<FText v-else size="small" variant="heading" weight="bold">
								{{ getCoverageScore(DOCUMENT_CLASS.TESTS_EVIDENCE) }}%
							</FText>
						</FDiv>
						<FDiv width="fill-container" height="hug-content" align="middle-right" gap="small">
							<FIconButton
								v-tooltip="`Go to Operate`"
								category="packed"
								size="small"
								icon="i-operate-line"
								class="scale-click h-auto!"
								state="secondary"
								@click="navigateToOperate"
							></FIconButton>
						</FDiv>
					</FDiv>
				</FDiv>

				<FDiv padding="x-small">
					<!-- Three columns inside Evidence section -->
					<FDiv direction="row" gap="x-small" height="hug-content">
						<!-- PLATFORMS Column -->
						<FDiv direction="column" width="fill-container" gap="x-small">
							<FDiv
								direction="row"
								align="middle-left"
								gap="small"
								padding="small"
								border="small solid subtle bottom"
							>
								<FIcon source="i-platform-line" size="small" state="secondary"></FIcon>
								<FText weight="medium" state="secondary" size="small">PLATFORMS</FText>
							</FDiv>
							<FDiv direction="column" gap="x-small" padding="small">
								<StancesApplicationCard
									v-for="platformId in platformIds"
									:key="platformId"
									:application-id="platformId"
								/>
							</FDiv>
						</FDiv>

						<!-- APPLICATIONS Column -->
						<FDiv direction="column" width="fill-container" gap="x-small">
							<FDiv
								direction="row"
								align="middle-left"
								gap="small"
								padding="small"
								border="small solid subtle bottom"
							>
								<FIcon source="i-platform-line" size="small" state="secondary"></FIcon>
								<FText weight="medium" state="secondary" size="small">APPLICATIONS</FText>
							</FDiv>
							<FDiv direction="column" gap="x-small" padding="small">
								<StancesApplicationCard
									v-for="applicationId in applicationIds"
									:key="applicationId"
									:application-id="applicationId"
								/>
							</FDiv>
						</FDiv>

						<!-- PROCESSES Column -->
						<FDiv direction="column" width="fill-container" gap="x-small">
							<FDiv
								direction="row"
								align="middle-left"
								gap="small"
								padding="small"
								border="small solid subtle bottom"
							>
								<FIcon source="i-process" size="small" state="secondary"></FIcon>
								<FText weight="medium" state="secondary" size="small">PROCESSES</FText>
							</FDiv>
							<FDiv direction="column" gap="x-small" padding="small">
								<StancesApplicationCard
									v-for="processId in processIds"
									:key="processId"
									:application-id="processId"
								/>
							</FDiv>
						</FDiv>
					</FDiv>
				</FDiv>
			</FDiv>
		</FDiv>

		<!-- Add Documents Popover -->
		<DocumentPicker
			v-if="isAddDocumentsOpen"
			:target="`#add-document-btn-${selectedStanceCategory}`"
			:documents-to-exclude="excludedDocumentList"
			placement="auto"
			@on-document-select="addDocumentToStance"
			@close="isAddDocumentsOpen = false"
		>
			<template #footer-left>
				<FDiv
					direction="row"
					height="hug-content"
					width="fill-container"
					gap="small"
					align="middle-left"
					class="switch-overide"
				>
					<f-switch
						size="small"
						:value="shareWithOtherStances"
						state="default"
						@input="onShareWithOtherStancesChange"
					>
						<FDiv slot="label" padding="none">
							<FText variant="para" size="medium">Share with other Stances</FText>
						</FDiv>
					</f-switch>
				</FDiv>
			</template>
		</DocumentPicker>
	</FDiv>
</template>

<style>
.switch-overide {
	--color-neutral-secondary: var(--color-text-subtle);
}

.stance-grid {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); /* Auto-wrap columns */
	gap: 2px !important;
	width: 100%;
	.f-text {
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		width: 100%;
	}
}

.stance-card {
	border-radius: 2px;
}

.stance-score > .f-div {
	border-radius: 4px;
	margin-top: -2px;
	.f-text {
		font-size: 22px;
		color: var(--color-danger-default);
	}
}
</style>
