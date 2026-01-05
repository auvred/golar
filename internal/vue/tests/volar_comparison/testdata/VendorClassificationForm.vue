<script setup lang="ts">
import { Input } from "@/components/ui/input";
import { Field, FieldLabel, FieldError } from "@/components/ui/field";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle
} from "@/components/ui/dialog";
import { Plus, X } from "lucide-vue-next";
import { logger } from "@nonfx/logger";
import { toast } from "vue-sonner";
import { useForm } from "@tanstack/vue-form";
import { z } from "zod";
import type { ClassificationDimensionWithOptions } from "@nonfx/stance-schema";

type ClassificationOption = {
	name: string;
	slug: string;
	order: string;
	description?: string;
};

type Dimension = Partial<
	Omit<ClassificationDimensionWithOptions, "createdAt" | "updatedAt" | "createdBy" | "updatedBy">
>;

const props = defineProps<{
	open: boolean;
	isEdit?: boolean;
	title: string;
	description: string;
	submitLabel?: string;
	dimension?: Dimension;
}>();

const emit = defineEmits<{
	"update:open": [value: boolean];
	success: [];
}>();

// Zod schema for validation
const formSchema = z.object({
	name: z.string().min(1, "Dimension name is required").max(255, "Dimension name is too long"),
	description: z.string().max(1000, "Description is too long").optional().or(z.literal("")),
	order: z.string().min(1, "Order is required"),
	options: z
		.array(
			z.object({
				name: z.string().min(1, "Option name is required"),
				slug: z
					.string()
					.min(1, "Slug is required")
					.regex(
						/^[A-Z][a-zA-Z0-9_]*$/,
						"Slug must start with uppercase letter and contain only letters, numbers, and underscores"
					),
				order: z.string().min(1, "Order is required"),
				description: z.string().max(1000, "Description is too long").optional().or(z.literal(""))
			})
		)
		.min(1, "At least one option is required")
});

// TanStack Form
const form = useForm({
	defaultValues: {
		name: props.dimension?.name ?? "",
		description: props.dimension?.description ?? "",
		order: String(props.dimension?.order ?? 1),
		options: (props.dimension?.options ?? []).map(opt => ({
			name: opt.name,
			slug: opt.slug,
			order: String(opt.order),
			description: opt.description ?? ""
		})) as Array<ClassificationOption>
	},
	validators: {
		onChange: formSchema
	},
	onSubmit: async ({ value }) => {
		try {
			if (props.isEdit && props.dimension?.id) {
				// Update existing dimension
				await $fetch(`/api/admin/classification-dimensions/${props.dimension.id}`, {
					method: "PUT",
					body: {
						name: value.name,
						description: value.description ?? undefined,
						order: Number.parseInt(value.order, 10),
						options: value.options.map(opt => ({
							name: opt.name,
							slug: opt.slug,
							order: Number.parseInt(opt.order, 10),
							description: opt.description ?? undefined
						}))
					}
				});
				toast.success("Classification dimension updated successfully!");
			} else {
				// Create new dimension
				await $fetch("/api/admin/classification-dimensions", {
					method: "POST",
					body: {
						name: value.name,
						description: value.description ?? undefined,
						order: Number.parseInt(value.order, 10),
						options: value.options.map(opt => ({
							name: opt.name,
							slug: opt.slug,
							order: Number.parseInt(opt.order, 10),
							description: opt.description ?? undefined
						}))
					}
				});
				toast.success("Classification dimension created successfully!");
			}

			// Emit success event to parent
			emit("success");
			// Close dialog
			emit("update:open", false);
		} catch (error) {
			logger.error(
				`Failed to ${props.isEdit ? "update" : "create"} classification dimension:`,
				error
			);
			toast.error(
				`Failed to ${props.isEdit ? "update" : "create"} classification dimension. Please try again.`
			);
		}
	}
});

// Watch for prop changes to reset form when opening dialog
watch(
	() => props.open,
	isOpen => {
		if (isOpen) {
			form.reset();
			form.setFieldValue("name", props.dimension?.name ?? "");
			form.setFieldValue("description", props.dimension?.description ?? "");
			form.setFieldValue("order", String(props.dimension?.order ?? 1));
			form.setFieldValue(
				"options",
				(props.dimension?.options ?? []).map(opt => ({
					name: opt.name,
					slug: opt.slug,
					order: String(opt.order),
					description: opt.description ?? ""
				}))
			);
		}
	}
);

function addNewOptionRow() {
	const currentOptions = form.getFieldValue("options") ?? [];
	form.setFieldValue("options", [
		...currentOptions,
		{
			name: "",
			slug: "",
			order: String(currentOptions.length + 1),
			description: ""
		}
	]);
}

function removeOptionFromForm(index: number) {
	const currentOptions = form.getFieldValue("options") ?? [];
	const newOptions = [...currentOptions];
	newOptions.splice(index, 1);
	form.setFieldValue("options", newOptions);
}

// Helper function to check if field is invalid
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function isInvalid(field: any) {
	const hasAttemptedSubmit = form.state.submissionAttempts > 0;
	return (field.state.meta.isTouched || hasAttemptedSubmit) && !field.state.meta.isValid;
}
</script>

<template>
	<Dialog :open="open" @update:open="val => emit('update:open', val)">
		<DialogContent class="max-h-[90vh] max-w-3xl overflow-hidden">
			<DialogHeader>
				<DialogTitle>{{ title }}</DialogTitle>
				<DialogDescription>{{ description }}</DialogDescription>
			</DialogHeader>

			<!-- Form Content -->
			<form
				@submit="
					e => {
						e.preventDefault();
						e.stopPropagation();
						form.handleSubmit();
					}
				"
			>
				<div class="max-h-[calc(90vh-180px)] space-y-4 overflow-y-auto pr-2">
					<!-- Dimension Name Field ---->
					<form.Field name="name">
						<template #default="{ field }">
							<Field :data-invalid="isInvalid(field)">
								<FieldLabel :for="field.name">Dimension Name</FieldLabel>
								<Input
									:id="field.name"
									:name="field.name"
									:model-value="field.state.value"
									placeholder="e.g., Criticality, DataType"
									:aria-invalid="isInvalid(field)"
									autocomplete="off"
									@blur="field.handleBlur"
									@input="field.handleChange($event.target.value)"
								/>
								<p class="text-muted-foreground text-xs">
									This dimension will be used as a classification category (e.g.,
									Vendors.Criticality.Critical)
								</p>
								<FieldError v-if="isInvalid(field)" :errors="field.state.meta.errors" />
							</Field>
						</template>
					</form.Field>

					<!-- Dimension Description Field -->
					<form.Field name="description">
						<template #default="{ field }">
							<Field :data-invalid="isInvalid(field)">
								<FieldLabel :for="field.name">Description (Optional)</FieldLabel>
								<Textarea
									:id="field.name"
									:name="field.name"
									:model-value="field.state.value"
									placeholder="Describe this classification dimension"
									:aria-invalid="isInvalid(field)"
									rows="2"
									@blur="field.handleBlur"
									@input="field.handleChange($event.target.value)"
								/>
								<FieldError v-if="isInvalid(field)" :errors="field.state.meta.errors" />
							</Field>
						</template>
					</form.Field>

					<!-- Dimension Order Field -->
					<form.Field name="order">
						<template #default="{ field }">
							<Field :data-invalid="isInvalid(field)">
								<FieldLabel :for="field.name">Hierarchy Order</FieldLabel>
								<Input
									:id="field.name"
									:name="field.name"
									:model-value="field.state.value"
									type="number"
									min="1"
									placeholder="e.g., 1, 2, 3"
									:aria-invalid="isInvalid(field)"
									autocomplete="off"
									@blur="field.handleBlur"
									@input="field.handleChange($event.target.value)"
								/>
								<p class="text-muted-foreground text-xs">
									Determines position in vendor path (1 = first level, 2 = second level, etc.)
								</p>
								<FieldError v-if="isInvalid(field)" :errors="field.state.meta.errors" />
							</Field>
						</template>
					</form.Field>

					<!-- Classification Options Field -->
					<form.Field name="options">
						<template #default="{ field }">
							<Field :data-invalid="isInvalid(field)">
								<div class="flex items-center justify-between">
									<FieldLabel>Classification Options</FieldLabel>
									<Button type="button" variant="outline" size="sm" @click="addNewOptionRow">
										<Plus class="mr-2 h-4 w-4" />
										Add Option
									</Button>
								</div>

								<FieldError v-if="isInvalid(field)" :errors="field.state.meta.errors" />

								<div
									v-if="!field.state.value || field.state.value.length === 0"
									class="rounded-md border border-dashed p-8"
								>
									<p class="text-muted-foreground text-center text-sm">
										No options added. Click "Add Option" to get started.
									</p>
								</div>

								<div v-else class="space-y-3">
									<div
										v-for="(option, idx) in field.state.value"
										:key="idx"
										class="bg-muted/30 space-y-3 rounded-lg border p-4"
									>
										<div class="flex items-start justify-between gap-3">
											<div class="flex-1 space-y-3">
												<!-- Option Name -->
												<form.Field :name="`options[${idx}].name`">
													<template #default="{ field: nameField }">
														<Field :data-invalid="isInvalid(nameField)">
															<FieldLabel :for="`option-name-${idx}`" class="text-xs"
																>Option Name</FieldLabel
															>
															<Input
																:id="`option-name-${idx}`"
																:model-value="option.name"
																placeholder="e.g., Critical, PII"
																:aria-invalid="isInvalid(nameField)"
																@blur="nameField.handleBlur"
																@input="
																	(e: Event) => {
																		const options = [...field.state.value];
																		const currentOption = options[idx];
																		options[idx] = {
																			name: (e.target as HTMLInputElement).value,
																			slug: currentOption?.slug ?? '',
																			order: currentOption?.order ?? String(idx + 1),
																			description: currentOption?.description ?? ''
																		};
																		field.handleChange(options);
																		nameField.handleChange((e.target as HTMLInputElement).value);
																	}
																"
															/>
															<FieldError
																v-if="isInvalid(nameField)"
																:errors="nameField.state.meta.errors"
															/>
														</Field>
													</template>
												</form.Field>

												<!-- Option Slug -->
												<form.Field :name="`options[${idx}].slug`">
													<template #default="{ field: slugField }">
														<Field :data-invalid="isInvalid(slugField)">
															<FieldLabel :for="`option-slug-${idx}`" class="text-xs"
																>Slug (for URL paths)</FieldLabel
															>
															<Input
																:id="`option-slug-${idx}`"
																:model-value="option.slug"
																placeholder="e.g., Critical, PII"
																:aria-invalid="isInvalid(slugField)"
																@blur="slugField.handleBlur"
																@input="
																	(e: Event) => {
																		const options = [...field.state.value];
																		const currentOption = options[idx];
																		options[idx] = {
																			name: currentOption?.name ?? '',
																			slug: (e.target as HTMLInputElement).value,
																			order: currentOption?.order ?? String(idx + 1),
																			description: currentOption?.description ?? ''
																		};
																		field.handleChange(options);
																		slugField.handleChange((e.target as HTMLInputElement).value);
																	}
																"
															/>
															<p class="text-muted-foreground text-xs">
																Must start with uppercase letter (e.g., Critical, NonCritical)
															</p>
															<FieldError
																v-if="isInvalid(slugField)"
																:errors="slugField.state.meta.errors"
															/>
														</Field>
													</template>
												</form.Field>

												<!-- Option Order -->
												<form.Field :name="`options[${idx}].order`">
													<template #default="{ field: orderField }">
														<Field :data-invalid="isInvalid(orderField)">
															<FieldLabel :for="`option-order-${idx}`" class="text-xs"
																>Order</FieldLabel
															>
															<Input
																:id="`option-order-${idx}`"
																:model-value="option.order"
																type="number"
																min="1"
																placeholder="e.g., 1, 2, 3"
																:aria-invalid="isInvalid(orderField)"
																@blur="orderField.handleBlur"
																@input="
																	(e: Event) => {
																		const options = [...field.state.value];
																		const currentOption = options[idx];
																		options[idx] = {
																			name: currentOption?.name ?? '',
																			slug: currentOption?.slug ?? '',
																			order: (e.target as HTMLInputElement).value,
																			description: currentOption?.description ?? ''
																		};
																		field.handleChange(options);
																		orderField.handleChange((e.target as HTMLInputElement).value);
																	}
																"
															/>
															<p class="text-muted-foreground text-xs">
																Lower order = more stringent (used for vendor classification
																priority)
															</p>
															<FieldError
																v-if="isInvalid(orderField)"
																:errors="orderField.state.meta.errors"
															/>
														</Field>
													</template>
												</form.Field>

												<!-- Option Description -->
												<form.Field :name="`options[${idx}].description`">
													<template #default="{ field: descriptionField }">
														<Field :data-invalid="isInvalid(descriptionField)">
															<FieldLabel :for="`option-description-${idx}`" class="text-xs"
																>Description (Optional)</FieldLabel
															>
															<Textarea
																:id="`option-description-${idx}`"
																:model-value="option.description"
																placeholder="Describe when this option applies"
																:aria-invalid="isInvalid(descriptionField)"
																class="min-h-[60px]"
																@blur="descriptionField.handleBlur"
																@input="
																	(e: Event) => {
																		const options = [...field.state.value];
																		const currentOption = options[idx];
																		options[idx] = {
																			name: currentOption?.name ?? '',
																			slug: currentOption?.slug ?? '',
																			order: currentOption?.order ?? String(idx + 1),
																			description: (e.target as HTMLTextAreaElement).value
																		};
																		field.handleChange(options);
																		descriptionField.handleChange(
																			(e.target as HTMLTextAreaElement).value
																		);
																	}
																"
															/>
															<FieldError
																v-if="isInvalid(descriptionField)"
																:errors="descriptionField.state.meta.errors"
															/>
														</Field>
													</template>
												</form.Field>
											</div>

											<Button
												type="button"
												variant="ghost"
												size="icon"
												@click="removeOptionFromForm(idx)"
											>
												<X class="h-4 w-4" />
											</Button>
										</div>
									</div>
								</div>
							</Field>
						</template>
					</form.Field>
				</div>

				<DialogFooter class="mt-4">
					<Button type="button" variant="outline" @click="emit('update:open', false)"
						>Cancel</Button
					>
					<Button type="submit" :disabled="form.state.isSubmitting">
						{{ submitLabel || (isEdit ? "Update" : "Create") }}
					</Button>
				</DialogFooter>
			</form>
		</DialogContent>
	</Dialog>
</template>
