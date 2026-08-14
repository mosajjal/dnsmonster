# dnsmonster docs site

Astro + Starlight documentation site, Tailwind CSS v4, shadcn/ui, Fenko Security design system.

## Commands

```sh
npm run dev                  # dev server on :4321
npx astro dev --background   # detached; astro dev stop | status | logs
npm run build                # static build into dist/
```

## Structure

- `src/content/docs/` — all pages. `index.mdx` is the site root **and** the "Getting started"
  overview; there is no separate landing page.
- `src/components/` — `Hero`, `Features`, `StatBand`, `OutputGrid`, `StageBadge`, and the
  `PageTitle` Starlight override (honours `hideTitle: true` in frontmatter).
- `src/components/ui/` — shadcn/ui primitives. Add more with `npx shadcn@latest add <component>`.
- `src/styles/global.css` — the single source of truth for design tokens and Starlight overrides.
- `astro.config.mjs` — sidebar definition. "Getting started" and "Reference" are explicit; the rest
  autogenerate from their directories using `sidebar.order` in each page's frontmatter.

## Rules

- Never hardcode a colour in a component. Use the tokens in `global.css` (`--fk-*` for the custom
  layer, `--sl-color-*` for Starlight, shadcn's `--primary`/`--muted`/etc. for UI primitives).
- Both themes are first-class. Anything added to `:root` needs its `[data-theme='light']` value too.
- Astro scoped styles do not reach markup injected with `set:html` — use `:global()` for those.
- Headings are `display: inline` inside `.sl-heading-wrapper`; style the wrapper, not the heading.
- Fonts are Saira (UI/headings) and JetBrains Mono (code), self-hosted through Fontsource.

## Reference

Astro docs: https://docs.astro.build — Starlight docs: https://starlight.astro.build
