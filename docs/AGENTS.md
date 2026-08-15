# dnsmonster docs site

Hugo with the [Hextra](https://imfing.github.io/hextra/) theme, pulled in as a Hugo module.
No npm, no JS framework. Requires **Hugo extended** and **Go** (for the module).

```sh
hugo server                # dev
hugo --minify              # build into public/
hugo mod get -u            # update the theme
```

## Rules

- **Keep it light.** This site was Docsy, then Astro + Starlight + React + shadcn, then hand-written
  Hugo layouts, now Hextra. The first two were replaced for being heavy. Do not add a JS toolchain
  or a component library.
- **No expensive CSS.** No gradients, `filter`, `backdrop-filter`, `mask-image`, or animation. An
  earlier revision used a `box-shadow` keyframe animation, a `position: fixed` full-viewport gradient
  and a blurred sticky header, and scrolling was janky.
- **Brand lives in one file.** `assets/css/custom.css` — Hextra loads it automatically. It sets
  `--primary-hue/saturation/lightness` (Fennec Amber), the Saira and JetBrains Mono faces, and the
  Fenko graphite surfaces. Don't hardcode colours elsewhere.
- **Dark-only.** There is no light palette and no theme switcher, by request.
  `params.theme.displayToggle = false` removes the switcher (and with it the sidebar's sticky
  footer, which painted Hextra's `bg-dark` #111 instead of Fenko graphite).
  `assets/js/head/theme.js` overrides the theme's copy to force the `dark` class and drop any
  stored `color-theme` — Hextra prefers localStorage over the configured default, which would
  otherwise strand anyone who had toggled to light. (The body script re-persists the resolved theme,
  so the key settles on `dark` rather than absent.) That override **must keep `setTheme` defined
  and global**; `js/core/theme.js` calls it.
- Saira is the canonical Fenko typeface. Do not swap it for Inter.

## Content

- Front matter is TOML: `title`, `description`, `weight`. `weight` orders within the sidebar group.
  Do **not** set `linkTitle` on a section `_index.md` — Hextra uses it as the section's name in the
  sidebar.
- Section order comes from `weight` in each section's `_index.md`.
- `content/_index.md` is the site root **and** the Getting Started page — there is no separate
  landing page, by request. It carries `[cascade] type = "docs"`, which is what gives every page the
  docs sidebar while keeping URLs at the root rather than under `/docs/`.
  `[cascade]` must be the **last** table in the front matter or it swallows the scalar keys above it.
- Callouts use Hextra's shortcode: `{{< callout type="info|warning|error" >}}`. Single quotes inside
  shortcode parameters — double quotes break parsing.
- URLs must not change. `netlify.toml` in the repo root holds 301s from the old Docsy `/docs/*`
  paths; renaming a page means adding a redirect there.

## Theme overrides

Kept deliberately small — each one is a maintenance liability when Hextra updates.

- `layouts/docs/single.html` — verbatim copy of the theme's, plus a page-actions row after the
  content. Hextra renders "Edit this page" and "Scroll to top" inside `_partials/toc.html`, which
  pins them under the "On this page" column; they belong at the foot of the content.
  `params.editURL.enable = false` suppresses the theme's edit link.
- `assets/css/custom.css` hides `div:has(> #backToTop)` — hiding the button alone leaves an empty
  bordered box at the foot of the TOC.

## Writing

The prose is the maintainer's. Do not rewrite documentation copy into marketing voice, and do not
introduce claims that aren't in the source — an earlier revision asserted "no agents on your
resolvers", which is wrong, since the dnstap path configures the resolver itself.

## Deployment

Netlify. `netlify.toml` at the repo root is the source of truth for build settings and redirects.
The build needs Go on the image for the Hugo module; this is proven — the previous Docsy setup was
also a Hugo module on the same pipeline.
