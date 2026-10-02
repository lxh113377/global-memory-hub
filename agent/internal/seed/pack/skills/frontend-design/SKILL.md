---
name: frontend-design
description: Create distinctive, production-grade frontend interfaces with high design quality. Use this skill when the user asks to build web components, pages, artifacts, posters, or applications (examples include websites, landing pages, dashboards, React components, HTML/CSS/layouts, or when styling/beautifying any web UI). Generates creative, polished code and UI design that avoids generic AI aesthetics.
version: 2.1.0
license: Complete terms in LICENSE.txt
---

This skill guides creation of distinctive, production-grade frontend interfaces that avoid generic "AI slop" aesthetics. Implement real working code with exceptional attention to aesthetic details and creative choices.

The user provides frontend requirements: a component, page, application, or interface to build. They may include context about the purpose, audience, or technical constraints.

## Section 0: Pre-Flight Design Read (MANDATORY)

Before ANY code, infer the design context and output a one-line "Design Read":

```
Design Read: [page type] | [vibe words ×3] | [audience] | [brand assets if any] | [quiet constraint]
```

Example:
```
Design Read: SaaS pricing page | bold, trustworthy, spacious | enterprise CTO | blue logo | must work in dark mode
```

This forces the agent to "read the room" before generating. Skip this = generic output guaranteed.

## Section 1: Three Dials Configuration

Every design sits on three axes. Infer values from the Design Read, then set them explicitly:

### DESIGN_VARIANCE (1-10)
Symmetry/safety ←→ Artsy chaos/surprise

| Signal | Value |
|--------|-------|
| Enterprise / B2B / finance | 2-4 |
| SaaS / startup / product | 4-6 |
| Creative agency / portfolio | 6-8 |
| Art project / experimental / music | 8-10 |
| Government / healthcare / legal | 1-3 |

### MOTION_INTENSITY (1-10)
Static/still ←→ Cinematic/kinetic

| Signal | Value |
|--------|-------|
| Dashboard / data-heavy / productivity | 1-3 |
| Marketing page / landing | 3-5 |
| Portfolio / showcase / creative | 5-7 |
| Gaming / entertainment / music | 7-10 |
| Accessibility-sensitive audience | 1-2 (respect prefers-reduced-motion) |

### VISUAL_DENSITY (1-10)
Airy/minimal ←→ Cockpit/dense

| Signal | Value |
|--------|-------|
| Luxury / premium / editorial | 2-4 |
| Landing page / hero section | 3-5 |
| SaaS app / dashboard | 5-7 |
| Developer tool / admin panel | 7-9 |
| Monitoring / trading / analytics | 8-10 |

**Rule**: Write the three dial values before coding. They constrain all downstream choices.

## Section 2: Design System Honest Map

If the brief reads as one of these, use the OFFICIAL package — do not recreate CSS by hand:

| Brief reads as | Use | Do NOT |
|----------------|-----|--------|
| Windows enterprise | Fluent UI | Hand-roll Fluent-like CSS |
| Google ecosystem | Material Design 3 | Fake Material with custom CSS |
| IBM / enterprise data | Carbon Design | Approximate Carbon |
| Developer-facing / GitHub | Primer | GitHub-lookalike |
| UK government / public | GOV.UK Design System | Custom "government-looking" |
| Apple ecosystem | Human Interface Guidelines | Fake Apple aesthetics |

**Honesty rule**: If you're making something that "looks like" an existing design system, use the real system. Faking it = uncanny valley.

## Section 3: Design Thinking

Before coding, commit to a BOLD aesthetic direction:
- **Purpose**: What problem does this interface solve? Who uses it?
- **Tone**: Pick an extreme: brutally minimal, maximalist chaos, retro-futuristic, organic/natural, luxury/refined, playful/toy-like, editorial/magazine, brutalist/raw, art deco/geometric, soft/pastel, industrial/utilitarian, etc.
- **Constraints**: Technical requirements (framework, performance, accessibility).
- **Differentiation**: What makes this UNFORGETTABLE? What's the one thing someone will remember?

**CRITICAL**: Choose a clear conceptual direction and execute it with precision. Bold maximalism and refined minimalism both work — the key is intentionality, not intensity.

## Section 4: Frontend Aesthetics Guidelines

### Typography
Choose fonts that are beautiful, unique, and interesting.

**Banned defaults** — never use as first choice:

| Font | Why banned | When OK |
|------|-----------|---------|
| Inter | Default of defaults, zero personality | Only if the user explicitly requests it |
| Roboto | Google default, overused | Only in a Material Design context |
| Arial, Segoe UI, system-ui | OS default, no design intent | Only for native-OS feel, by design |
| Space Grotesk | AI's "creative" default | Never — use Syne / Clash Display / Bricolage |
| Fraunces, Instrument_Serif | The "brief = serif" AI tell | Only if the brief explicitly calls editorial |

**Good choices** (rotate, don't repeat across generations):
- Display: Playfair Display, Syne, Clash Display, Cabinet Grotesk, Bricolage Grotesque
- Body: DM Sans, Plus Jakarta Sans, Satoshi, General Sans, Switzer
- Mono: JetBrains Mono, Fira Code, Berkeley Mono, Fragment Mono

Pair a distinctive display font with a refined body font. Never use the same font for both.

### Color & Theme

**Banned palettes** (AI defaults — never use):
- Purple/blue gradient on white (#667eea → #764ba2 "Lila Rule")
- Purple glow on dark mesh background
- Warm beige/brass/oxblood/ochre "premium consumer" palette
- Three equal-weight accent colors
- slate-900 text on white background with purple accents

**Good approaches**:
- Dominant color with ONE sharp accent (not three equal weights)
- CSS variables for consistency: `--color-primary`, `--color-accent`, `--color-surface`
- Dark themes: use near-black (#0a0a0a, #111) not pure #000
- Light themes: use warm off-white (#fafaf9, #f5f5f0) not pure #fff
- Test contrast ratios: minimum 4.5:1 for body text (WCAG AA)

### Motion
- Use animations for effects and micro-interactions
- Prioritize CSS-only solutions for HTML; Motion library for React
- High-impact moments: one well-orchestrated page load with staggered reveals > scattered micro-interactions
- Scroll-triggering and hover states that surprise
- **ALWAYS respect `prefers-reduced-motion: reduce`** — provide instant transitions

### Spatial Composition
- Unexpected layouts: asymmetry, overlap, diagonal flow, grid-breaking elements
- Generous negative space OR controlled density (match VISUAL_DENSITY dial)

**Banned layouts**:

| Pattern | Why banned | Alternative |
|---------|-----------|-------------|
| Three equal-width cards in a row | AI's default feature section | Vary sizes: 1 large + 2 small, or asymmetric grid |
| Centered hero over dark mesh bg | AI landing page #1 | Split hero, diagonal, overlapping elements |
| Glassmorphism everywhere | 2023 AI trend, now cliche | Use sparingly or skip entirely |
| Gradient text on gradient bg | Unreadable + AI tell | Gradient text on solid bg, or solid text on gradient |

### Backgrounds & Visual Details
- Create atmosphere and depth rather than defaulting to solid colors
- Contextual effects: gradient meshes, noise textures, geometric patterns, layered transparencies, dramatic shadows, decorative borders, grain overlays
- Match VISUAL_DENSITY dial: airy = fewer effects, dense = layered textures

## Section 5: Anti-Slop Verification (Post-Generation)

After generating code, run this checklist:

1. **Font check**: Is the first font Inter/Roboto/Space Grotesk? → Replace.
2. **Color check**: Is the primary color purple (#667eea or similar)? → Replace.
3. **Layout check**: Are there three equal-width cards in a row? → Break symmetry.
4. **Dark mode check**: Is it just white-on-black with purple accents? → Redesign.
5. **Motion check**: Are animations just `fade-in` on everything? → Add variety (slide, scale, stagger).
6. **Differentiation check**: Could this be confused with another AI-generated page? → Add unique element.

If any check fails, fix before delivering.

## Section 6: Implementation

Implement working code (HTML/CSS/JS, React, Vue, etc.) that is:
- Production-grade and functional
- Visually striking and memorable
- Cohesive with a clear aesthetic point-of-view
- Meticulously refined in every detail

**IMPORTANT**: Match implementation complexity to the aesthetic vision. Maximalist designs need elaborate code with extensive animations and effects. Minimalist designs need restraint, precision, and careful attention to spacing, typography, and subtle details.

**Accessibility**: Every design must meet WCAG 2.1 AA minimum:
- Color contrast ≥ 4.5:1 (body), ≥ 3:1 (large text)
- Keyboard navigable (all interactive elements focusable)
- Screen reader compatible (semantic HTML, ARIA where needed)
- `prefers-reduced-motion` respected
- `prefers-color-scheme` considered

Remember: You are capable of extraordinary creative work. Don't hold back — show what can truly be created when thinking outside the box and committing fully to a distinctive vision.

## Section 7: Redesign Audit Protocol

Use this when auditing or restyling an **existing** design rather than building a new one:

1. Read or screenshot the current design.
2. **Score each dial** (1-10) for what the design currently exhibits.
3. **Identify banned patterns** from Section 4 that it violates.
4. **Check design-system honesty** — is it faking a system from Section 2?
5. Emit the audit report before proposing changes:

```
Design Audit:
  Current dials: VARIANCE=X | MOTION=Y | DENSITY=Z
  Target dials:  VARIANCE=X' | MOTION=Y' | DENSITY=Z'
  Banned patterns found: [list]
  Design system: [honest / faking X / none]
  Top 3 fixes: [prioritized list]
```

Declare dials in both paths with the same format: `Dials: VARIANCE=6 | MOTION=4 | DENSITY=5`

## Version history

- **2.1.0** (2026-09-29) — Absorbed the retired `design-taste` skill so dials and bans have one owner.
  Added Section 7 (Redesign Audit Protocol), why-banned / when-OK columns to the font ban list, and a
  banned-layout table. Merged content originally adapted from Leonxlnx/taste-skill.
- **2.0.0** (2026-09-28) — Added Pre-Flight Design Read, Three Dials, Design System Honest Map,
  banned palette/font tables, Anti-Slop Verification, WCAG requirements.
