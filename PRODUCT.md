# Product

## Register

product

## Users

Two operator groups, one codebase:

- **Onboard deck crew** (bosun, deck officers, ABs) on a single vessel. Primary
  context: standing on an open mooring deck, tablet in hand, often in daylight
  glare, spray, or wind, sometimes gloved. They glance at the screen between
  physical tasks: which line is on which drum, what condition it's in, is a turn
  or inspection due. Short, interrupted sessions. Getting the wrong line or
  drum has real safety and cost consequences.
- **Shore fleet managers** at a desk, overseeing every vessel. Mouse and
  keyboard, longer sessions, denser data: catalogue maintenance, fleet-wide
  condition reports, logbooks, certificates.

The deck tablet is the floor the design is built on; the office view scales up
from it, never the reverse.

## Product Purpose

Track every mooring line across its life: where it physically sits (deck map of
winches, drums, storage), its identity and certification, side-tracking and
turning, inspections (manual + ingested from a third-party API), condition
rollups, and document/photo evidence. It replaces spreadsheets and paper
logbooks for mooring-line management at Norwegian Cruise Line. Success: a crew
member can find any line and read its true condition in seconds on deck, and a
shore manager can trust the fleet-wide condition and certification picture
without chasing vessels.

## Brand Personality

Rugged and precise. A maritime instrument, not an app: legible, trustworthy,
built to be read at a glance under bad conditions and hard use. Function first,
decoration never. Three words: **rugged, precise, legible.** It should feel like
certified deck equipment a crew relies on, not like software being sold.

Voice in UI copy: direct and operational. Name the thing and the action ("Turn
line", "Log inspection", "Move to drum"). No marketing tone, no cleverness.

## Anti-references

- **Generic AI SaaS.** No cream/parchment backgrounds, no gradient text, no
  tiny tracked uppercase eyebrow above every section, no hero-metric template
  (big number + small label + gradient), no endless identical icon-card grids.
- Not a consumer/playful app: no bubbly mascots, candy colors, or gamified
  flourishes. Status color is information, not decoration.
- Not legacy enterprise gray: density without hierarchy, sub-12px text, and
  dated form chrome are failures too. Dense is fine; illegible is not.

## Design Principles

1. **Glare-legible first.** Every screen must read on a tablet in daylight:
   high contrast, large status cues, generous touch targets (44px floor).
   If a choice trades legibility for elegance, legibility wins.
2. **Status is the signal.** Good / Monitor / Action condition is the most
   important information on most screens. It must be unmistakable and never
   carried by color alone (pair with label, shape, or position for
   color-blind and glare safety).
3. **Physical truth.** The deck map and location model mirror the real vessel.
   The interface should match what the crew sees over the rail, so the map is
   spatial and concrete, not an abstract list.
4. **Density with hierarchy.** Shore views can be dense, but scale and weight
   contrast must always show what matters first. Never flat walls of equal text.
5. **Operational, not promotional.** Copy and layout serve a task in progress.
   Name the noun, name the action, get out of the way.

## Accessibility & Inclusion

- Target WCAG 2.1 AA minimum; push toward AAA contrast for condition status and
  body text given the glare context.
- Condition status never relies on hue alone: always paired with text label
  and/or shape so it survives sunlight, color blindness, and grayscale.
- Touch targets ≥44px (already tokenized as `--touch`) for gloved/on-deck use.
- Honor `prefers-reduced-motion`: motion is functional feedback only, with a
  static fallback. Nothing essential is gated behind an animation.
