# HomeStock Design System Reference

## Source of Truth

Inspect these files before making larger UI changes:

- `web/src/index.css`
- `web/src/components/layout/AppLayout.tsx`
- `web/src/components/layout/Sidebar.tsx`
- `web/src/components/layout/Header.tsx`
- `web/src/components/ui/button.tsx`
- `web/src/components/ui/input.tsx`
- `web/src/components/ui/select.tsx`
- `web/src/components/ui/dialog.tsx`
- `web/src/pages/inventory/InventoryPage.tsx`
- `web/src/pages/inventory/ItemFormDialog.tsx`

## Visual Direction

The product uses a practical inventory-management style:

- green primary brand
- soft gray surfaces
- strong contrast on actions and metrics
- compact desktop-first layout that still works on narrow widths
- understated shadows and rounded corners

Avoid flashy gradients, large hero sections, oversized cards, or consumer-style decorative patterns unless the user explicitly asks for a redesign.

## Color Tokens

Use semantic tokens instead of raw hex values whenever possible.

Primary and status intent:

- `primary`: main CTA, active state, brand emphasis
- `secondary`: supporting positive emphasis
- `tertiary`: warning, expiring stock, attention
- `error` / `destructive`: deletion, invalid state, hard failure

Surface hierarchy:

- page background: `bg-background`
- major cards: `bg-surface-container-lowest`
- softer blocks and controls: `bg-surface-container-low` or `bg-surface-container`
- subtle separators: `border-outline-variant/20` or `border-outline-variant/10`

Useful existing patterns:

- brand gradient: `bg-gradient-primary`
- glass header: `glass-effect`

## Typography

Use the existing sans stack from `index.css`:

- `"Inter", "PingFang SC", sans-serif`

Common sizes already present in the app:

- page title: `text-2xl font-extrabold tracking-tight`
- section or dialog title: `text-lg` to `text-xl`, semibold or extrabold depending on density
- body/control text: `text-sm`
- helper/meta text: `text-xs text-on-surface-variant`
- table headers / metric labels: `text-xs font-bold uppercase tracking-widest`

Do not add a second font family for new management screens.

## Layout Model

Global shell:

- fixed left sidebar
- fixed top header
- main content offset by sidebar width and header height
- content container inside `main` with `p-6`

Implications:

- do not use full-screen pages that ignore the shell
- do not add custom margins that fight the sidebar/header offsets
- keep page sections vertically stacked with `gap-6`

Typical page shape:

1. toolbar row with title, helper copy, filters, utility actions, and one primary CTA
2. dominant data card or task card
3. summary cards or secondary sections

## Control Patterns

### Buttons

Use existing variants from `web/src/components/ui/button.tsx`:

- `default`: main CTA
- `outline`: secondary action or neutral utility
- `ghost`: low-emphasis row action
- `destructive`: destructive confirmation

Use sizes intentionally:

- default buttons for page actions
- `size="icon"` for compact utilities
- smaller custom width/height only when matching an existing dense table pattern

### Inputs and selects

Use the built-in control height `h-10`.

Common usage:

- search fields: icon inside input, `pl-9`
- filters: fixed-width `SelectTrigger`
- dialog forms: standard `Input` and `Select` with `space-y-1.5`

### Cards and tables

Use a single dominant card for dense operational data:

- `rounded-xl`
- soft border
- `shadow-sm`
- `overflow-hidden` when table pagination or sticky internal sections are present

Use tables for structured item data. Put row actions at the far right and avoid excessive inline buttons.

### Badges and status

Use `Badge` for status and category cues. Keep status mapping semantic:

- normal: primary/default
- warning: outline with tertiary accent
- destructive: destructive

## Interaction Patterns

- hide low-priority row actions until hover when density matters
- keep one obvious primary action per toolbar or dialog footer
- confirm destructive actions in a dialog
- show loading inline where the user expects it, not in a disconnected banner
- reset pagination or filtered state when the underlying query changes

## Copy Style

- use short operational labels
- keep helper copy to one sentence
- avoid verbose explanatory paragraphs in the page body
- keep terminology consistent with inventory management and the existing Chinese labels

## Mobile and Responsiveness

The app is desktop-first, but new UI still needs to degrade cleanly:

- allow toolbar rows to wrap with `flex-wrap`
- keep search and filters usable on narrower widths
- avoid multi-column forms unless the fields can stack acceptably
- prefer responsive grids like `grid-cols-1 md:grid-cols-3`

## Avoid

- unrelated color systems
- floating action buttons
- oversized hero banners
- deep nested cards inside cards unless there is a strong information hierarchy
- multiple competing primary actions in the same row
