# dnsmonster docs site

Plain Hugo. No theme, no Hugo modules, no npm, no JS framework. Layouts are local and small.

```sh
hugo server                # dev
hugo --minify              # build into public/
```

## Rules

- **Keep it minimal.** This site was previously Docsy, then Astro + Starlight + React + shadcn. Both
  were replaced for being heavy. Do not reintroduce a build toolchain, a component library, or a
  theme module without being asked.
- **No expensive CSS.** No gradients, `filter`, `backdrop-filter`, `mask-image`, or animation. An
  earlier revision used a `box-shadow` keyframe animation, a `position: fixed` full-viewport gradient
  and a blurred sticky header, and scrolling was janky. Everything must paint once.
- **Colours are tokens.** All of them live at the top of `assets/css/main.css`. Never hardcode a
  colour in a layout. Anything added to `:root` needs a `:root[data-theme='light']` value.
- **Both themes are first class.** Dark is default, light is toggled via `data-theme` on `<html>`.
- Fonts are Saira (text) and JetBrains Mono (code), self-hosted in `static/fonts/`. Saira is the
  canonical Fenko typeface — do not swap it for Inter.

## Content

- Front matter is TOML: `title`, `description`, `weight`. `weight` orders within the sidebar group;
  `linkTitle` overrides the sidebar label.
- Section order comes from `weight` in each section's `_index.md`.
- `content/_index.md` is the site root **and** the Getting Started page. There is no separate
  landing page, by request. `content/getting-started/_index.md` uses `build.render = "never"` and
  exists only to group the nav.
- Shortcodes: `callout` (note/tip/caution/danger) and `stats`. Single quotes inside shortcode
  parameters — double quotes break parsing.
- URLs must not change. `netlify.toml` in the repo root holds 301s from the old Docsy `/docs/*`
  paths; renaming a page means adding a redirect there.

## Deployment

Netlify. `netlify.toml` at the repo root is the source of truth for build settings and redirects —
the Netlify UI is overridden by it.
