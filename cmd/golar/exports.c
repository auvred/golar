#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>

#ifdef GOLAR_NAPI_DYNAMIC
#include <dlfcn.h>
#include <pthread.h>
#endif

#define NAPI_VERSION 6
#include "../../napi-include/node_api.h"

#include "exports.h"

#ifdef GOLAR_NAPI_DYNAMIC
#define X(NAME) static __typeof__(&NAME) golar_##NAME##_ptr;
#include "napi_dynamic_symbols.inc"
#undef X

static int golar_resolve_symbol(void **target, const char *name) {
	dlerror();
	*target = dlsym(RTLD_DEFAULT, name);
	const char *err = dlerror();
	if (err != NULL) {
		fprintf(stderr, "golar.node: failed to resolve %s: %s\n", name, err);
		return 0;
	}
	return 1;
}

static bool golar_init_napi_symbols_once_ok = false;
static void golar_init_napi_symbols_once() {
#define X(NAME) \
	if (!golar_resolve_symbol((void **)&golar_##NAME##_ptr, #NAME)) { \
		return; \
	}
#include "napi_dynamic_symbols.inc"
#undef X
	golar_init_napi_symbols_once_ok = true;
}

static pthread_once_t golar_napi_symbols_once = PTHREAD_ONCE_INIT;
static bool golar_init_napi_symbols(void) {
	if (pthread_once(&golar_napi_symbols_once, golar_init_napi_symbols_once) != 0) {
		fprintf(stderr, "golar.node: pthread_once failed\n");
		return false;
	}

	return golar_init_napi_symbols_once_ok;
}

#include "napi_dynamic_aliases.h"
#endif

#define METHOD_DESCRIPTOR_STR(NAME, VALUE) (napi_property_descriptor){ \
	.utf8name = NAME, \
	.name = NULL, \
	.method = VALUE, \
	.getter = NULL, \
	.setter = NULL, \
	.value = NULL, \
	.attributes = napi_enumerable, \
	.data = NULL \
}

static napi_value setSyncBuffer_bridge(napi_env env, napi_callback_info info) {
	size_t argc = 2;
	napi_value argv[argc];
	napi_get_cb_info(env, info, &argc, argv, NULL, NULL);

	uint32_t thread_id;
	napi_get_value_uint32(env, argv[0], &thread_id);

	size_t len = 0;
	void *sync_buffer;
	napi_get_arraybuffer_info(env, argv[1], &sync_buffer, &len);
	golar_js_setSyncBuffer(thread_id, (uintptr_t)sync_buffer, len);

	return NULL;
}

static napi_value linter_RuleTesterLint_bridge(napi_env env, napi_callback_info info) {
	size_t argc = 4;
	napi_value argv[4];
	napi_get_cb_info(env, info, &argc, argv, NULL, NULL);

	char *files_data;
	size_t files_len;
	napi_get_value_string_utf8(env, argv[0], NULL, 0, &files_len);
	files_len++;
	files_data = malloc(sizeof(char) * files_len);
	napi_get_value_string_utf8(env, argv[0], files_data, files_len, &files_len);

	char *fileName_data;
	size_t fileName_len;
	napi_get_value_string_utf8(env, argv[1], NULL, 0, &fileName_len);
	fileName_len++;
	fileName_data = malloc(sizeof(char) * fileName_len);
	napi_get_value_string_utf8(env, argv[1], fileName_data, fileName_len, &fileName_len);

	char *ruleName_data;
	size_t ruleName_len;
	napi_get_value_string_utf8(env, argv[2], NULL, 0, &ruleName_len);
	ruleName_len++;
	ruleName_data = malloc(sizeof(char) * ruleName_len);
	napi_get_value_string_utf8(env, argv[2], ruleName_data, ruleName_len, &ruleName_len);

	char *options_data;
	size_t options_len;
	napi_get_value_string_utf8(env, argv[3], NULL, 0, &options_len);
	options_len++;
	options_data = malloc(sizeof(char) * options_len);
	napi_get_value_string_utf8(env, argv[3], options_data, options_len, &options_len);

	struct golar_js_linter_RuleTesterLint_return res = golar_js_linter_RuleTesterLint(files_data, files_len, fileName_data, fileName_len, ruleName_data, ruleName_len, options_data, options_len);

	free(files_data);
	free(fileName_data);
	free(ruleName_data);
	free(options_data);

	napi_value res_str;
	napi_create_string_utf8(env, res.r0, res.r1, &res_str);
	free(res.r0);

	return res_str;
}

void workspace_created_cb(napi_env env, napi_value cb, void *context, void *data) {
	napi_value undefined;
	napi_get_undefined(env, &undefined);
	napi_call_function(env, undefined, cb, 0, NULL, NULL);
}
static napi_value workspace_New_bridge(napi_env env, napi_callback_info info) {
	size_t argc = 1;
	napi_value on_created_cb;

	napi_get_cb_info(env, info, &argc, &on_created_cb, NULL, NULL);

	napi_value resource_name;
	napi_create_string_utf8(env, "golar: workspace created", NAPI_AUTO_LENGTH, &resource_name);

	napi_threadsafe_function cb_ptr;
	napi_create_threadsafe_function(env, on_created_cb, NULL, resource_name, 0, 1, NULL, NULL, NULL, workspace_created_cb, &cb_ptr);

	golar_js_workspace_New((uintptr_t)cb_ptr);
	return NULL;
}

static uint32_t get_thread_id_from_args(napi_env env, napi_callback_info info) {
	size_t argc = 1;
	napi_value thread_id_value;
	napi_get_cb_info(env, info, &argc, &thread_id_value, NULL, NULL);

	uint32_t thread_id;
	napi_get_value_uint32(env, thread_id_value, &thread_id);
	return thread_id;
}

static napi_value workspace_ReadRequestedFileAt_bridge(napi_env env, napi_callback_info info) {
	uint32_t thread_id = get_thread_id_from_args(env, info);
	golar_js_workspace_ReadRequestedFileAt(thread_id);
	return NULL;
}

static napi_value workspace_ReadFileById_bridge(napi_env env, napi_callback_info info) {
	uint32_t thread_id = get_thread_id_from_args(env, info);
	golar_js_workspace_ReadFileById(thread_id);
	return NULL;
}

static napi_value workspace_GetTypeAtLocation_bridge(napi_env env, napi_callback_info info) {
	uint32_t thread_id = get_thread_id_from_args(env, info);
	golar_js_workspace_GetTypeAtLocation(thread_id);
	return NULL;
}

static void call_lint_done_cb(napi_env env, napi_value cb, void *context, void *data) {
	napi_value undefined;
	napi_get_undefined(env, &undefined);
	napi_call_function(env, undefined, cb, 0, NULL, NULL);
}
static napi_value workspace_Lint_bridge(napi_env env, napi_callback_info info) {
	size_t argc = 1;
	napi_value done_cb;
	napi_get_cb_info(env, info, &argc, &done_cb, NULL, NULL);

	napi_value resource_name;
	napi_create_string_utf8(env, "golar: lint done", NAPI_AUTO_LENGTH, &resource_name);

	napi_threadsafe_function done_cb_tsfn;
	napi_create_threadsafe_function(env, done_cb, NULL, resource_name, 0, 1, NULL, NULL, NULL, call_lint_done_cb, &done_cb_tsfn);

	golar_js_workspace_Lint((uintptr_t)done_cb_tsfn);
	return NULL;
}

static napi_value workspace_Report_bridge(napi_env env, napi_callback_info info) {
	uint32_t thread_id = get_thread_id_from_args(env, info);
	golar_js_workspace_Report(thread_id);
	return NULL;
}

static napi_value workspace_ReportCount_bridge(napi_env env, napi_callback_info info) {
	napi_value count;
	napi_create_uint32(env, golar_js_workspace_ReportCount(), &count);
	return count;
}

napi_value create_service_code_done_cb(napi_env env, napi_callback_info info) {
	void *data;
	napi_get_cb_info(env, info, NULL, NULL, NULL, &data);

	golar_js_jsCodegenCreateServiceCodeResponse((uint32_t)(uintptr_t)data);

	return NULL;
}

void call_create_service_code_cb(napi_env env, napi_value cb, void *context, void *data) {
	napi_value undefined;
	napi_get_undefined(env, &undefined);

	napi_value result;
	napi_call_function(env, undefined, cb, 0, NULL, &result);

	// TODO(perf): fast path for sync returns
	napi_value then;
	napi_get_named_property(env, result, "then", &then);

	napi_value then_cb;
	napi_create_function(env, "then", 4, create_service_code_done_cb, context, &then_cb);

	napi_value then_promise;
	napi_call_function(env, result, then, 1, &then_cb, &then_promise);
}

static napi_value registerJsCodegen_bridge(napi_env env, napi_callback_info info) {
	size_t argc = 2;
	napi_value argv[argc];
	napi_get_cb_info(env, info, &argc, argv, NULL, NULL);

	uint32_t thread_id;
	napi_get_value_uint32(env, argv[0], &thread_id);

	napi_value resource_name;
	napi_create_string_utf8(env, "golar: create service code", NAPI_AUTO_LENGTH, &resource_name);

	napi_threadsafe_function create_service_code_tsfn;
	napi_create_threadsafe_function(env, argv[1], NULL, resource_name, 0, 1, NULL, NULL, (void *)(uintptr_t)thread_id, call_create_service_code_cb, &create_service_code_tsfn);

	golar_js_registerJsCodegen(thread_id, (uintptr_t)create_service_code_tsfn);

	return NULL;
}

static napi_value registerIpcCodegen_bridge(napi_env env, napi_callback_info info) {
	golar_js_registerIpcCodegen();
	return NULL;
}

static void call_exit_code_cb(napi_env env, napi_value cb, void *context, void *data) {
	napi_value undefined;
	napi_get_undefined(env, &undefined);

	napi_value exit_code;
	napi_create_uint32(env, (uint32_t)(uintptr_t)data, &exit_code);

	napi_call_function(env, undefined, cb, 1, &exit_code, NULL);
}

static napi_value tsc_bridge(napi_env env, napi_callback_info info) {
	size_t argc = 2;
	napi_value argv[argc];
	napi_get_cb_info(env, info, &argc, argv, NULL, NULL);

	uint32_t thread_id;
	napi_get_value_uint32(env, argv[0], &thread_id);

	napi_value resource_name;
	napi_create_string_utf8(env, "golar: tsc done", NAPI_AUTO_LENGTH, &resource_name);

	napi_threadsafe_function done_cb_tsfn;
	napi_create_threadsafe_function(env, argv[1], NULL, resource_name, 0, 1, NULL, NULL, NULL, call_exit_code_cb, &done_cb_tsfn);

	golar_js_tsc(thread_id, (uintptr_t)done_cb_tsfn);

	return NULL;
}

static napi_value workspace_TypeCheck_bridge(napi_env env, napi_callback_info info) {
	size_t argc = 1;
	napi_value done_cb;
	napi_get_cb_info(env, info, &argc, &done_cb, NULL, NULL);

	napi_value resource_name;
	napi_create_string_utf8(env, "golar: typecheck done", NAPI_AUTO_LENGTH, &resource_name);

	napi_threadsafe_function done_cb_tsfn;
	napi_create_threadsafe_function(env, done_cb, NULL, resource_name, 0, 1, NULL, NULL, NULL, call_exit_code_cb, &done_cb_tsfn);

	golar_js_workspace_TypeCheck((uintptr_t)done_cb_tsfn);
	return NULL;
}

void napi_call_threadsafe_function_any(uintptr_t func, uintptr_t data, size_t is_blocking) {
	napi_call_threadsafe_function((napi_threadsafe_function)func, (void*)data, is_blocking);
}

NAPI_MODULE_INIT() {
	#ifdef GOLAR_NAPI_DYNAMIC
	if (!golar_init_napi_symbols()) {
		return NULL;
	}
	#endif

	napi_property_descriptor descriptors[] = {
		METHOD_DESCRIPTOR_STR("setSyncBuffer", setSyncBuffer_bridge),
		METHOD_DESCRIPTOR_STR("registerJsCodegen", registerJsCodegen_bridge),
		METHOD_DESCRIPTOR_STR("registerIpcCodegen", registerIpcCodegen_bridge),
		METHOD_DESCRIPTOR_STR("tsc", tsc_bridge),

		METHOD_DESCRIPTOR_STR("linter_RuleTesterLint", linter_RuleTesterLint_bridge),
		METHOD_DESCRIPTOR_STR("workspace_New", workspace_New_bridge),
		METHOD_DESCRIPTOR_STR("workspace_ReadRequestedFileAt", workspace_ReadRequestedFileAt_bridge),
		METHOD_DESCRIPTOR_STR("workspace_ReadFileById", workspace_ReadFileById_bridge),
		METHOD_DESCRIPTOR_STR("workspace_GetTypeAtLocation", workspace_GetTypeAtLocation_bridge),
		METHOD_DESCRIPTOR_STR("workspace_Lint", workspace_Lint_bridge),
		METHOD_DESCRIPTOR_STR("workspace_TypeCheck", workspace_TypeCheck_bridge),
		METHOD_DESCRIPTOR_STR("workspace_Report", workspace_Report_bridge),
		METHOD_DESCRIPTOR_STR("workspace_ReportCount", workspace_ReportCount_bridge),
	};

	napi_define_properties(env, exports, sizeof(descriptors)/sizeof(descriptors[0]), descriptors);
	return exports;
}
