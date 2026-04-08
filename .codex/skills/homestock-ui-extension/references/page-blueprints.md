# HomeStock Page Blueprints

## List Page

Use this for inventory, shopping, history, and similar record-heavy modules.

Structure:

1. page header block
2. filter/action toolbar
3. main table card
4. pagination inside the card if needed
5. summary cards below

Implementation cues:

- outer container: `flex flex-col h-full gap-6`
- title block: `text-2xl font-extrabold` plus `text-xs` helper copy
- toolbar: `flex items-center gap-3 flex-wrap`
- right action group: `flex items-center gap-2 flex-wrap`
- main card: `rounded-xl border border-outline-variant/20 bg-surface-container-lowest shadow-sm overflow-hidden`

Use this shape by default when adding a new operational page.

## Dashboard Section

Use this when the page is insight-first rather than record-first.

Structure:

1. title and helper copy
2. metrics row
3. one or two larger cards for trends, warnings, or grouped summaries
4. optional recent activity table

Rules:

- keep metric cards compact
- use icons with soft tinted circles
- reserve strong color for statuses that require attention
- do not turn the dashboard into a marketing landing page

## Settings Section

Use this when adding configurable behavior.

Structure:

1. title and short description
2. stacked setting cards grouped by theme
3. controls inside each card with clear labels and helper text
4. save/apply actions at the bottom or card footer

Rules:

- favor grouped cards over one giant form
- use switches, selects, and inputs with consistent spacing
- expose destructive settings in isolated sections

## Dialog Form

Use this for add/edit flows that should not navigate away from the list page.

Structure:

1. `DialogHeader` with concise title
2. form body with `space-y-4`
3. paired fields in `grid grid-cols-2 gap-3` where appropriate
4. inline error message near the footer
5. `DialogFooter` with cancel + confirm

Rules:

- keep the form focused on one task
- avoid wizard-style multi-step flows unless the task complexity demands it
- prefer editing in place via dialog for CRUD on existing records

## Button Placement Guide

Choose the narrowest scope that fits the action:

- `Header`: theme, sync, notifications, app-wide actions
- page toolbar: create, import, export, refresh, page-scoped filters
- table row: edit, delete, quick actions on one record
- dialog footer: submit, cancel, confirm destructive action

If an action does not need persistent visibility, do not promote it to the header.

## Feature Add Checklist

Before finishing a UI change, verify:

1. the feature uses existing tokens and primitives
2. the primary action is obvious and singular
3. loading, empty, and error states exist
4. destructive behavior is confirmed
5. spacing matches the rest of the app
6. the result still fits the sidebar + header shell
