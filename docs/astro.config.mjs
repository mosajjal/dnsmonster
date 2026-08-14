// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import react from '@astrojs/react';
import tailwindcss from '@tailwindcss/vite';

// https://astro.build/config
export default defineConfig({
	site: 'https://dnsmonster.dev',
	integrations: [
		react(),
		starlight({
			title: 'dnsmonster',
			description:
				'Passive DNS monitoring framework. Capture, filter and ship hundreds of thousands of DNS queries per second to the backend of your choice.',
			logo: {
				light: './src/assets/logo-wide-light.svg',
				dark: './src/assets/logo-wide-dark.svg',
				replacesTitle: true,
			},
			favicon: '/favicon.ico',
			head: [
				{
					tag: 'link',
					attrs: {
						rel: 'icon',
						href: '/favicons/favicon-32x32.png',
						type: 'image/png',
						sizes: '32x32',
					},
				},
				{
					tag: 'link',
					attrs: {
						rel: 'apple-touch-icon',
						href: '/favicons/apple-touch-icon-180x180.png',
						sizes: '180x180',
					},
				},
				{ tag: 'meta', attrs: { name: 'theme-color', content: '#0d1117' } },
			],
			customCss: ['./src/styles/global.css'],
			components: {
				PageTitle: './src/components/PageTitle.astro',
				Footer: './src/components/Footer.astro',
			},
			social: [
				{
					icon: 'github',
					label: 'GitHub',
					href: 'https://github.com/mosajjal/dnsmonster',
				},
			],
			editLink: {
				baseUrl: 'https://github.com/mosajjal/dnsmonster/edit/main/docs/',
			},
			lastUpdated: true,
			sidebar: [
				{
					label: 'Getting started',
					items: [
						{ label: 'Overview', link: '/' },
						{ label: 'Installation', slug: 'getting-started/installation' },
						{ label: 'Post-installation', slug: 'getting-started/post-installation' },
					],
				},
				{
					label: 'Configuration',
					items: [{ autogenerate: { directory: 'configuration' } }],
				},
				{
					label: 'Inputs and filters',
					items: [{ autogenerate: { directory: 'inputs' } }],
				},
				{
					label: 'Outputs',
					items: [{ autogenerate: { directory: 'outputs' } }],
				},
				{
					label: 'Tutorials',
					items: [{ autogenerate: { directory: 'tutorials' } }],
				},
				{
					label: 'Reference',
					items: [
						{ label: 'FAQ', slug: 'faq' },
						{ label: 'Privacy policy', slug: 'privacy' },
					],
				},
			],
		}),
	],
	vite: { plugins: [tailwindcss()] },
});
