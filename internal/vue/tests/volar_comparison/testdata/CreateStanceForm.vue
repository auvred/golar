<script setup lang="ts">
import { ref } from "vue";
import {
	getFlag,
	useStanceStore,
	getChildRegions,
	type DepartmentTreeNodeWithLevel,
	type GeographicStanceWithLevel
} from "~/store/stance-store";
import { logger } from "@nonfx/logger";
import { useUserStore } from "~/store/user-store";
import { defaultRegionHierarchy, type HierarchyNode } from "@nonfx/foundation/geography";
import type { StanceWithEnrichedData } from "@nonfx/stance-schema";
import type { CreateStanceRequest } from "../../../../packages/stance-base/server/api/stances/index.post";
import { Input } from "@/components/ui/input";
import { Field, FieldLabel } from "@/components/ui/field";

const userStore = useUserStore();

const isOpen = defineModel<boolean>("isOpen", {
	type: Boolean,
	required: true,
	default: false
});

const props = defineProps({
	parentStance: {
		type: Object as PropType<DepartmentTreeNodeWithLevel | null>,
		required: false,
		default: null
	},

	parentGeographicStance: {
		type: Object as PropType<GeographicStanceWithLevel | null>,
		required: false,
		default: null
	},

	target: {
		type: HTMLElement,
		required: true
	},

	excludedRegions: {
		type: Array as () => Array<string>,
		default: () => []
	}
});

const { excludedRegions, target, parentStance, parentGeographicStance } = toRefs(props);

const emit = defineEmits<{
	(e: "close"): void;
	(e: "created", newStance: StanceWithEnrichedData): void;
}>();

export type RegionType = {
	regionPath: string;
	displayName: string;
	flag: string;
	isDisabled?: boolean;
};

const disabledRegions = computed(() => {
	return parentStance.value
		? ["Global", ...parentStance.value.regionStances.map(region => region.regionPath)]
		: [];
});

function buildFlatList(
	node: Record<string, HierarchyNode>,
	currentPath = "",
	result: Array<RegionType> = []
): Array<RegionType> {
	for (const [key, value] of Object.entries(node)) {
		const newPath = currentPath ? `${currentPath}.${key}` : key;

		const flag = getFlag(newPath); // Default to globe emoji if no flag found

		result.push({
			regionPath: newPath,
			displayName: value.displayName,
			flag,
			isDisabled: disabledRegions.value.includes(newPath)
		});

		buildFlatList(value.children, newPath, result);
	}

	return result;
}

const stanceStore = useStanceStore();

const isCreating = ref(false);
const showRegionSelect = ref(false);
const selectedRegion = ref<RegionType | null>(null);
const stanceType = ref<"region" | "department">("region"); // Default to region
const departmentName = ref("");
const searchTerm = ref<string | null>(null);
const formState = ref({
	region: null as RegionType | null
});

const flatGeographicRegions = computed(() => {
	// If we have a geographic parent stance, only show its children
	const geographicParent = parentGeographicStance.value;
	if (geographicParent) {
		const childRegions = getChildRegions(geographicParent.regionPath);

		// Get existing child geographic stances
		const existingChildRegions = stanceStore.allStances
			.filter(
				stance =>
					stance.departmentPath === null && // Pure geographic stance
					stance.regionPath.startsWith(`${geographicParent.regionPath}.`) && // Is child
					stance.regionPath.split(".").length === geographicParent.regionPath.split(".").length + 1 // Direct child
			)
			.map(stance => stance.regionPath);

		return childRegions.map(child => ({
			regionPath: child.regionPath,
			displayName: child.displayName,
			flag: child.flag,
			isDisabled: existingChildRegions.includes(child.regionPath)
		}));
	}

	// Otherwise show all regions (for organization stance creation)
	return buildFlatList(defaultRegionHierarchy).filter(region => {
		return !excludedRegions.value.includes(region.regionPath);
	});
});

const filteredGeographicRegions = computed(() => {
	if (searchTerm.value) {
		const lowerSearchTerm = searchTerm.value.toLowerCase();
		return flatGeographicRegions.value.filter(
			region =>
				region.displayName.toLowerCase().includes(lowerSearchTerm) ||
				region.regionPath.toLowerCase().includes(lowerSearchTerm)
		);
	}

	return flatGeographicRegions.value;
});

const handleRegionSelect = (region: RegionType) => {
	selectedRegion.value = region;
	formState.value.region = region;
	searchTerm.value = null;
	showRegionSelect.value = false;
};

// Clear search term when region selector opens
watch(showRegionSelect, newValue => {
	if (newValue) {
		searchTerm.value = null;
	}
});

const userEmail = userStore.user?.emails?.at(0) ?? "system";

function sanitizeDepartmentName(name: string): string {
	return name.trim().replace(/\s+/g, "_");
}

const handleStanceTypeChange = (event: CustomEvent) => {
	stanceType.value = event.detail.value ? "department" : "region";
	selectedRegion.value = null;
	departmentName.value = "";
	formState.value.region = null;
};

const isFormValid = computed(() => {
	if (stanceType.value === "department") {
		return departmentName.value.trim().length > 0 && parentStance.value?.departmentPath;
	} else {
		return selectedRegion.value !== null;
	}
});

const createStance = async () => {
	try {
		isCreating.value = true;

		let stanceCreateRequest: CreateStanceRequest;

		if (stanceType.value === "department") {
			if (!departmentName.value.trim()) {
				logger.error("Department name is required.");
				isCreating.value = false;
				return;
			}

			if (!parentStance.value?.departmentPath) {
				logger.error("Parent department path is required for department stance.");
				isCreating.value = false;
				return;
			}

			const sanitizedDepartmentName = sanitizeDepartmentName(departmentName.value);
			const newDepartmentPath = `${parentStance.value.departmentPath}.${sanitizedDepartmentName}`;

			stanceCreateRequest = {
				departmentPath: newDepartmentPath,
				regionPath: "Global",
				title: departmentName.value.trim(),
				owner: userEmail
			};
		} else {
			if (!selectedRegion.value?.regionPath) {
				logger.error("No region selected.");
				isCreating.value = false;
				return;
			}

			stanceCreateRequest = {
				departmentPath: parentStance.value?.departmentPath ?? null,
				regionPath: selectedRegion.value.regionPath,
				title: selectedRegion.value.displayName,
				owner: userEmail
			};
		}

		const newStance = await stanceStore.createStance(stanceCreateRequest);

		isCreating.value = false;
		emit("created", newStance);
		emit("close");
	} catch (error) {
		isCreating.value = false;
		logger.error("Failed to create stance:", error);
	}
};
</script>

<template>
	<FDialog
		v-model:is-open="isOpen"
		:overlay="true"
		:target="target"
		size="custom(400px,fit-content)"
		placement="bottom-end"
	>
		<FDialogHeader @close-dialog="$emit('close')">
			<FText variant="heading" size="small">
				{{
					showRegionSelect
						? "Select Region"
						: stanceType === "department"
							? "Create Department Stance"
							: "Create Regional Stance"
				}}
			</FText>
		</FDialogHeader>

		<FDiv direction="column" padding="none" gap="none" state="default">
			<!-- Stance Type Selection -->
			<FDiv
				v-if="!showRegionSelect"
				height="hug-content"
				direction="row"
				align="middle-left"
				gap="medium"
				padding="small medium"
				border="small solid subtle bottom"
			>
				<FDiv height="hug-content" width="90px" align="middle-left">
					<FText variant="heading" weight="regular" size="small" state="subtle">Type</FText>
				</FDiv>
				<FDiv
					direction="row"
					height="hug-content"
					width="fill-container"
					gap="small"
					align="middle-left"
				>
					<f-switch
						size="small"
						:value="stanceType === 'department'"
						state="default"
						data-qa="stance-type-department-switch"
						@input="handleStanceTypeChange"
					>
						<FDiv slot="label" padding="none">
							<FText variant="para" size="medium">Department</FText>
						</FDiv>
					</f-switch>
				</FDiv>
			</FDiv>

			<FDiv
				v-if="!showRegionSelect && stanceType === 'department'"
				height="hug-content"
				direction="row"
				align="middle-left"
				gap="medium"
				padding="small medium"
			>
				<FDiv height="hug-content" width="90px" align="middle-left">
					<Field>
						<FieldLabel for="department-name">Name</FieldLabel>
					</Field>
				</FDiv>
				<FDiv width="fill-container">
					<Field>
						<Input
							id="department-name"
							v-model="departmentName"
							placeholder="Enter department name"
							data-qa="department-name-input"
						/>
					</Field>
				</FDiv>
			</FDiv>

			<FDiv
				v-if="!showRegionSelect && stanceType === 'region'"
				height="hug-content"
				direction="row"
				align="middle-left"
				gap="medium"
				padding="small medium"
			>
				<FDiv height="hug-content" width="90px" align="middle-left">
					<FText variant="heading" weight="regular" size="small" state="subtle">Region</FText>
				</FDiv>
				<FDiv
					v-if="selectedRegion"
					align="middle-left"
					gap="medium"
					padding="small"
					@click="showRegionSelect = true"
				>
					<FDiv direction="row" gap="small" align="middle-left">
						<FText inline variant="heading" size="x-large">{{ selectedRegion.flag }}</FText>
						<FText size="small" weight="medium">{{ selectedRegion.displayName }}</FText>
					</FDiv>
					<FIcon source="i-edit" state="primary"></FIcon>
				</FDiv>
				<FDiv v-else padding="small" align="middle-center">
					<FButton
						category="outline"
						size="x-small"
						data-qa-id="stance-region-select"
						label="Select Region"
						@click="showRegionSelect = true"
					/>
				</FDiv>
			</FDiv>

			<!-- Region Selector List -->
			<FDiv v-if="showRegionSelect" direction="column">
				<FDiv padding="small" height="hug-content">
					<f-search
						data-qa="region-search"
						placeholder="Search regions..."
						@input="($e: any) => (searchTerm = $e.detail.value)"
					></f-search>
				</FDiv>
				<FDiv direction="column" height="300px" overflow="scroll">
					<FDiv
						v-for="region in filteredGeographicRegions"
						:key="region.regionPath"
						direction="row"
						padding="small"
						align="middle-left"
						width="fill-container"
						height="hug-content"
						:disabled="region.isDisabled"
						clickable
						:selected="selectedRegion?.regionPath === region.regionPath ? 'background' : 'none'"
						@click="handleRegionSelect(region)"
					>
						<FDiv
							:style="{ 'margin-left': `${region.regionPath.split('.').length * 8}px` }"
							width="fill-container"
							gap="small"
							align="middle-left"
							height="hug-content"
							overflow="hidden"
							:data-qa-id="`stance-region-option-${region.displayName}`"
						>
							<FText inline variant="heading" size="x-large">{{ region.flag }}</FText>
							<FText size="small">{{ region.displayName }}</FText>
						</FDiv>
					</FDiv>
				</FDiv>
			</FDiv>
		</FDiv>

		<FDialogFooter>
			<FDiv v-if="showRegionSelect">
				<FButton label="Cancel" category="outline" size="small" @click="showRegionSelect = false" />
			</FDiv>
			<FDiv v-else>
				<FButton
					id="create-stance"
					label="Create"
					category="fill"
					variant="round"
					size="small"
					state="primary"
					:loading="isCreating"
					:disabled="isCreating || !isFormValid"
					@click="createStance"
				/>
			</FDiv>
		</FDialogFooter>
	</FDialog>
</template>
