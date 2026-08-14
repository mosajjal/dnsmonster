import { defineCollection, z } from 'astro:content';
import { docsLoader } from '@astrojs/starlight/loaders';
import { docsSchema } from '@astrojs/starlight/schema';

export const collections = {
	docs: defineCollection({
		loader: docsLoader(),
		schema: docsSchema({
			extend: z.object({
				/** Hide Starlight's default page title (used on pages that open with a custom hero). */
				hideTitle: z.boolean().optional(),
			}),
		}),
	}),
};
