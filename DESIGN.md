---
name: Massa
description: A calm, private instrument panel for tracking weight, BMI, and other personal metrics over time
colors:
  carbon: "oklch(0.16 0 0)"
  slate: "oklch(0.22 0.005 170)"
  graphite: "oklch(0.28 0.006 170)"
  graphite-hover: "oklch(0.33 0.007 170)"
  hairline: "oklch(0.32 0.006 170)"
  mist: "oklch(0.95 0.003 170)"
  fog: "oklch(0.64 0.01 170)"
  verdigris: "oklch(0.70 0.09 170)"
  verdigris-hover: "oklch(0.76 0.09 170)"
  ember: "oklch(0.62 0.17 25)"
typography:
  display:
    fontFamily: "IBM Plex Mono, ui-monospace, SFMono-Regular, monospace"
    fontSize: "clamp(1.75rem, 5vw, 2.5rem)"
    fontWeight: 500
    lineHeight: 1.1
    letterSpacing: "normal"
    fontFeature: "tnum"
  headline:
    fontFamily: "IBM Plex Sans, ui-sans-serif, system-ui, sans-serif"
    fontSize: "1.5rem"
    fontWeight: 600
    lineHeight: 1.25
  title:
    fontFamily: "IBM Plex Sans, ui-sans-serif, system-ui, sans-serif"
    fontSize: "1.125rem"
    fontWeight: 600
    lineHeight: 1.3
  body:
    fontFamily: "IBM Plex Sans, ui-sans-serif, system-ui, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 400
    lineHeight: 1.5
  label:
    fontFamily: "IBM Plex Sans, ui-sans-serif, system-ui, sans-serif"
    fontSize: "0.75rem"
    fontWeight: 500
    lineHeight: 1.4
    letterSpacing: "0.01em"
rounded:
  sm: "6px"
  md: "10px"
spacing:
  sm: "8px"
  md: "16px"
  lg: "24px"
components:
  button-primary:
    backgroundColor: "{colors.verdigris}"
    textColor: "{colors.carbon}"
    typography: "{typography.label}"
    rounded: "{rounded.sm}"
    padding: "10px 16px"
  button-primary-hover:
    backgroundColor: "{colors.verdigris-hover}"
    textColor: "{colors.carbon}"
    rounded: "{rounded.sm}"
  button-secondary:
    backgroundColor: "{colors.graphite}"
    textColor: "{colors.mist}"
    typography: "{typography.label}"
    rounded: "{rounded.sm}"
    padding: "10px 16px"
  button-secondary-hover:
    backgroundColor: "{colors.graphite-hover}"
    textColor: "{colors.mist}"
    rounded: "{rounded.sm}"
  button-destructive:
    backgroundColor: "transparent"
    textColor: "{colors.ember}"
    typography: "{typography.label}"
    rounded: "{rounded.sm}"
    padding: "10px 16px"
  input:
    backgroundColor: "{colors.graphite}"
    textColor: "{colors.mist}"
    typography: "{typography.body}"
    rounded: "{rounded.sm}"
    padding: "10px 12px"
  surface-card:
    backgroundColor: "{colors.slate}"
    rounded: "{rounded.md}"
    padding: "20px"
---

# Design System: Massa

## 1. Overview

**Creative North Star: "The Quiet Gauge"**

Massa reads like the display face of a precision instrument at rest: a
near-black panel, legible numerals, and exactly one color that glows when
something is worth your attention. It is closer to the dim readout on a
bathroom scale at 6am, or the home screen of an Oura ring, than to a SaaS
admin dashboard. Nothing here is trying to motivate, congratulate, or alarm —
it states the number and the trend, and gets out of the way.

This system explicitly rejects the **generic SaaS dashboard** look the
codebase currently has — gray-and-blue default Tailwind, identical bordered
white cards stacked end to end — and the **gamified fitness app** register of
streaks, badges, and gradient "you crushed it!" energy. Weight and BMI are
presented as readings, not scores.

**Key Characteristics:**
- Near-black carbon surface; depth comes from tonal steps, not shadows or borders.
- One accent color (verdigris, an oxidized-teal) — reserved for the single
  thing on screen that matters right now.
- Numerals set in a monospaced face with tabular figures, so readings feel
  measured rather than typeset.
- Tight, precise corner radii (6–10px) — an instrument panel, not a pillow.
- Equal-weight responsive layout: the same calm reading on phone and desktop.

## 2. Colors

A near-monochrome dark palette with a single living accent. Strategy:
**Restrained** — the accent appears on at most one element class per screen.

### Primary
- **Verdigris** (`oklch(0.70 0.09 170)`): the one accent. Used for the
  reading that matters right now — today's weight/BMI value, the active
  range or metric toggle, the live chart line, primary action buttons (Add
  entry, Connect, Sync). Named for oxidized copper/teal patina — a color
  that looks like it's always been there, not one that was just applied.
- **Verdigris Hover** (`oklch(0.76 0.09 170)`): brightened verdigris for
  hover/active states on filled primary elements. Text on both is Carbon —
  Mist-on-Verdigris falls to ~2.2:1, well under the AA floor, so filled
  Verdigris elements always pair with Carbon text (~7.6:1).

### Neutral
- **Carbon** (`oklch(0.16 0 0)`): the base surface — page background. Pure
  neutral, no hue tint, so Verdigris reads as the only color in the room.
- **Slate** (`oklch(0.22 0.005 170)`): one tonal step up from Carbon. Used
  for cards/sections (the dashboard panels, settings sections).
- **Graphite** (`oklch(0.28 0.006 170)`): a second tonal step up. Used for
  inputs, inactive segmented-control options, and secondary/ghost buttons —
  things that sit "inside" a Slate card.
- **Graphite Hover** (`oklch(0.33 0.007 170)`): hover state for Graphite
  surfaces.
- **Hairline** (`oklch(0.32 0.006 170)`): the only border color in the
  system, used at 1px and only where a divider is structurally necessary
  (e.g. between rows in the entry list). Not used to outline cards.
- **Mist** (`oklch(0.95 0.003 170)`): primary text. Near-white with a
  whisper of the accent hue so it never looks like print-shop gray-on-black.
- **Fog** (`oklch(0.64 0.01 170)`): secondary/meta text — timestamps, units,
  helper labels. Reaches ~5:1 against Carbon; never used for body copy a
  user must read closely.

### Semantic
- **Ember** (`oklch(0.62 0.17 25)`): form-validation errors and the
  destructive (delete) action only. A warm, slightly desaturated red —
  legible as "something needs attention" without reading as an alarm.

### Named Rules
**The Single Glow Rule.** Verdigris appears on exactly one element class per
screen: the primary action, the active toggle, or the live data series —
never all three competing at once, and never as decoration (icon tints,
section headers, borders).

**The Numbers Don't Judge Rule.** Weight, BMI, trend lines, and subjective
ratings are always Mist or Verdigris, regardless of whether the number went
up or down, or which direction is "good". This extends to directional and
subjective metrics: a 10 on the overwhelm scale renders in exactly the same
Verdigris as a 1 — no amber or ember bands at the high end, no gradient
across the scale, no green-to-red run. A hard day is data, not a failure
state. Reference lines (the overwhelm baseline of 3) are drawn in Fog:
present, quiet, uncoloured. Ember is reserved for system/form errors —
never for "you gained weight" or "you had a bad day" framing.

## 3. Typography

**Display Font:** IBM Plex Mono (with `ui-monospace, SFMono-Regular, monospace`)
**Body Font:** IBM Plex Sans (with `ui-sans-serif, system-ui, sans-serif`)

**Character:** A geometric, slightly technical sans for everything you read,
paired with a monospaced face — set in tabular figures — for everything you
*measure*. The mono face is what makes a number feel like a reading rather
than a label.

### Hierarchy
- **Display** (500, `clamp(1.75rem, 5vw, 2.5rem)`, line-height 1.1, IBM Plex
  Mono, tabular figures): the headline stat — latest weight, BMI value,
  weekly average. Appears once or twice per screen, never for body text.
- **Headline** (600, 1.5rem/24px, line-height 1.25, IBM Plex Sans): page
  titles ("Massa", "Settings").
- **Title** (600, 1.125rem/18px, line-height 1.3, IBM Plex Sans): section
  headings ("Add weight entry", "Google Health", "Recent entries").
- **Body** (400, 0.875rem/14px, line-height 1.5, IBM Plex Sans, max ~70ch):
  form labels' helper text, status copy, entry-list rows.
- **Label** (500, 0.75rem/12px, letter-spacing 0.01em, IBM Plex Sans):
  field labels, units, timestamps, toggle-button text.

### Named Rules
**The Reading vs. Label Rule.** If a number is something the user *measures*
(weight, BMI, height), set it in Display (IBM Plex Mono, tabular). If it's
metadata about a measurement (date, unit, source), set it in Label (IBM Plex
Sans). Never mix the two faces for the same value.

## 4. Elevation

Flat by default. There are no shadows anywhere in the system — depth is
conveyed entirely through the three-step tonal ramp (Carbon → Slate →
Graphite). A card is "raised" because it's a lighter gray than the page
behind it, not because it casts a shadow on it.

### Named Rules
**The Flat-By-Default Rule.** Surfaces never cast shadows, at rest or on
hover. If an element needs to feel interactive, shift it one tonal step
(Slate → Graphite) or brighten Verdigris (→ Verdigris Hover), not add depth.

## 5. Components

### Buttons
- **Shape:** 6px radius (`{rounded.sm}`) — precise, not pillowed.
- **Primary** (`button-primary`): Verdigris fill, Carbon text, `10px 16px`
  padding, Label typography. Used once per view for the dominant action (Add
  entry, Connect Google Health, Sync now).
- **Secondary / Ghost** (`button-secondary`): Graphite fill, Mist text. Used
  for Settings, Log out, Cancel, and inactive segmented-control options.
- **Destructive** (`button-destructive`): transparent background, Ember
  text, no fill. Delete actions stay quiet until pressed — no red block to
  flinch at while scanning the entry list.
- **Hover/Focus:** filled buttons step to their `*-hover` color; all buttons
  get a 2px Verdigris focus ring (`outline: 2px solid oklch(0.70 0.09 170)`,
  offset 2px) for keyboard navigation.

### Segmented Controls (range presets, daily/weekly, weight/BMI toggles)
- **Style:** a row of `button-secondary`-shaped pills (6px radius, Graphite
  background, Mist text) where the active option becomes `button-primary`
  (Verdigris fill, Carbon text). Exactly one option per group may glow — this
  is one of the system's sanctioned uses of the Single Glow Rule.

### Cards / Containers (`surface-card`)
- **Corner Style:** 10px radius (`{rounded.md}`).
- **Background:** Slate, on the Carbon page background.
- **Shadow Strategy:** none (see Elevation). Separation comes purely from
  the Carbon → Slate tonal step.
- **Border:** none. Hairline is reserved for in-list dividers only.
- **Internal Padding:** 20px (`lg`-ish), with `16px` (`md`) gaps between
  stacked sections.

### Inputs / Fields
- **Style:** Graphite background, Mist text, 6px radius, no border (the
  Carbon/Slate → Graphite step itself signals "this is editable").
- **Focus:** 2px Verdigris outline, offset 2px — same treatment as button
  focus, for one consistent "you are here" signal across the app.
- **Error:** Ember helper text below the field; the field itself stays
  Graphite (no red border) — the message communicates the problem, the input
  doesn't shout it.

### Navigation
- A single-row header: Headline-weight "Massa" wordmark on the left,
  `button-secondary` Settings/Log out on the right. No active-state
  indicators or icons needed — there's only ever one page open at a time.

### Stat Readout (signature component)
The dashboard's core unit: a Label (e.g. "Latest weight", "This week's avg",
"BMI") above a Display-weight value in Mist, with the unit set in Label/Fog
immediately after the number (`82.4 kg`, not a separate line). Multiple Stat
Readouts sit in a single Slate `surface-card` as a loose grid — no per-stat
borders or boxes; whitespace alone separates them.

## 6. Do's and Don'ts

### Do:
- **Do** keep the page Carbon and let Verdigris be the only saturated color
  anywhere on screen (The Single Glow Rule).
- **Do** set every weight/BMI/height value in IBM Plex Mono with tabular
  figures (The Reading vs. Label Rule).
- **Do** convey hierarchy and "card-ness" with the Carbon → Slate → Graphite
  tonal steps, never with borders or shadows.
- **Do** use Ember only for form/system errors and the destructive action —
  never to color-code a weight change.
- **Do** keep corner radii tight (6px controls, 10px cards) — an instrument
  panel, not a consumer wellness app.

### Don't:
- **Don't** reintroduce "bland gray-and-blue, default Tailwind palette" —
  the look this redesign replaces. No `bg-blue-600`, no `bg-gray-50`/`bg-white`
  cards with `border-gray-200`.
- **Don't** stack "identical bordered cards... repeated for every section" —
  vary density and let whitespace do the separating work inside a single
  Slate surface where possible.
- **Don't** add streaks, badges, gradients, or "you crushed it!" copy —
  Massa doesn't coach or celebrate (no gamified-fitness energy).
- **Don't** color weight-up vs. weight-down differently (no red-for-gain /
  green-for-loss anywhere, including the chart line).
- **Don't** add drop shadows, glassmorphism, or hover-lift effects — flat by
  default, always.
