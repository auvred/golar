<script setup lang="ts">
import { computed, ref, onMounted, useSlots } from "vue";
import FIcon, { type FIconSize } from "./../f-icon/f-icon.vue";
import FDiv from "./../f-div/f-div.vue";
import FCounter from "./../f-counter/f-counter.vue";
import Loader from "../loader.vue";
import {
	getCustomFillColor,
	getTextContrast,
	LightenDarkenColor,
	validateHTMLColor,
	validateHTMLColorName
} from "./../../utils";
import FPopover from "../f-popover/f-popover.vue";
import type { FPopOverOffset } from "@nonfx/flow-core/dist/src/components/f-popover/f-popover";

type ButtonState =
	| "primary"
	| "neutral"
	| "success"
	| "secondary"
	| "warning"
	| "danger"
	| "inherit"
	| "highlight"
	| `custom, ${string}`;

type ButtonAction = string | (() => unknown);

export type FButtonProps = {
	label: string;
	category?: "fill" | "outline" | "transparent" | "packed";
	size?: "large" | "medium" | "small" | "x-small";
	state?: ButtonState;
	variant?: "round" | "curved" | "block";
	iconLeft?: string;
	iconRight?: string;
	counter?: number;
	loading?: boolean;
	disabled?: boolean;
	selectedAction?: ButtonAction;
	labelWrap?: boolean;
	qaId?: string;
	type?: "button" | "submit" | "reset";
};

const props = withDefaults(defineProps<FButtonProps>(), {
	category: "fill",
	size: "medium",
	state: "primary",
	variant: "round",
	loading: false,
	disabled: false,
	labelWrap: false
});

const emit = defineEmits<{
	(e: "dropdown-open"): void;
	(e: "click", event: MouseEvent): void;
}>();

const slots = useSlots();
const fill = ref("");

const isActionsOpen = ref(false);
const actionsTarget = ref<HTMLElement | undefined>(undefined);
const popOverOffset = ref<FPopOverOffset | undefined>(undefined);

// Computed Properties
const hasActions = computed(() => !!slots.default);

const counterSize = computed(() => {
	if (props.size === "small") {
		return "medium";
	}
	if (props.size === "x-small") {
		return "small";
	}
	return props.size;
});

const iconSize = computed((): FIconSize => {
	const sizeMap: Record<string, FIconSize> = {
		large: "medium",
		medium: "small",
		small: "x-small",
		"x-small": "x-small"
	};
	return sizeMap[props.size] ?? "small";
});

const textColor = computed(() => {
	return getTextContrast(fill.value) === "dark-text" ? "#202a36" : "#fcfcfd";
});

const iconClasses = computed(() => ({
	"fill-button-surface": props.category === "fill" && !fill.value,
	"fill-button-surface-light":
		fill.value && props.category === "fill" && getTextContrast(fill.value) === "light-text",
	"fill-button-surface-dark":
		fill.value && props.category === "fill" && getTextContrast(fill.value) === "dark-text"
}));

const counterClasses = computed(() => ({
	"fill-button-surface": !fill.value && props.category === "fill",
	"fill-button-surface-light":
		props.category === "fill" && fill.value && getTextContrast(fill.value) === "light-text",
	"fill-button-surface-dark":
		props.category === "fill" && fill.value && getTextContrast(fill.value) === "dark-text"
}));

const buttonClasses = computed(() => ({
	"custom-loader": fill.value,
	"custom-hover": fill.value && props.category === "fill",
	"has-options": hasActions.value
}));

const styles = computed(() => {
	if (!fill.value) {
		return "";
	}

	if (props.loading) {
		if (props.category === "fill") {
			return `
        background-color: ${LightenDarkenColor(fill.value, -150)};
        border: 1px solid ${LightenDarkenColor(fill.value, -150)};
        color: transparent;
        fill: ${fill.value}
      `;
		} else if (props.category === "outline") {
			return `
        background: transparent;
        border: 1px solid ${fill.value};
        fill: ${fill.value};
      `;
		} else {
			return `
        background: transparent;
        border: none;
        fill: ${fill.value};
      `;
		}
	}

	if (props.category === "fill") {
		return `
      background: ${fill.value};
      border: 1px solid ${fill.value};
      color: ${textColor.value}
    `;
	} else if (props.category === "outline") {
		return `
      background: transparent;
      border: 1px solid ${fill.value};
      color: ${fill.value}
    `;
	} else {
		return `
      background: transparent;
      border: none;
      color: ${fill.value}
    `;
	}
});

// Methods
const validateProperties = () => {
	if (!props.label) {
		throw new Error("f-button: label is mandatory field");
	}
	if (
		props.state.includes("custom") &&
		fill.value &&
		!validateHTMLColor(fill.value) &&
		!validateHTMLColorName(fill.value)
	) {
		throw new Error("f-button: enter correct color-name or hex-color-code");
	}
};

const handleButtonActions = (event: MouseEvent) => {
	event.stopPropagation();

	if (isActionsOpen.value) {
		isActionsOpen.value = false;
		return;
	}

	emit("dropdown-open");

	actionsTarget.value = event.currentTarget as HTMLElement;
	popOverOffset.value = {
		mainAxis: 4,
		crossAxis: 0,
		alignmentAxis: 0
	};
	isActionsOpen.value = true;
};

const closeButtonActions = () => {
	isActionsOpen.value = false;
};

// Lifecycle
onMounted(() => {
	validateProperties();
	fill.value = getCustomFillColor(props.state);
});
</script>

<template>
	<FDiv
		:width="variant === 'block' ? 'fill-container' : 'hug-content'"
		height="hug-content"
		overflow="hidden"
	>
		<button
			:class="buttonClasses"
			:style="styles"
			:data-qa-id="qaId"
			class="f-button"
			v-bind="props"
			@click="$emit('click', $event)"
		>
			<template v-if="loading">
				<Loader />
			</template>

			<FIcon
				v-if="iconLeft"
				:data-qa-icon-left="iconLeft"
				:source="iconLeft"
				:state="state"
				:class="['left-icon', iconClasses]"
				:size="iconSize"
				clickable
			/>

			{{ label }}

			<FIcon
				v-if="iconRight"
				:data-qa-icon-right="iconRight"
				:source="iconRight"
				:state="state"
				:class="['right-icon', iconClasses]"
				:size="iconSize"
				clickable
			/>

			<FCounter
				v-if="counter"
				:data-qa-counter="counter"
				:state="state"
				:size="counterSize"
				:label="counter"
				:class="counterClasses"
			/>
		</button>

		<template v-if="hasActions">
			<span
				class="options-wrapper"
				:category="category"
				:size="size"
				:state="state"
				:variant="variant"
				@click="handleButtonActions"
			>
				<FIcon
					:class="iconClasses"
					:state="state"
					:size="iconSize"
					:source="isActionsOpen ? 'i-chevron-up' : 'i-chevron-down'"
				/>
			</span>

			<FPopover
				id="f-button-actions"
				v-model="isActionsOpen"
				:target="actionsTarget"
				:popover-offset="popOverOffset"
				size="hug-content"
				placement="bottom-end"
				:overlay="false"
				shadow
				@overlay-click="closeButtonActions"
				@click.stop
			>
				<FDiv @click="closeButtonActions">
					<slot></slot>
				</FDiv>
			</FPopover>
		</template>
	</FDiv>
</template>

<style lang="scss">
@use "sass:map";
@use "sass:math";
@use "./../../mixins/scss/mixins" as mixins;

/**
$states map created to hold values of different states
**/
$states: (
	"primary": (
		"background": var(--color-primary-default),
		"hover": var(--color-primary-default-hover),
		"loading": var(--color-primary-surface)
	),
	"neutral": (
		"background": var(--color-neutral-secondary),
		"hover": var(--color-neutral-default-hover),
		"loading": var(--color-surface-tertiary)
	),
	"secondary": (
		"background": var(--color-text-secondary),
		"hover": var(--color-text-secondary-hover),
		"loading": var(--color-surface-tertiary)
	),
	"success": (
		"background": var(--color-success-default),
		"hover": var(--color-success-default-hover),
		"loading": var(--color-success-surface)
	),
	"warning": (
		"background": var(--color-warning-default),
		"hover": var(--color-warning-default-hover),
		"loading": var(--color-warning-surface)
	),
	"danger": (
		"background": var(--color-danger-default),
		"hover": var(--color-danger-default-hover),
		"loading": var(--color-danger-surface)
	),
	"highlight": (
		"background": var(--color-highlight-default),
		"hover": var(--color-highlight-default-hover),
		"loading": var(--color-highlight-surface)
	),
	"inherit": (
		"background": var(--color-inherit-button),
		"hover": var(--color-inherit-button-hover),
		"loading": var(--color-inherit-button-loader)
	)
);

// $sizes will all values related to different sizes
$sizes: (
	"x-small": (
		"height": 20px,
		"padding": 0px 8px,
		"options-padding": 0px 4px,
		"fontSize": 10px,
		"gap": 4px,
		"loaderSize": 12px
	),
	"small": (
		"height": 28px,
		"padding": 0px 12px,
		"options-padding": 0px 8px,
		"fontSize": 12px,
		"gap": 8px,
		"loaderSize": 16px
	),
	"medium": (
		"height": 36px,
		"padding": 0px 16px,
		"options-padding": 0px 12px,
		"fontSize": 14px,
		"gap": 12px,
		"loaderSize": 20px
	),
	"large": (
		"height": 44px,
		"padding": 0px 20px,
		"options-padding": 0px 16px,
		"fontSize": 16px,
		"gap": 16px,
		"loaderSize": 24px
	)
);

// inner element used to apply all styles
.f-button {
	@include mixins.base();

	touch-action: manipulation;
	user-select: none;
	height: fit-content;
	display: inline-flex;
	flex: 0 0 auto;
	vertical-align: top;

	cursor: pointer;
	display: flex;
	align-items: center;
	justify-content: center;
	font-weight: bold;
	text-transform: uppercase;
	color: var(--color-surface-default);

	vertical-align: top;
	width: fit-content;
	position: relative;

	&[variant="block"] {
		display: flex;
		flex: 1 1 auto;
		width: 100%;
	}
	&[disabled] {
		@include mixins.disabled();
	}

	&[size="x-small"] {
		--input-border-radius-round: 10px;
	}
	&[size="small"] {
		--input-border-radius-round: 14px;
	}
	&[size="medium"] {
		--input-border-radius-round: 18px;
	}
	&[size="large"] {
		--input-border-radius-round: 22px;
	}

	&.custom-loader {
		&[loading="true"] {
			svg {
				fill: inherit !important;
				stroke: none;
				*[fill^="var(--color-primary-default)"] {
					fill: inherit !important;
				}
			}
		}
	}

	&.custom-hover {
		&:hover {
			opacity: 0.9;
		}
	}

	@each $state, $color in $states {
		&[state="#{$state}"][category="fill"] {
			background-color: map.get($color, "background");
			border: 1px solid map.get($color, "background"); // added to match width of outline category
			// hover changes background
			&:hover {
				background-color: map.get($color, "hover");
				border: 1px solid map.get($color, "hover");
			}
			// if loading attribute specified then change background, border, color and svg
			&[loading="true"] {
				background-color: map.get($color, "loading");
				border: 1px solid map.get($color, "loading");
				color: transparent;
				svg path.loader-fill {
					fill: map.get($color, "background");
				}
			}
		}

		// outline category states styles, where backgrounds are transparent
		&[state="#{$state}"][category="outline"] {
			background-color: transparent;
			border: 1px solid map.get($color, "background");
			color: map.get($color, "background");
			&:hover {
				border: 1px solid map.get($color, "hover");
				color: map.get($color, "hover");
			}

			&[loading="true"] {
				border: 1px solid map.get($color, "background");
				color: transparent;
				svg path.loader-fill {
					fill: map.get($color, "background");
				}
			}
		}
		//transparent category with transparent background and border
		&[state="#{$state}"][category="transparent"],
		&[state="#{$state}"][category="packed"] {
			background-color: transparent;
			border: 1px solid transparent;
			color: map.get($color, "background");
			&:hover {
				border: 1px solid transparent;
				color: map.get($color, "hover");
			}
			&[loading="true"] {
				color: transparent;
				svg path.loader-fill {
					fill: map.get($color, "background");
				}
			}
		}
	}
	&.has-options {
		border-top-right-radius: 0px !important;
		border-bottom-right-radius: 0px !important;
	}
	/**
  * Iterating over sizes with padding, fontsize, height
  **/
	@each $size, $value in $sizes {
		&[size="#{$size}"] {
			font-size: map.get($value, "fontSize");
			&[variant="round"] {
				border-radius: var(--input-border-radius-round);
			}
			&[category="packed"] {
				padding: 0px;
			}
			&:not([category="packed"]) {
				height: map.get($value, "height");
				padding: map.get($value, "padding");
				gap: map.get($value, "padding");
			}

			&.has-options {
				padding-right: map.get($value, "gap") !important;
			}

			&[loading="true"] {
				svg {
					height: map.get($value, "loaderSize");
				}
			}
		}
	}

	// if variant is curved then apply 4px border radius
	&[variant="curved"] {
		border-radius: var(--input-border-radius-curved);
	}

	&[variant="block"] {
		display: flex;
		flex: 0 1 auto;
		width: 100%;
	}
	// disabled button will use `disabled()` mixins
	&[disabled] {
		opacity: 1;
		pointer-events: none;
		@include mixins.disabled();
	}
	// loading use `rotate` mixins
	&[loading="true"] {
		pointer-events: none;
		@include mixins.rotate("svg");
		svg {
			position: absolute;
			top: unset;
		}
	}

	&:focus-visible,
	&:focus,
	button:focus-visible,
	button:focus {
		outline: none;
	}
}

.options-wrapper {
	display: flex;
	align-items: center;
	justify-content: center;
	box-sizing: border-box;
	cursor: pointer;
	//

	@each $state, $color in $states {
		&[state="#{$state}"][category="fill"] {
			background-color: map.get($color, "background");
			border: 1px solid map.get($color, "background"); // added to match width of outline category
			border-left: 1px solid var(--color-border-default) !important;
			// hover changes background
			&:hover {
				background-color: map.get($color, "hover");
				border: 1px solid map.get($color, "hover");
			}
		}

		// outline category states styles, where backgrounds are transparent
		&[state="#{$state}"][category="outline"] {
			background-color: transparent;
			border: 1px solid map.get($color, "background");
			border-left: 1px solid transparent !important;
			color: map.get($color, "background");
			&:hover {
				border: 1px solid map.get($color, "hover");
				color: map.get($color, "hover");
			}
		}
		//transparent category with transparent background and border
		&[state="#{$state}"][category="transparent"],
		&[state="#{$state}"][category="packed"] {
			background-color: transparent;
			border: 1px solid transparent;
			color: map.get($color, "background");
			&:hover {
				border: 1px solid transparent;
				color: map.get($color, "hover");
			}
		}
	}

	@each $size, $value in $sizes {
		&[size="#{$size}"] {
			&[variant="round"] {
				border-top-right-radius: var(--input-border-radius-round);
				border-bottom-right-radius: var(--input-border-radius-round);
			}
			&[category="packed"] {
				padding: 0px;
			}
			&:not([category="packed"]) {
				height: map.get($value, "height");
				padding: map.get($value, "options-padding");
			}
		}
	}
}
</style>
