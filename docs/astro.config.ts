import { defineConfig } from 'astro/config'
import starlight from '@astrojs/starlight'
import starlightBlog from 'starlight-blog'
import starlightLinksValidator from 'starlight-links-validator'

export default defineConfig({
	site: 'https://golar.dev',
	integrations: [
		starlight({
			title: 'Golar',
			description:
				'Documentation for Golar, a typescript-go-based embedded language tooling orchestrator for Astro, Ember, Svelte, Vue, and other TypeScript-based languages.',
			tagline:
				'Embedded language tooling orchestrator powered by typescript-go',
			plugins: [
				starlightBlog({
					authors: {
						auvred: {
							name: 'auvred',
							picture: 'https://avatars.githubusercontent.com/u/61150013',
							url: 'https://github.com/auvred',
						},
					},
				}),
				starlightLinksValidator(),
			],
			social: [
				{
					icon: 'github',
					label: 'GitHub',
					href: 'https://github.com/auvred/golar',
				},
			],
			sidebar: [
				{
					label: 'Guides',
					items: [{ label: 'Getting started', slug: 'guides/getting-started' }],
				},
				{
					label: 'Languages',
					items: [
						{ label: 'Astro', slug: 'languages/astro' },
						{ label: 'Ember', slug: 'languages/ember' },
						{ label: 'Svelte', slug: 'languages/svelte' },
						{ label: 'Vue', slug: 'languages/vue' },
					],
				},
				{
					label: 'Run Modes',
					items: [
						{
							label: 'Default mode',
							slug: 'modes/default',
						},
						{
							label: 'lint',
							slug: 'modes/lint',
						},
						{
							label: 'typecheck',
							slug: 'modes/typecheck',
						},
						{
							label: 'tsc',
							slug: 'modes/tsc',
						},
					],
				},
				{
					label: 'Custom Lint Rules',
					items: [
						{
							label: 'JavaScript rules',
							slug: 'custom-rules/javascript',
						},
						{
							label: 'Rust rules',
							slug: 'custom-rules/rust',
						},
					],
				},
			],
			editLink: {
				baseUrl: 'https://github.com/auvred/golar/edit/main/docs/',
			},
			tableOfContents: {
				maxHeadingLevel: 4,
			},
			customCss: ['./src/styles/custom.css'],
			components: {
				SiteTitle: './src/components/starlight/SiteTitle.astro',
			},
		}),
	],
})
