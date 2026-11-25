# Responsive Design Patterns

Guidelines for keeping the Gloski dashboard usable across mobile phones,
tablets, and desktop screens.

## Breakpoints

We use Tailwind's default mobile-first breakpoints:

| Breakpoint | Min Width | Typical Devices |
|---|---|---|
| (default) | `0px` | Mobile phones |
| `sm` | `640px` | Large phones |
| `md` | `768px` | Tablets |
| `lg` | `1024px` | Laptops |
| `xl` | `1280px` | Desktops |

**Primary breakpoint:** `md` (`768px`) — most layouts switch from mobile to
desktop here.

## Mobile-first layout patterns

- **Padding & spacing:** start compact on small screens (`p-4`, `gap-3`) and
  step up at `md:` (`md:p-6`, `md:gap-4` or `md:gap-6`).
- **Typography & truncation:** scale headings modestly (`text-lg md:text-xl`)
  and use truncation or line clamping so long device names, paths, and errors
  never push layouts sideways.
- **Stacking:** flex containers stack vertically by default and switch to rows
  at `sm:` or `md:` as space allows.
- **Navigation drawer:** below `md`, the desktop sidebar hides and a compact
  mobile header opens a slide-over navigation drawer that closes on route
  change or Escape.
- **Page header:** page titles, descriptions, and action buttons wrap cleanly
  on narrow viewports and sit inline on desktop.
- **Two-panel views:** narrow screens toggle between list and detail views with
  a back button rather than squeezing both panes side by side.
- **Responsive action labels:** secondary labels on toolbar buttons can hide on
  mobile (icon-only with accessible labels) while keeping full text on desktop.
- **Tables:** wrap wide tables in horizontal/vertical scroll containers with
  sticky headers rather than clipping cell content.

## Z-index scale

| Z-Index | Usage |
|---|---|
| `z-40` | Sticky headers |
| `z-50` | Mobile drawer, modals |
| `z-60` | Tooltips, dropdowns |

## Checklist for new pages

- Uses the standard page layout shell with a responsive header and action bar
- Uses mobile-first spacing, typography, and stacking
- Fixed-width side panels expand to full width below `md`
- Two-panel layouts support mobile list/detail switching
- Text truncates or wraps cleanly without horizontal page overflow
- Wide tables scroll inside a bounded container with sticky headers

## Testing key widths

Check layouts in browser DevTools at `375px` (small phone), `430px` (large
phone), `768px` (tablet), `1024px` (small laptop), and `1440px` (desktop).
