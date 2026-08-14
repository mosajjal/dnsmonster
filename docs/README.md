# dnsmonster documentation

The documentation site for [dnsmonster](https://github.com/mosajjal/dnsmonster), published at
[dnsmonster.dev](https://dnsmonster.dev).

Plain [Hugo](https://gohugo.io). **No theme, no Hugo modules, no npm** — the layouts live in
`layouts/` and total a few hundred lines. A build needs nothing but the Hugo binary (extended not
required).

## Running locally

```sh
hugo server
```

To inspect exactly what ships:

```sh
hugo --minify && python3 -m http.server 4400 --directory public
```

## Layout

```
hugo.toml            site config, nav weights come from content
content/             every page (Hugo TOML front matter)
  _index.md            the site root — this IS the Getting Started page
  getting-started/     installation, post-installation
  configuration/  inputs/  outputs/  tutorials/
  faq.md  privacy.md
layouts/
  baseof.html          shell: topbar, sidebar, main, TOC
  home.html  page.html  section.html  404.html
  partials/            head, sidebar, footer
  shortcodes/          callout, stats
assets/css/main.css  the entire stylesheet, ~8KB minified
static/              fonts, favicons, diagrams, .well-known/security.txt
```

## Writing pages

Front matter is TOML. `title`, `description` and `weight` are all that matter — `weight` orders the
page within its sidebar group. `linkTitle` overrides the sidebar label.

```toml
+++
title = "Apache Kafka"
description = "One sentence, used for the page lead and the meta description."
weight = 3
+++
```

Section order in the sidebar comes from the `weight` in each section's `_index.md`
(getting-started 1, configuration 2, inputs 3, outputs 4, tutorials 5).

There is no separate landing page. `content/_index.md` is the site root **and** the Getting Started
overview. `content/getting-started/_index.md` sets `build.render = "never"` — it exists only to group
the section in the navigation.

### Shortcodes

```
{{< callout type="caution" title="Optional title" >}}
Markdown body. Types: note, tip, caution, danger.
{{< /callout >}}

{{< stats >}}
200k+ | queries per second
15+ | output backends
{{< /stats >}}
```

Avoid double quotes inside a shortcode parameter — use single quotes.

## Design

Fenko Security palette: Fennec Amber (`#F5A623` dark, `#D4920A` light) on graphite neutrals. Saira
for text, JetBrains Mono for code, both self-hosted from `static/fonts/`. Dark is the default; the
Theme button toggles and persists to `localStorage`.

Every colour is a custom property at the top of `assets/css/main.css`. Edit it there. Anything added
to `:root` needs a `:root[data-theme='light']` value too.

The stylesheet deliberately avoids gradients, `filter`, `backdrop-filter`, `mask-image` and
animation. An earlier revision used all of them and made scrolling janky. Everything here paints
once. Keep it that way.

## Deployment

Netlify, configured by `netlify.toml` in the repository root (`hugo --minify`, publish `docs/public`).
That file also holds the 301s from the old Docsy `/docs/*` URLs.

## Prose linting

```sh
vale content
```
