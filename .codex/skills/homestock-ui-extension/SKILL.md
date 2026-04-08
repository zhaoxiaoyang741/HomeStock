---
name: homestock-ui-extension
description: Design and implement new HomeStock web pages, buttons, dialogs, cards, filters, tables, and feature flows so they match the existing React and Tailwind UI in `web/src`. Use when Codex adds or revises frontend screens, controls, empty states, or feature entry points for this project and needs to stay aligned with the current layout, green palette, spacing, typography, and interaction patterns.
---

# HomeStock UI Extension

Extend the existing HomeStock web UI instead of inventing a new visual language. Treat the current implementation in `web/src` as the source of truth.

## Workflow

1. Inspect the current shell before editing.
Read `web/src/index.css`, `web/src/components/layout/AppLayout.tsx`, `web/src/components/layout/Sidebar.tsx`, `web/src/components/layout/Header.tsx`, and the closest existing page.

2. Classify the requested change before writing code.
Choose one of these shapes first: list page, dashboard section, settings block, dialog form, toolbar action, or row-level action.

3. Reuse existing primitives before creating new ones.
Prefer `Button`, `Input`, `Select`, `Badge`, `Dialog`, `Table`, and existing layout containers from `web/src/components/ui`.

4. Preserve the current shell and hierarchy.
Keep the fixed sidebar, fixed header, and scrollable main content pattern from `AppLayout`. Do not add a second app shell or a conflicting navigation model.

5. Add all required states for the feature.
Handle loading, empty, error, and destructive confirmation states when relevant. Do not ship a new page or action with only the happy path.

6. Verify visual consistency after implementation.
Check light and dark theme behavior, spacing around the toolbar and main card, and whether the new feature still feels like part of the same product.

## Non-Negotiables

- Use the project tokens from `web/src/index.css`. Do not introduce unrelated brand colors.
- Keep the product's green-first hierarchy: primary actions and highlights use `primary`, warning states use `tertiary`, destructive states use `error` or `destructive`.
- Keep surface layering soft and neutral. Prefer `bg-surface-container-lowest`, `bg-surface-container-low`, and borders like `border-outline-variant/20`.
- Match the existing type scale:
  - page title: `text-2xl font-extrabold tracking-tight`
  - card metric: `text-xl font-extrabold`
  - table/meta labels: `text-xs font-bold uppercase tracking-widest`
  - standard controls and body text: `text-sm`
- Preserve the current corner language: `rounded-lg` for controls, `rounded-xl` for larger cards and dialogs when already used.
- Keep the UI compact and operational. This app favors efficient management screens over marketing-style layouts.
- Use Chinese UI copy unless the surrounding module is already English.

## Placement Rules

### New page

Use the established page skeleton:

```tsx
<div className="flex flex-col h-full gap-6">
```

Start with a toolbar row:
- left: page title and one-line helper text
- right: filters, utility actions, then one primary action

Put the main task area in one dominant card or table block before any summary cards.

### New button or action

Place the action by scope:
- global action: `Header`
- page-level action: page toolbar
- record-level action: table row or dropdown menu
- destructive action: confirmation dialog or clearly destructive context

Prefer icon + label for important creation or submission actions. Use icon-only buttons for compact utilities such as refresh, theme toggle, collapse, and obvious row actions.

### New form or dialog

Use `Dialog` primitives and keep dialog width close to `sm:max-w-lg` unless the form truly needs more space.

Use:
- `space-y-1.5` for label + field groups
- `grid grid-cols-2 gap-3` for paired fields
- outline secondary action on the left, primary confirm action on the right

### New data display

For operational data, prefer a table inside a bordered card. Put status in `Badge`, metadata in subdued text, and row actions on hover when density matters.

If the new feature is summary-first, use compact metric cards that mirror the inventory summary pattern before introducing charts or heavier visuals.

## Decision Rules

- If the request can fit an existing page, extend that page instead of creating a new top-level route.
- If a new route is necessary, add it to the existing sidebar/navigation model with the same density and icon treatment.
- If a new visual pattern conflicts with the current codebase, follow the codebase, not generic design advice.
- If a request is ambiguous, optimize for clarity, scanability, and low interaction cost.

## References

Read these files only when needed:

- `references/design-system.md`
Use for tokens, spacing, typography, component states, and layout constraints.

- `references/page-blueprints.md`
Use for recommended structures when adding list pages, dashboards, settings sections, dialogs, and action entry points.
