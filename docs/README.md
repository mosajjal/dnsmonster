# dnsmonster documentation

The documentation site for [dnsmonster](https://github.com/mosajjal/dnsmonster), published at
[dnsmonster.dev](https://dnsmonster.dev).

[Hugo](https://gohugo.io) with the [Hextra](https://imfing.github.io/hextra/) theme, pulled in as a
Hugo module. No npm and no JS framework — search, dark mode and the sidebar all come from the theme.

Requires **Hugo extended** and **Go** (Go is only needed to resolve the theme module).

## Running locally

```sh
hugo server
```

To inspect exactly what ships:

```sh
hugo --minify && python3 -m http.server 4400 --directory public
```

To update the theme:

```sh
hugo mod get -u && hugo mod tidy
```

## Layout

```
hugo.toml              site config: navbar, search, theme params
content/               every page (TOML front matter)
  _index.md              the site root — this IS the Getting Started page
  getting-started/       installation, post-installation
  configuration/  inputs/  outputs/  tutorials/
  faq.md  privacy.md
assets/css/custom.css  the entire brand layer; Hextra loads it automatically
layouts/docs/single.html  the one theme override (see below)
static/                fonts, favicons, diagrams, .well-known/security.txt
```

## Writing pages

Front matter is TOML. `title`, `description` and `weight` are what matter — `weight` orders the page
within its sidebar group.

```toml
+++
title = "Apache Kafka"
description = "One sentence, used for the page lead and the meta description."
weight = 3
+++
```

Section order comes from the `weight` in each section's `_index.md` (getting-started 1,
configuration 2, inputs 3, outputs 4, tutorials 5).

Two gotchas worth knowing:

- **Don't set `linkTitle` on a section `_index.md`.** Hextra uses it as the section's name in the
  sidebar, so `linkTitle = "Overview"` renames the whole section to "Overview".
- **`[cascade]` must be the last table in the front matter.** In TOML, any scalar key after a table
  header belongs to that table — put it first and it swallows `title` and `weight`.

There is no separate landing page. `content/_index.md` is the site root **and** the Getting Started
page, and its `[cascade] type = "docs"` is what gives every page the docs sidebar while keeping URLs
at the root rather than nested under `/docs/`.

### Callouts

```
{{< callout type="warning" >}}
Markdown body. Types: info, warning, error.
{{< /callout >}}
```

Use single quotes inside shortcode parameters — a double quote inside a `title="..."` breaks parsing.

## Theme overrides

Deliberately minimal, since each one drifts when Hextra updates:

- `layouts/docs/single.html` — a verbatim copy of the theme's, plus a page-actions row after the
  content. Hextra renders "Edit this page" and "Scroll to top" inside `_partials/toc.html`, which
  pins them under the "On this page" column.
- `assets/css/custom.css` hides `div:has(> #backToTop)`, because hiding the button alone leaves an
  empty bordered box at the foot of the TOC.

## Design

Fenko Security palette: Fennec Amber (`#F5A623`) driving Hextra's
`--primary-hue/saturation/lightness`, on graphite neutrals. Saira for text, JetBrains Mono for code,
both self-hosted from `static/fonts/`.

**Dark-only.** There is no light theme and no switcher. `assets/js/head/theme.js` pins the `dark`
class and clears any stored `color-theme`, so a preference left over from when the switcher existed
cannot strand a reader on an unmaintained light palette.

All of it is in `assets/css/custom.css`. Edit it there.

## Deployment

Netlify, configured by `netlify.toml` in the repository root (`hugo --minify`, publish `docs/public`).
That file also holds the 301s from the old Docsy `/docs/*` URLs.

## Prose linting

```sh
vale content
```
