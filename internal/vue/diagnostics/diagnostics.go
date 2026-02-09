package vue_diagnostics

import "github.com/microsoft/typescript-go/shim/diagnostics"

var Single_file_component_can_contain_only_one_script_setup_element = &diagnostics.Message{}
var Single_file_component_can_contain_only_one_template_element = &diagnostics.Message{}
var Single_file_component_can_contain_only_one_script_element = &diagnostics.Message{}
var X_0_has_no_adjacent_v_if_or_v_else_if = &diagnostics.Message{}
var Elements_cannot_have_multiple_X_0_with_the_same_name = &diagnostics.Message{}
var Multiple_conditional_directives_cannot_coexist_on_the_same_element = &diagnostics.Message{}
var X_0_is_missing_expression = &diagnostics.Message{}
var Duplicate_X_0_call = &diagnostics.Message{}

func init() {
	diagnostics.Message_Set_code(Single_file_component_can_contain_only_one_script_setup_element, 1_000_000)
	diagnostics.Message_Set_category(Single_file_component_can_contain_only_one_script_setup_element, diagnostics.CategoryError)
	diagnostics.Message_Set_key(Single_file_component_can_contain_only_one_script_setup_element, "Single_file_component_can_contain_only_one_script_setup_element")
	diagnostics.Message_Set_text(Single_file_component_can_contain_only_one_script_setup_element, "Single file component can contain only one <script setup> element.")

	diagnostics.Message_Set_code(Single_file_component_can_contain_only_one_template_element, 1_000_001)
	diagnostics.Message_Set_category(Single_file_component_can_contain_only_one_template_element, diagnostics.CategoryError)
	diagnostics.Message_Set_key(Single_file_component_can_contain_only_one_template_element, "Single_file_component_can_contain_only_one_template_element")
	diagnostics.Message_Set_text(Single_file_component_can_contain_only_one_template_element, "Single file component can contain only one <template> element.")

	diagnostics.Message_Set_code(Single_file_component_can_contain_only_one_script_element, 1_000_002)
	diagnostics.Message_Set_category(Single_file_component_can_contain_only_one_script_element, diagnostics.CategoryError)
	diagnostics.Message_Set_key(Single_file_component_can_contain_only_one_script_element, "Single_file_component_can_contain_only_one_script_element")
	diagnostics.Message_Set_text(Single_file_component_can_contain_only_one_script_element, "Single file component can contain only one <script> element.")

	diagnostics.Message_Set_code(X_0_has_no_adjacent_v_if_or_v_else_if, 1_000_003)
	diagnostics.Message_Set_category(X_0_has_no_adjacent_v_if_or_v_else_if, diagnostics.CategoryError)
	diagnostics.Message_Set_key(X_0_has_no_adjacent_v_if_or_v_else_if, "X_0_has_no_adjacent_v_if_or_v_else_if")
	diagnostics.Message_Set_text(X_0_has_no_adjacent_v_if_or_v_else_if, "{0} has no adjacent v-if or v-else-if.")

	diagnostics.Message_Set_code(Elements_cannot_have_multiple_X_0_with_the_same_name, 1_000_003)
	diagnostics.Message_Set_category(Elements_cannot_have_multiple_X_0_with_the_same_name, diagnostics.CategoryError)
	diagnostics.Message_Set_key(Elements_cannot_have_multiple_X_0_with_the_same_name, "Elements_cannot_have_multiple_X_0_with_the_same_name")
	diagnostics.Message_Set_text(Elements_cannot_have_multiple_X_0_with_the_same_name, "Elements cannot have multiple {0} with the same name.")

	diagnostics.Message_Set_code(Multiple_conditional_directives_cannot_coexist_on_the_same_element, 1_000_004)
	diagnostics.Message_Set_category(Multiple_conditional_directives_cannot_coexist_on_the_same_element, diagnostics.CategoryError)
	diagnostics.Message_Set_key(Multiple_conditional_directives_cannot_coexist_on_the_same_element, "Multiple_conditional_directives_cannot_coexist_on_the_same_element")
	diagnostics.Message_Set_text(Multiple_conditional_directives_cannot_coexist_on_the_same_element, "Multiple conditional directives cannot coexist on the same element.")

	diagnostics.Message_Set_code(X_0_is_missing_expression, 1_000_005)
	diagnostics.Message_Set_category(X_0_is_missing_expression, diagnostics.CategoryError)
	diagnostics.Message_Set_key(X_0_is_missing_expression, "X_0_is_missing_expression")
	diagnostics.Message_Set_text(X_0_is_missing_expression, "{0} is missing expression.")

	diagnostics.Message_Set_code(Duplicate_X_0_call, 1_000_006)
	diagnostics.Message_Set_category(Duplicate_X_0_call, diagnostics.CategoryError)
	diagnostics.Message_Set_key(Duplicate_X_0_call, "Duplicate_X_0_call")
	diagnostics.Message_Set_text(Duplicate_X_0_call, "Duplicate {0} call.")
}
