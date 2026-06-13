# Product

## Register

product

## Users

A small allowlist of trusted users (the owner plus a handful of family/friends),
each tracking their own weight/BMI over time. Used roughly equally on phone and
desktop: quick daily weight logging on the go, plus periodic trend review on a
larger screen. Data is personal health information — private by default, no
social or comparative features.

## Product Purpose

Massa is a personal weight/BMI tracker that syncs with Google Health. It exists
to make logging a weight effortless and to make the resulting trend (weight over
time, BMI, weekly averages) easy to read at a glance. Success looks like: open
the app, see today's number and the trend immediately, log a new entry in
seconds, and (when connected) trust that Google Health stays in sync without
manual effort.

## Brand Personality

Calm and quiet — a private journal, not a coach or a leaderboard. Numbers are
presented without judgment: no alarming reds for weight gain, no celebratory
fireworks for loss. Visually closer to Oura/Whoop's premium quantified-self
feel than to Apple Health's soft pastel cards or a typical SaaS admin panel —
restrained, confident typography, generous space, and a single accent color
reserved for the metric that actually matters right now.

## Anti-references

- Generic SaaS admin dashboard: bland gray-and-blue, default Tailwind palette,
  identical bordered cards repeated for every section.
- Gamified fitness apps: streaks, badges, gradients, "you crushed it!" copy —
  too much pressure and noise for a daily health check-in.

## Design Principles

- **Numbers without judgment.** Weight and BMI are presented neutrally — no
  color-coded good/bad framing on the data itself. Tone stays factual and calm.
- **One accent, used deliberately.** A single accent color is reserved for the
  thing that matters most on a given screen (the latest reading, the active
  range), not spread across buttons and badges as decoration.
- **Calm density.** Show real data — charts, history, multiple stats — without
  resorting to a wall of identical cards. Hierarchy and spacing do the work
  that borders and boxes currently do.
- **Equally at home on phone and desktop.** Every surface is designed for both
  a quick mobile check-in and a deliberate desktop review session, not one
  shrunk to fit the other.
- **Get out of the way.** Logging a new entry and reading the trend are the two
  jobs that matter; everything else (settings, sync status) stays quiet and
  out of the primary flow.

## Accessibility & Inclusion

WCAG AA contrast throughout, full keyboard navigation, and
`prefers-reduced-motion` alternatives for any animation. No additional personal
accommodations specified.
