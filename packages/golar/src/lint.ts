import type { LintConfiguredRule } from './config.ts'
import type { SourceFile } from './unstable-tsgo.ts'
import type { Program } from './workspace.ts'

export interface CharacterReportRange {
	/** 0-indexed */
	begin: number
	/** 0-indexed */
	end: number
}

export type RuleReport =
	| {
			/** Range in `context.sourceFile`, the service code rules visit */
			range: CharacterReportRange
			sourceRange?: never
			message: string
	  }
	| {
			/**
			 * Range in the original file text (as read from disk), for text the
			 * service code does not contain, like a Svelte `class:name` directive
			 */
			sourceRange: CharacterReportRange
			range?: never
			message: string
	  }

export type RuleContext = {
	program: Program
	sourceFile: SourceFile
	report: (report: RuleReport) => void
}

export type RuleConfig = {
	name: string
	setup(context: RuleContext): void
}

export function defineRule(config: RuleConfig): LintConfiguredRule {
	return {
		rule: {
			isCustomJs: true,
			...config,
		},
	} as unknown as LintConfiguredRule
}

export type NativeRuleConfig = {
	addonPath: string
	name: string
}

export function defineNativeRule(config: NativeRuleConfig) {
	return {
		rule: {
			isNative: true,
			...config,
		},
	} as unknown as LintConfiguredRule
}
