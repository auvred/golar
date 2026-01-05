<script setup lang="ts">
import { navigateTo } from "#app";
import { useStatementMetadataStore } from "#layers/stance-base/app/store/statement-metadata-store";
import type { EvidenceWithFileEndPoint } from "#layers/stance-base/server";
import { logger } from "@nonfx/logger";
import type {
	DBStatementSchemaWithComputedProps,
	ApplicationWithEnvironmentsAndStances
} from "@nonfx/stance-schema";
import type { PropType } from "vue";
import { getMetadataForNode } from "~/lib";
import { useApplicationStore } from "~/store/application-store";
import { useAuditStore } from "~/store/audits-store";
import { useDocumentMappingStore } from "~/store/document-mapping-store";
import type { ControlEvidence } from "~/types/application-types";
import { isPdfFile, isImageFile } from "~/utils/file-utils";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue
} from "@/components/ui/select";

const documentMappingStore = useDocumentMappingStore();
const applicationStore = useApplicationStore();
const auditStore = useAuditStore();
const statementMetadataStore = useStatementMetadataStore();

const props = defineProps({
	control: {
		type: Object as PropType<DBStatementSchemaWithComputedProps>,
		required: true
	},

	showAddEvidence: {
		type: Boolean,
		default: true
	},

	stanceId: {
		type: String,
		required: true
	},

	applicationId: {
		type: String,
		required: false,
		default: undefined
	}
});

const evidenceJsonContent = ref<string | null>(null);
const selectedEvidence = ref<EvidenceWithFileEndPoint | null>(null);
const selectedReasoningEvidence = ref<EvidenceWithFileEndPoint | null>(null);
const isLoading = ref<boolean>(false);
const errorMessage = ref<string | null>(null);
const selectedEvidenceForTimelineId = ref<string | null>(null);
const isMarkingEffective = ref<boolean>(false);
const isMarkingIneffective = ref<boolean>(false);
const isMarkingControlEffective = ref<boolean>(false);
const selectedEvidenceIds = ref<Set<string>>(new Set());
const isBulkMarkingEffective = ref<boolean>(false);
const isBulkMarkingIneffective = ref<boolean>(false);

// Application selector state for audit mode (when no applicationId is provided)
const selectedAppId = ref<string | null>(null);
const isRequestingAll = ref<boolean>(false);

// Determine if we're in audit mode (no specific applicationId)
const isAuditMode = computed(() => !applicationId?.value);

// Get applications for this control (for audit mode)
const controlApplications = computed((): Array<ApplicationWithEnvironmentsAndStances> => {
	if (!isAuditMode.value) {
		return [];
	}
	return auditStore.getApplicationsForControl(control.value.id);
});

// Currently active application ID (either from prop or selected in audit mode)
const activeApplicationId = computed((): string | null => {
	if (applicationId?.value) {
		return applicationId.value;
	}
	return selectedAppId.value;
});

const selectedEvidenceForTimeline = computed(() => {
	if (!selectedEvidenceForTimelineId.value) {
		return null;
	}
	return (
		filteredControlEvidences.value.find(
			evidence => evidence.id === selectedEvidenceForTimelineId.value
		) ?? null
	);
});

const isSlideoutOpen = defineModel<boolean>("isSlideoutOpen", {
	default: false,
	required: true
});

const emit = defineEmits<{
	(e: "update:isSlideoutOpen", value: boolean): void;
	(e: "closeSlideout"): void;
}>();

function closeSlideout() {
	isSlideoutOpen.value = false;
	closeEvidence();
	selectedEvidenceForTimelineId.value = null;
	selectedReasoningEvidence.value = null;
	selectedEvidenceIds.value.clear();
	selectedAppId.value = null;
	applicationStore.closeControlSlideout();
	if (isAuditMode.value) {
		auditStore.closeAuditControlSlideout();
	}
	emit("update:isSlideoutOpen", false);
}

const { control, showAddEvidence, stanceId, applicationId } = toRefs(props);

const firstControlClause = computed(() => {
	const [firstClause] = control.value.nodes;
	return firstClause ?? null;
});

const clauseMeta = computed(() => {
	if (firstControlClause.value) {
		return getMetadataForNode(firstControlClause.value);
	}
	return null;
});

const calculatedMeta = computed(() => {
	const rmfMetaKind = statementMetadataStore.metadatas.find(meta => meta.metadataType === "rmf");

	if (!rmfMetaKind) {
		return {};
	}

	const currentAxle = clauseMeta.value?.resolvedScopes.find(
		resolvedScope => resolvedScope.metadataName === rmfMetaKind.metadataName
	);

	return {
		currentAxle
	};
});

const controlDocument = computed(() => {
	const { documentId } = control.value;
	if (!documentId) {
		return null;
	}

	const document = documentMappingStore.documentsMetaList.find(doc => doc.id === documentId);
	if (!document) {
		return null;
	}
	return document;
});

/**
 * Check if evidence is a stub (pending evidence request placeholder)
 * Stub evidence should be hidden from the UI
 */
function isStubEvidence(evidence: EvidenceWithFileEndPoint): boolean {
	return (
		evidence.fileName === "pending-evidence-request" &&
		evidence.approvalStatus === "PENDING_UPLOAD_REQUEST"
	);
}

const controlEvidences = computed<ControlEvidence>(() => {
	// In audit mode, use audit store evidences
	if (isAuditMode.value) {
		const auditEvidence = auditStore.auditControlEvidences[control.value.id];
		if (!auditEvidence) {
			return {
				totalEvidences: 0,
				totalApprovedEvidences: 0,
				totalRejectedEvidences: 0,
				totalPendingEvidences: 0,
				latestEvidence: null,
				evidence: []
			};
		}
		return auditEvidence;
	}

	// Normal mode - use application store
	const evidence = applicationStore.applicationEvidences[control.value.id];
	if (!evidence) {
		return {
			totalEvidences: 0,
			totalApprovedEvidences: 0,
			totalRejectedEvidences: 0,
			totalPendingEvidences: 0,
			latestEvidence: null,
			evidence: []
		};
	}
	return evidence;
});

const controlStatus = computed(() => {
	const appId = activeApplicationId.value;
	if (!appId) {
		return null;
	}
	const application = applicationStore.applications.find(app => app.id === appId);
	if (!application) {
		return null;
	}

	return (
		application.statements?.find(statement => statement.statementId === control.value.id)?.status ??
		null
	);
});

const filteredControlEvidences = computed((): Array<EvidenceWithFileEndPoint> => {
	const evidences = controlEvidences.value?.evidence ?? [];

	// If we have an active application (either from prop or selected), filter for that application
	if (activeApplicationId.value) {
		return evidences.filter(
			evidence => evidence.appId === activeApplicationId.value && !isStubEvidence(evidence)
		);
	}

	// No specific application - return all evidences (grouped view)
	return evidences.filter(evidence => !isStubEvidence(evidence));
});

// Group evidences by application
const groupedEvidences = computed(() => {
	const groups: Record<
		string,
		{ appId: string; appName: string; evidences: Array<EvidenceWithFileEndPoint> }
	> = {};

	filteredControlEvidences.value.forEach((evidence: EvidenceWithFileEndPoint) => {
		const appId = evidence.appId ?? "ungrouped";
		const appName =
			applicationStore.applications.find(app => app.id === evidence.appId)?.name ?? "Ungrouped";

		groups[appId] ??= {
			appId,
			appName,
			evidences: []
		};

		groups[appId].evidences.push(evidence);
	});

	return Object.values(groups);
});

// Computed properties for file type detection
const selectedEvidenceFileType = computed(() => {
	if (!selectedEvidence.value?.fileName) {
		return "other";
	}
	if (isPdfFile(selectedEvidence.value.fileName)) {
		return "pdf";
	}
	if (isImageFile(selectedEvidence.value.fileName)) {
		return "image";
	}
	return "other";
});

async function openEvidence(evidence: EvidenceWithFileEndPoint) {
	try {
		evidenceJsonContent.value = null;
		isLoading.value = true;
		errorMessage.value = null;

		if (!evidence.fileEndpoint) {
			errorMessage.value = "Evidence file endpoint is not available";
			return;
		}
		selectedEvidence.value = evidence;

		// For PDF and image files, we don't need to fetch content
		// The viewers will handle loading directly from the URL
		if (isPdfFile(evidence.fileName) || isImageFile(evidence.fileName)) {
			isLoading.value = false;
			return;
		}

		// For other file types (JSON, etc.), fetch and display in Monaco editor
		const evidenceContent = await $fetch<string>(evidence.fileEndpoint);

		if (!evidenceContent) {
			errorMessage.value = "Failed to fetch evidence content";
			isLoading.value = false;
			return;
		}
		evidenceJsonContent.value = JSON.stringify(evidenceContent, null, 2);
	} catch (error) {
		errorMessage.value = "Failed to fetch evidence content";
		logger.error("Failed to fetch evidence content", error);
	} finally {
		isLoading.value = false;
	}
}

const slideoutSize = computed(() => {
	if (groupedEvidences.value.length === 0) {
		return "custom(352px,100vh)" as const;
	}
	return "custom(1400px,100vh)" as const;
});

function selectEvidenceForTimeline(evidence: EvidenceWithFileEndPoint) {
	selectedEvidenceForTimelineId.value = evidence.id;
	// Also open AI review if available
	if (evidence.metadata?.llmReview?.reasoning) {
		selectedReasoningEvidence.value = evidence;
	} else {
		selectedReasoningEvidence.value = null;
	}
}

const closeEvidence = () => {
	selectedEvidence.value = null;
	evidenceJsonContent.value = null;
};

const isControlMarkedAsEffectiveOrIneffective = computed(() => {
	return controlStatus.value === "effective" || controlStatus.value === "ineffective";
});

const isMarkControlEffectiveDisabled = computed(() => {
	// Disable if there is no active applicationId or control id
	if (!activeApplicationId.value || !control.value.id) {
		return true;
	}

	// if (isControlMarkedAsEffectiveOrIneffective.value) {
	// 	return true;
	// }

	// Get all evidences for the current application
	const appEvidences = controlEvidences.value.evidence.filter(
		evidence => evidence.appId === activeApplicationId.value
	);

	// If no evidences exist for this application, disable the button
	if (appEvidences.length === 0) {
		return true;
	}

	// Count evidences that are either approved or rejected
	const reviewedEvidences = appEvidences.filter(
		evidence => evidence.approvalStatus === "APPROVED" || evidence.approvalStatus === "REJECTED"
	);

	// Enable button only if all evidences have been reviewed (approved or rejected)
	// and there's at least one evidence
	const evidenceWithStubRequestCount = appEvidences.filter(
		evidence => !isStubEvidence(evidence)
	).length;

	const allEvidencesReviewed = reviewedEvidences.length === evidenceWithStubRequestCount;

	return !allEvidencesReviewed;
});

const ineffectiveButtonState = computed((): "danger" | "secondary" => {
	// if (controlStatus.value === "effective") {
	// 	return "secondary";
	// }
	return "danger";
});

const effectiveButtonState = computed((): "success" | "secondary" => {
	// if (controlStatus.value === "ineffective") {
	// 	return "secondary";
	// }
	return "success";
});

const controlActionMessage = computed(() => {
	// Get application name
	const application = applicationStore.applications.find(
		app => app.id === activeApplicationId.value
	);
	const appName = application?.name ?? "this application";

	// Get all evidences for the current application
	const appEvidences = controlEvidences.value.evidence.filter(
		evidence => evidence.appId === activeApplicationId.value
	);

	// Case 1: Control already marked as effective or ineffective
	if (isControlMarkedAsEffectiveOrIneffective.value) {
		// TODO: Replace with actual actionBy and timestamp when available from API
		// For now, use the last evidence's requestor as a placeholder
		const lastEvidence = appEvidences[appEvidences.length - 1];
		const actionBy = lastEvidence?.approvedBy ?? "Unknown User";
		const actionTakenAt = lastEvidence?.approvedAt;

		const actionDate = actionTakenAt
			? new Date(actionTakenAt).toLocaleDateString("en-US", {
					month: "short",
					day: "numeric"
				})
			: null;

		const status = controlStatus.value === "effective" ? "Effective" : "Ineffective";
		return `Control marked as ${status} by ${actionBy}${actionDate ? `, ${actionDate}` : ""}`.trim();
	}

	// Case 3: No evidence has been uploaded
	if (appEvidences.length === 0) {
		return "No evidence has been uploaded to take action";
	}

	// Case 4: Evidence exists but not all are reviewed (relevant/irrelevant)
	const reviewedEvidences = appEvidences.filter(
		evidence => evidence.approvalStatus === "APPROVED" || evidence.approvalStatus === "REJECTED"
	);

	const evidenceWithStubRequestCount = appEvidences.filter(
		evidence => !isStubEvidence(evidence)
	).length;

	if (reviewedEvidences.length < evidenceWithStubRequestCount) {
		const pendingCount = evidenceWithStubRequestCount - reviewedEvidences.length;
		return `${pendingCount} piece${pendingCount !== 1 ? "s" : ""} of evidence pending review before action can be taken`;
	}

	// Case 5: All evidences reviewed, ready to mark control
	return `Mark control for ${appName} as`;
});

const evidenceActionMessage = computed(() => {
	if (!selectedEvidenceForTimeline.value) {
		return "Select an evidence to review";
	}

	const evidence = selectedEvidenceForTimeline.value;
	const { approvalStatus, fileEndpoint, approvedBy, approvedAt } = evidence;

	// Case 1: Evidence already reviewed (approved or rejected)
	if (approvalStatus === "APPROVED") {
		const reviewDate = approvedAt
			? new Date(approvedAt).toLocaleDateString("en-US", {
					month: "short",
					day: "numeric"
				})
			: null;

		return `Marked as Relevant by ${approvedBy ?? "Unknown"}, ${reviewDate ?? ""}`.trim();
	}

	if (approvalStatus === "REJECTED") {
		// TODO: Replace with actual reviewer and timestamp when available from API
		const reviewDate = approvedAt
			? new Date(approvedAt).toLocaleDateString("en-US", {
					month: "short",
					day: "numeric"
				})
			: null;

		return `Marked as Irrelevant by ${approvedBy ?? "Unknown"} ${reviewDate ? `, ${reviewDate}` : ""}`.trim();
	}

	// Case 2: No file uploaded for this evidence
	if (!fileEndpoint && !(approvalStatus === "ATTESTED")) {
		return "No file uploaded for this evidence";
	}

	// Case 5: Ready to review evidence
	return "Mark Evidence as";
});

const evidenceRelevantButtonState = computed(() => {
	if (!selectedEvidenceForTimeline.value) {
		return "secondary";
	}

	// const { approvalStatus } = selectedEvidenceForTimeline.value;

	// // If evidence is marked as irrelevant, make relevant button secondary
	// if (approvalStatus === "REJECTED") {
	// 	return "secondary";
	// }

	return "success";
});

const evidenceIrrelevantButtonState = computed(() => {
	if (!selectedEvidenceForTimeline.value) {
		return "secondary";
	}

	// const { approvalStatus } = selectedEvidenceForTimeline.value;

	// // If evidence is marked as relevant, make irrelevant button secondary
	// if (approvalStatus === "APPROVED") {
	// 	return "secondary";
	// }

	return "danger";
});

// Check if mark evidence buttons should be disabled
const isMarkEvidenceButtonDisabled = computed(() => {
	if (!selectedEvidenceForTimeline.value) {
		return true;
	}

	// if (isControlMarkedAsEffectiveOrIneffective.value) {
	// 	return true;
	// }

	// Check if evidence has a file uploaded
	const hasFile = !!selectedEvidenceForTimeline.value.fileEndpoint;
	const isAttested = selectedEvidenceForTimeline.value.approvalStatus === "ATTESTED";
	if (!hasFile && !isAttested) {
		return true;
	}

	// const { approvalStatus } = selectedEvidenceForTimeline.value;
	// Only disable if evidence is manually approved/rejected (not agent status)
	// if (approvalStatus === "APPROVED" || approvalStatus === "REJECTED") {
	// 	return true;
	// }

	// Allow manual review for agent-reviewed evidence and pending evidence
	// Remove the requestor check to allow any authorized user to mark evidence
	return false;
});

function showUploadEvidence() {
	if (!activeApplicationId.value) {
		return;
	}
	const uploadUrl = `/stances/${stanceId.value}/applications/${activeApplicationId.value}/controls/${control.value.id}/upload-evidence`;
	navigateTo(uploadUrl);
}

async function markEvidenceAsEffective() {
	if (!selectedEvidenceForTimeline.value) {
		return;
	}
	const { id } = selectedEvidenceForTimeline.value;

	try {
		isMarkingEffective.value = true;
		await $fetch<{ data: Array<unknown> }>(`/api/evidences/${id}`, {
			method: "PUT",
			body: {
				approvalStatus: "APPROVED"
			}
		});

		// Refresh evidences
		if (isAuditMode.value) {
			await auditStore.loadAuditControlEvidences();
		} else if (stanceId.value) {
			await applicationStore.getApplicationEvidenceByStanceIds({ stanceId: stanceId.value });
		}
	} catch (error) {
		logger.error("Failed to mark evidence as effective", error);
		errorMessage.value = "Failed to mark evidence as effective";
	} finally {
		isMarkingEffective.value = false;
	}
}

async function markEvidenceAsIneffective() {
	if (!selectedEvidenceForTimeline.value) {
		return;
	}

	const { id } = selectedEvidenceForTimeline.value;

	try {
		isMarkingIneffective.value = true;
		await $fetch<{ data: Array<unknown> }>(`/api/evidences/${id}`, {
			method: "PUT",
			body: {
				approvalStatus: "REJECTED"
			}
		});

		// Refresh evidences
		if (isAuditMode.value) {
			await auditStore.loadAuditControlEvidences();
		} else if (stanceId.value) {
			await applicationStore.getApplicationEvidenceByStanceIds({ stanceId: stanceId.value });
		}
	} catch (error) {
		logger.error("Failed to mark evidence as ineffective", error);
		errorMessage.value = "Failed to mark evidence as ineffective";
	} finally {
		isMarkingIneffective.value = false;
	}
}

async function markControlAs(status: "effective" | "ineffective") {
	if (!activeApplicationId.value || !control.value.id) {
		return;
	}

	try {
		isMarkingControlEffective.value = true;
		await $fetch(`/api/applications/${activeApplicationId.value}/statements/${control.value.id}`, {
			method: "PATCH",
			headers: {
				"Content-Type": "application/json"
			},
			body: {
				status
			}
		});

		// Refresh data
		if (isAuditMode.value) {
			await Promise.all([
				applicationStore.getApplicationById(activeApplicationId.value),
				auditStore.loadAuditControlEvidences()
			]);
		} else if (activeApplicationId.value && stanceId.value) {
			await Promise.all([
				applicationStore.getApplicationById(activeApplicationId.value),
				applicationStore.getApplicationEvidenceByStanceIds({
					stanceId: stanceId.value
				})
			]);
		}

		errorMessage.value = null;
	} catch (error) {
		logger.error("Failed to mark control as effective", error);
		errorMessage.value = "Failed to mark control as effective";
	} finally {
		isMarkingControlEffective.value = false;
		closeSlideout();
	}
}

// Multi-select functions
function toggleEvidenceSelection(evidence: EvidenceWithFileEndPoint, checked: boolean) {
	if (checked) {
		selectedEvidenceIds.value.add(evidence.id);
	} else {
		selectedEvidenceIds.value.delete(evidence.id);
	}
}

function isEvidenceChecked(evidenceId: string): boolean {
	return selectedEvidenceIds.value.has(evidenceId);
}

const selectedEvidencesList = computed(() => {
	return filteredControlEvidences.value.filter(evidence =>
		selectedEvidenceIds.value.has(evidence.id)
	);
});

// Check if all selected evidence items are already marked (approved or rejected)
const areAllSelectedEvidencesMarked = computed(() => {
	if (selectedEvidencesList.value.length === 0) {
		return false;
	}
	return selectedEvidencesList.value.every(
		evidence => evidence.approvalStatus === "APPROVED" || evidence.approvalStatus === "REJECTED"
	);
});

// Bulk operations
async function bulkMarkAsEffective() {
	if (selectedEvidenceIds.value.size === 0) {
		return;
	}

	try {
		isBulkMarkingEffective.value = true;

		// Call API for each selected evidence
		const promises = selectedEvidencesList.value.map(evidence =>
			$fetch<{ data: Array<unknown> }>(`/api/evidences/${evidence.id}`, {
				method: "PUT",
				body: {
					approvalStatus: "APPROVED"
				}
			})
		);

		await Promise.all(promises);

		if (stanceId.value) {
			await applicationStore.getApplicationEvidenceByStanceIds({ stanceId: stanceId.value });
		}

		// Clear selection
		selectedEvidenceIds.value.clear();
	} catch (error) {
		logger.error("Failed to bulk mark evidences as effective", error);
		errorMessage.value = "Failed to mark some evidences as effective";
	} finally {
		isBulkMarkingEffective.value = false;
	}
}

async function bulkMarkAsIneffective() {
	if (selectedEvidenceIds.value.size === 0) {
		return;
	}

	try {
		isBulkMarkingIneffective.value = true;

		// Call API for each selected evidence
		const promises = selectedEvidencesList.value.map(evidence =>
			$fetch<{ data: Array<unknown> }>(`/api/evidences/${evidence.id}`, {
				method: "PUT",
				body: {
					approvalStatus: "REJECTED"
				}
			})
		);

		await Promise.all(promises);

		if (stanceId.value) {
			applicationStore.getApplicationEvidenceByStanceIds({ stanceId: stanceId.value });
		}

		// Clear selection
		selectedEvidenceIds.value.clear();
	} catch (error) {
		logger.error("Failed to bulk mark evidences as ineffective", error);
		errorMessage.value = "Failed to mark some evidences as ineffective";
	} finally {
		isBulkMarkingIneffective.value = false;
	}
}

// Select an application (for audit mode)
function selectApplication(appId: unknown) {
	if (typeof appId !== "string") {
		return;
	}
	selectedAppId.value = appId;
	selectedEvidence.value = null;
	evidenceJsonContent.value = null;
	selectedEvidenceForTimelineId.value = null;
	selectedEvidenceIds.value.clear();
}

// Request evidence for all applications (for audit mode)
async function requestEvidenceForAllApplications() {
	if (!isAuditMode.value || controlApplications.value.length === 0) {
		return;
	}

	try {
		isRequestingAll.value = true;

		// Create evidence requests for all applications that don't have evidence
		const promises = controlApplications.value.map(app =>
			$fetch<{ data: Array<unknown> }>("/api/requests", {
				method: "POST",
				body: {
					stanceId: stanceId.value,
					controlStatementId: control.value.id,
					name: "Evidence Request",
					applicationId: app.id
				}
			})
		);

		await Promise.all(promises);

		// Refresh evidences
		await auditStore.loadAuditControlEvidences();

		logger.info("Evidence requested for all applications");
	} catch (error) {
		logger.error("Failed to request evidence for all applications", error);
		errorMessage.value = "Failed to request evidence for some applications";
	} finally {
		isRequestingAll.value = false;
	}
}

// Auto-select first evidence when slideout opens
watch(
	() => isSlideoutOpen.value,
	newValue => {
		if (newValue && groupedEvidences.value.length > 0) {
			const [firstGroup] = groupedEvidences.value;
			if (firstGroup && firstGroup.evidences.length > 0) {
				const [firstEvidence] = firstGroup.evidences;
				if (firstEvidence) {
					selectedEvidenceForTimelineId.value = firstEvidence.id;
					// Auto-open AI review if available
					if (firstEvidence.metadata?.llmReview?.reasoning) {
						selectedReasoningEvidence.value = firstEvidence;
					}
				}
			}
		}
	},
	{ immediate: true }
);

// Clear viewer state when evidence selection changes
watch(selectedEvidenceForTimelineId, () => {
	selectedEvidence.value = null;
	evidenceJsonContent.value = null;
});
</script>

<template>
	<FPopover
		v-model="isSlideoutOpen"
		placement="top-end"
		:size="slideoutSize"
		target="body"
		shadow
		overlay
		:padding="0"
		:popover-offset="{
			mainAxis: 0,
			crossAxis: 0,
			alignmentAxis: 0
		}"
		style="border-radius: 8px 0 0 8px"
	>
		<FDiv
			state="default"
			border="small solid subtle around"
			height="100%"
			direction="column"
			overflow="hidden"
			show-scrollbar
			style="border-radius: 8px 0 0 8px"
		>
			<FDiv
				height="hug-content"
				border="small solid subtle bottom"
				align="middle-left"
				padding="medium"
				gap="large"
			>
				<FDiv align="middle-left" width="fill-container" gap="large">
					<FText variant="para" size="medium" weight="medium" state="secondary">
						Control Evidence
					</FText>
					<!-- Request All button (audit mode only) -->
					<FButton
						v-if="isAuditMode"
						label="Request All"
						category="outline"
						size="small"
						icon-left="i-mail"
						:loading="isRequestingAll"
						@click="requestEvidenceForAllApplications"
					/>
				</FDiv>
				<FIcon source="i-close" size="small" clickable @click="closeSlideout"></FIcon>
			</FDiv>
			<!-- 3 Column Layout -->
			<FDiv height="fill-container" width="fill-container" direction="row">
				<!-- Column 1: Evidence List grouped by Application -->
				<FDiv
					direction="column"
					width="350px"
					height="fill-container"
					overflow="hidden"
					border="small solid subtle right"
				>
					<FDiv direction="column" overflow="scroll" show-scrollbar height="fill-container">
						<!-- Control Details -->
						<FDiv
							padding="medium medium none medium"
							direction="column"
							gap="small"
							height="hug-content"
						>
							<FDiv height="hug-content"
								><FText size="medium" weight="regular" state="subtle">Control</FText></FDiv
							>
							<FDiv>
								<FText size="medium" weight="medium" state="default">{{ control.title }}</FText>
							</FDiv>
						</FDiv>
						<FDiv
							padding="medium medium none medium"
							direction="column"
							gap="small"
							height="hug-content"
						>
							<FDiv height="hug-content"
								><FText size="medium" weight="regular" state="subtle">Control Status</FText></FDiv
							>
							<FDiv>
								<FText
									v-if="controlStatus === 'effective'"
									size="medium"
									weight="medium"
									state="success"
									>Effective</FText
								>
								<FText
									v-else-if="controlStatus === 'ineffective'"
									size="medium"
									weight="medium"
									state="danger"
									>Ineffective</FText
								>
								<FDiv v-else height="hug-content" gap="small" align="middle-left">
									<FIcon
										source="i-alert-line"
										size="small"
										clickable
										@click="closeSlideout"
									></FIcon>
									<FText size="medium" weight="medium" state="default"
										>Awaiting control owner review</FText
									>
								</FDiv>
							</FDiv>
						</FDiv>
						<FDiv
							v-if="calculatedMeta.currentAxle"
							padding="medium medium none medium"
							height="hug-content"
							direction="column"
							gap="small"
						>
							<FDiv height="hug-content">
								<FText size="medium" weight="regular" state="subtle">Control Set</FText>
							</FDiv>
							<FDiv gap="small" align="middle-left" direction="column">
								<FDiv gap="medium">
									<FDiv width="hug-content" align="top-left"
										><FIcon source="i-process" size="medium" state="danger"
									/></FDiv>
									<FDiv direction="column" gap="x-small">
										<FText size="medium" weight="medium" state="secondary">{{
											calculatedMeta?.currentAxle?.fullTitle?.split(">")[0]?.trim()
										}}</FText>
										<FText size="medium" weight="regular" state="secondary">{{
											calculatedMeta?.currentAxle?.fullTitle?.split(">").slice(1).join(">").trim()
										}}</FText>
									</FDiv>
								</FDiv>
							</FDiv>
						</FDiv>

						<FDiv
							v-if="controlDocument"
							padding="medium"
							height="hug-content"
							border="small solid subtle bottom"
							direction="column"
							gap="small"
						>
							<FDiv><FText size="medium" weight="regular" state="subtle">Document</FText></FDiv>
							<FDiv gap="small" align="middle-left" height="hug-content">
								<FDiv width="hug-content" height="hug-content" align="top-left">
									<DocumentIcon
										:publisher="controlDocument?.publisher"
										:is-published="controlDocument.state === 'PUBLISHED'"
										size="large"
									></DocumentIcon>
								</FDiv>
								<FText size="medium" weight="regular" state="secondary">{{
									controlDocument.displayName
								}}</FText>
							</FDiv>
						</FDiv>

						<!-- Application Selector (Audit Mode Only) -->
						<FDiv
							v-if="isAuditMode && controlApplications.length > 0"
							padding="medium"
							height="hug-content"
							border="small solid subtle bottom"
							direction="column"
							gap="small"
						>
							<FDiv><FText size="medium" weight="regular" state="subtle">Applications</FText></FDiv>
							<Select v-model="selectedAppId" @update:model-value="selectApplication">
								<SelectTrigger class="w-full">
									<SelectValue placeholder="Select an application to view evidence" />
								</SelectTrigger>
								<SelectContent class="z-999">
									<SelectItem v-for="app in controlApplications" :key="app.id" :value="app.id">
										<FDiv direction="row" gap="small" align="middle-left">
											<FIcon source="i-app-line" size="small" state="secondary" />
											<span>{{ app.name }}</span>
										</FDiv>
									</SelectItem>
								</SelectContent>
							</Select>
						</FDiv>

						<FDiv
							padding="medium medium medium medium"
							height="hug-content"
							align="middle-left"
							state="default"
							gap="small"
						>
							<!-- <FIcon source="i-pb-postbox-line" size="medium" state="subtle" /> -->
							<FText size="medium" weight="regular" state="subtle">Evidence</FText>

							<!-- Bulk action buttons -->
							<FDiv
								v-if="selectedEvidenceIds.size > 1"
								direction="row"
								gap="small"
								width="hug-content"
							>
								<FButton
									label="Relevant"
									category="outline"
									size="small"
									state="success"
									:loading="isBulkMarkingEffective"
									:disabled="isBulkMarkingIneffective || areAllSelectedEvidencesMarked"
									@click="bulkMarkAsEffective"
								></FButton>
								<FButton
									label="Irrelevant"
									category="outline"
									size="small"
									state="danger"
									:loading="isBulkMarkingIneffective"
									:disabled="isBulkMarkingEffective || areAllSelectedEvidencesMarked"
									@click="bulkMarkAsIneffective"
								></FButton>
							</FDiv>

							<FDiv v-if="showAddEvidence" width="hug-content" height="hug-content">
								<FButton
									label="Add"
									category="outline"
									size="small"
									@click="showUploadEvidence"
								></FButton>
							</FDiv>
						</FDiv>
						<FDiv
							v-for="group in groupedEvidences"
							:key="group.appId"
							direction="column"
							gap="none"
							padding="none none"
							height="hug-content"
							class="mb-1"
						>
							<!-- Application Header -->
							<FDiv padding="medium" border="small solid subtle bottom" gap="medium">
								<FIcon source="i-app-line" size="medium" state="secondary" />
								<FText variant="para" size="medium" weight="medium" state="default">{{
									group.appName
								}}</FText>
							</FDiv>

							<!-- Evidence List -->
							<ApplicationEvidenceListItem
								v-for="evidence in group.evidences"
								:key="evidence.id"
								:evidence="evidence"
								:is-selected="selectedEvidenceForTimeline?.id === evidence.id"
								:is-checked="isEvidenceChecked(evidence.id)"
								:show-checkbox="!isControlMarkedAsEffectiveOrIneffective"
								@click="selectEvidenceForTimeline"
								@toggle-checkbox="toggleEvidenceSelection"
							/>
						</FDiv>
						<FDiv v-if="groupedEvidences.length === 0" padding="medium">
							<FText size="small" weight="regular" state="subtle">No evidence found</FText>
						</FDiv>
					</FDiv>
				</FDiv>

				<!-- Column 2: Timeline and Control Details -->
				<FDiv
					v-if="groupedEvidences.length"
					direction="column"
					width="fill-container"
					height="fill-container"
					overflow="hidden"
					border="small solid subtle right"
				>
					<FDiv>
						<FDiv
							direction="column"
							overflow="scroll"
							show-scrollbar
							height="fill-container"
							max-width="350px"
						>
							<FDiv
								padding="medium medium none medium"
								height="hug-content"
								align="middle-left"
								gap="small"
								direction="column"
							>
								<FDiv width="hug-content" height="hug-content" padding="none medium none none"
									><FText size="medium" weight="regular" state="subtle"
										>Evidence Details</FText
									></FDiv
								>

								<FDiv gap="medium">
									<FIcon source="i-pb-postbox-line" size="medium" state="subtle" />
									<FText size="medium" weight="medium" state="default">{{
										selectedEvidenceForTimeline?.fileName
									}}</FText>
								</FDiv>
							</FDiv>

							<FDiv
								padding="large medium large medium"
								height="hug-content"
								align="middle-left"
								state="default"
								gap="x-small"
								border="small solid subtle bottom"
								width="fill-container"
							>
								<FText size="medium" weight="regular" state="subtle">Evidence Status</FText>
								<FDiv
									width="hug-content"
									height="hug-content"
									padding="none none medium none"
									gap="small"
								>
									<FDiv height="hug-content" width="fill-container"
										><FText size="medium" weight="regular" state="default">{{
											evidenceActionMessage
										}}</FText></FDiv
									>
									<FIcon source="i-edit-line" size="small" state="subtle" />
								</FDiv>

								<FDiv
									v-if="selectedEvidenceIds.size <= 1"
									width="hug-content"
									height="hug-content"
									align="middle-left"
									gap="small"
								>
									<FButton
										label="Relevant"
										category="fill"
										size="small"
										:state="evidenceRelevantButtonState"
										:loading="isMarkingEffective"
										:disabled="
											selectedEvidenceIds.size > 1 ||
											isMarkingIneffective ||
											isMarkEvidenceButtonDisabled
										"
										@click="markEvidenceAsEffective"
									/>
									<FButton
										label="Irrelevant"
										category="fill"
										size="small"
										:state="evidenceIrrelevantButtonState"
										:loading="isMarkingIneffective"
										:disabled="
											selectedEvidenceIds.size > 1 ||
											isMarkingEffective ||
											isMarkEvidenceButtonDisabled
										"
										@click="markEvidenceAsIneffective"
									/>
								</FDiv>
							</FDiv>

							<!-- Selected Evidence Details -->
							<FDiv
								v-if="selectedEvidenceForTimeline"
								direction="column"
								gap="medium"
								height="hug-content"
							>
								<FDiv direction="column" gap="none" height="hug-content">
									<FDiv
										v-if="selectedEvidenceForTimeline.fileEndpoint"
										gap="medium"
										align="middle-left"
										height="hug-content"
										width="fill-container"
										padding="medium"
										clickable
										border="small solid subtle bottom"
										:class="selectedEvidence ? 'st-grdnt-selected pl-3!' : 'pl-3!'"
										:state="selectedEvidence ? 'secondary' : 'default'"
										style="cursor: pointer"
										@click="openEvidence(selectedEvidenceForTimeline)"
									>
										<FIcon source="i-controls-line" size="medium" state="default" />
										<FText size="medium" weight="regular" state="default">View Evidence</FText>
										<FIcon source="i-caret-right" size="medium" state="subtle" class="ml-auto" />
									</FDiv>
									<FDiv
										gap="medium"
										align="middle-left"
										height="hug-content"
										width="fill-container"
										padding="medium"
										clickable
										border="small solid subtle bottom"
										:class="
											selectedEvidenceForTimeline && !selectedEvidence
												? 'st-grdnt-selected pl-3!'
												: 'pl-3!'
										"
										:state="
											selectedEvidenceForTimeline && !selectedEvidence ? 'secondary' : 'default'
										"
										@click="closeEvidence()"
									>
										<FIcon source="i-agent-line" size="medium" state="default" />
										<FText size="medium" weight="regular" state="default"
											>View Agent Analysis</FText
										>
										<FIcon source="i-caret-right" size="medium" state="subtle" class="ml-auto" />
									</FDiv>
									<a
										v-if="selectedEvidenceForTimeline.fileEndpoint"
										:href="selectedEvidenceForTimeline.fileEndpoint"
										style="text-decoration: none; width: 100%"
										@click.stop
									>
										<FDiv
											gap="medium"
											align="middle-left"
											height="hug-content"
											width="fill-container"
											padding="medium"
											clickable
											border="small solid subtle bottom"
											style="cursor: pointer"
										>
											<FIcon source="i-download-line" size="medium" state="default" />
											<FText size="medium" weight="regular" state="default">Download</FText>
											<!-- <FIcon
													source="i-caret-right"
													size="medium"
													state="subtle"
													class="ml-auto"
												/> -->
										</FDiv>
									</a>
								</FDiv>

								<!-- Timeline -->
								<FDiv padding="none">
									<ApplicationEvidenceTimeline :evidence="selectedEvidenceForTimeline" />
								</FDiv>
							</FDiv>

							<FDiv
								v-else
								padding="large"
								align="middle-center"
								direction="column"
								gap="medium"
								height="fill-container"
							>
								<FIcon source="i-info-line" size="large" state="subtle" />
								<FText size="medium" weight="regular" state="subtle"
									>Select an evidence to view timeline</FText
								>
							</FDiv>
						</FDiv>

						<FDiv border="small solid subtle left">
							<FDiv
								v-if="selectedEvidence"
								style="
									position: absolute;
									top: 0;
									left: 0;
									z-index: 100;
									height: 100%;
									width: 100%;
									background-color: var(--color-surface-default);
								"
							>
								<!-- PDF Viewer -->
								<PDFViewer
									v-if="selectedEvidenceFileType === 'pdf' && selectedEvidence.fileEndpoint"
									:src="selectedEvidence.fileEndpoint"
									:loading="isLoading"
									:error="errorMessage ?? undefined"
									height="fill-container"
									width="fill-container"
								/>

								<!-- Image Viewer -->
								<ImageViewer
									v-if="selectedEvidenceFileType === 'image' && selectedEvidence.fileEndpoint"
									:src="selectedEvidence.fileEndpoint"
									:loading="isLoading"
									:error="errorMessage ?? undefined"
									height="fill-container"
									width="fill-container"
								/>

								<!-- JSON/Text Viewer (Monaco Editor) -->
								<FDiv
									v-else-if="evidenceJsonContent"
									direction="column"
									width="100%"
									height="100%"
									state="secondary"
									overflow="scroll"
									style="background-color: #000000 !important"
								>
									<LazyMonacoEditor
										:content="evidenceJsonContent"
										language="json"
										:copy-button="true"
										:show-line-numbers="true"
										:read-only="true"
										height="100%"
									></LazyMonacoEditor>
								</FDiv>
								<FDiv
									v-else-if="isLoading"
									width="fill-container"
									height="fill-container"
									align="middle-center"
									padding="medium"
									gap="small"
								>
									<FDiv align="middle-center" width="hug-content" height="hug-content" gap="small">
										<FullPageLoader></FullPageLoader>
										<FText size="small" weight="regular">Fetching evidence content...</FText>
									</FDiv>
								</FDiv>
								<EmptyState
									v-else-if="errorMessage"
									:icon="{ source: 'i-file-missing', size: 'large' }"
									message="Failed to load evidence content"
									:subtitle="errorMessage"
								/>
							</FDiv>
							<FDiv v-else direction="column" overflow="scroll">
								<FDiv
									padding="medium"
									height="hug-content"
									align="middle-left"
									state="default"
									gap="small"
									border="small solid subtle bottom"
								>
									<FIcon source="i-info-line" size="medium" state="subtle" />
									<FText size="medium" weight="medium" state="default">AI Review</FText>
								</FDiv>
								<FDiv
									direction="column"
									overflow="scroll"
									show-scrollbar
									height="fill-container"
									width="fill-container"
									state="secondary"
								>
									<FDiv
										v-if="selectedReasoningEvidence?.metadata?.llmReview?.reasoning"
										gap="large"
										padding="medium"
										class="llm-reasoning-content"
									>
										<FDiv
											v-if="selectedReasoningEvidence.metadata?.llmReview?.relevancePercentage"
											gap="small"
											height="hug-content"
										>
											<FText weight="bold" inline>Relevance:</FText>
											<FText inline>
												{{ selectedReasoningEvidence.metadata.llmReview.relevancePercentage }}%
											</FText>
										</FDiv>
										<FMarkdown :content="selectedReasoningEvidence.metadata.llmReview.reasoning" />

										<FText variant="heading">AI Evidence Description</FText>

										<FMarkdown
											v-if="selectedReasoningEvidence.evidenceActivityDescription"
											:content="selectedReasoningEvidence.evidenceActivityDescription"
										/>
									</FDiv>
									<FDiv
										v-else
										padding="large"
										align="middle-center"
										direction="column"
										gap="medium"
										height="fill-container"
									>
										<FIcon source="i-file-missing" size="large" state="subtle" />
										<FText size="medium" weight="regular" state="subtle" align="center"
											>No AI review available for this evidence</FText
										>
									</FDiv>
								</FDiv>
							</FDiv>
						</FDiv>
					</FDiv>
				</FDiv>
			</FDiv>
			<FDiv
				v-if="applicationId && groupedEvidences.length > 0"
				height="hug-content"
				width="fill-container"
				padding="medium"
				gap="medium"
				align="middle-right"
				state="secondary"
				border="small solid default top"
			>
				<FDiv align="middle-right" width="hug-content">
					<FText size="medium" weight="medium" state="default">{{ controlActionMessage }}</FText>
				</FDiv>
				<FButton
					label="Ineffective"
					size="medium"
					:state="ineffectiveButtonState"
					:loading="isMarkingControlEffective"
					:disabled="isMarkControlEffectiveDisabled"
					@click="markControlAs('ineffective')"
				/>
				<FButton
					label="Effective"
					size="medium"
					:state="effectiveButtonState"
					:loading="isMarkingControlEffective"
					:disabled="isMarkControlEffectiveDisabled"
					@click="markControlAs('effective')"
				/>
			</FDiv>
		</FDiv>
	</FPopover>
</template>

<style lang="scss">
.monaco-editor {
	--color-surface-default: #3a3a3a !important;
}

.llm-reasoning-content {
	h1,
	h2,
	h3 {
		margin: 0 !important;
	}
}
</style>
