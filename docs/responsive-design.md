# Responsive Design Patterns

This document describes the responsive design patterns used in the Gloski frontend.

## Breakpoints

We use Tailwind's default breakpoints:

| Breakpoint | Min Width | Typical Devices |
|------------|-----------|-----------------|
| (default)  | 0px       | Mobile phones   |
| `sm`       | 640px     | Large phones    |
| `md`       | 768px     | Tablets         |
| `lg`       | 1024px    | Laptops         |
| `xl`       | 1280px    | Desktops        |

**Primary breakpoint:** `md` (768px) - This is where most layouts switch from mobile to desktop.

## Mobile-First Approach

All styles are written mobile-first, with larger breakpoints adding complexity:

```tsx
// Mobile first: base styles are for mobile, md: adds desktop styles
<div className="p-4 md:p-6">           {/* Padding */}
<div className="text-lg md:text-xl">   {/* Text size */}
<div className="flex-col md:flex-row"> {/* Direction */}
<div className="w-full md:w-72">       {/* Width */}
```

## Common Patterns

### 1. Responsive Padding

```tsx
// Standard content padding
className="p-4 md:p-6"

// Header padding
className="px-4 py-3 md:px-6 md:py-4"

// Compact padding
className="p-3 md:p-4"
```

### 2. Responsive Text

```tsx
// Page titles
className="text-lg md:text-xl font-semibold"

// Descriptions
className="text-xs md:text-sm text-muted-foreground"

// Body text
className="text-sm md:text-base"
```

### 3. Responsive Gaps

```tsx
// Standard gap
className="gap-3 md:gap-4"

// Larger gap
className="gap-4 md:gap-6"
```

### 4. Hide/Show Elements

```tsx
// Hide on mobile, show on desktop
className="hidden md:flex"
className="hidden md:block"

// Show on mobile, hide on desktop
className="md:hidden"
className="flex md:hidden"
```

### 5. Responsive Widths

```tsx
// Full width on mobile, fixed on desktop
className="w-full md:w-72"
className="w-full md:w-96"

// Max width constraints
className="w-full max-w-md md:max-w-none"
```

### 6. Flex Direction Changes

```tsx
// Stack on mobile, row on desktop
className="flex flex-col md:flex-row"

// Row on mobile, column on desktop (rare)
className="flex flex-row md:flex-col"
```

## Layout Patterns

### Mobile Navigation (AppLayout)

The app uses a mobile drawer pattern:

```tsx
// layouts/AppLayout.tsx
<div className="h-screen flex flex-col md:flex-row">
  {/* Mobile header - visible only on mobile */}
  <header className="md:hidden flex items-center justify-between px-4 py-3 border-b">
    <Logo />
    <HamburgerButton onClick={toggleMobileSidebar} />
  </header>

  {/* Desktop sidebar - hidden on mobile */}
  <Sidebar className="hidden md:flex" />

  {/* Mobile drawer - controlled by state */}
  <MobileSidebar />

  {/* Main content */}
  <main className="flex-1 overflow-hidden">
    <Outlet />
  </main>
</div>
```

### Mobile Sidebar Drawer

```tsx
// layouts/Sidebar/Sidebar.tsx
export function MobileSidebar() {
  const { mobileSidebarOpen, setMobileSidebarOpen } = useSettingsStore()

  // Close on route change
  useEffect(() => {
    setMobileSidebarOpen(false)
  }, [location.pathname])

  // Close on Escape
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setMobileSidebarOpen(false)
    }
    document.addEventListener('keydown', handleEscape)
    return () => document.removeEventListener('keydown', handleEscape)
  }, [])

  // Prevent body scroll
  useEffect(() => {
    document.body.style.overflow = mobileSidebarOpen ? 'hidden' : ''
    return () => { document.body.style.overflow = '' }
  }, [mobileSidebarOpen])

  if (!mobileSidebarOpen) return null

  return (
    <div className="md:hidden fixed inset-0 z-50">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-background/80 backdrop-blur-sm"
        onClick={() => setMobileSidebarOpen(false)}
      />

      {/* Drawer */}
      <aside className="absolute inset-y-0 left-0 w-72 max-w-[85vw] flex flex-col border-r bg-background shadow-xl animate-in slide-in-from-left">
        {/* Navigation content */}
      </aside>
    </div>
  )
}
```

### Page Header (PageLayout)

```tsx
// layouts/PageLayout.tsx
<header className="border-b px-4 py-3 md:px-6 md:py-4 shrink-0">
  <div className="flex items-start md:items-center justify-between gap-3 flex-col sm:flex-row">
    <div className="min-w-0">
      <h1 className="text-lg md:text-xl font-semibold truncate">{title}</h1>
      {description && (
        <p className="text-xs md:text-sm text-muted-foreground mt-0.5 md:mt-1 line-clamp-2">
          {description}
        </p>
      )}
    </div>
    {actions && (
      <div className="flex items-center gap-2 flex-wrap shrink-0">
        {actions}
      </div>
    )}
  </div>
</header>
```

### Master-Detail Pattern

For two-panel layouts (Notes, Messages), use a view state to toggle between list and detail on mobile:

```tsx
// State
const [mobileView, setMobileView] = useState<'list' | 'detail'>('list')

// Selecting an item switches to detail view
const handleSelect = (id: string) => {
  setSelectedId(id)
  setMobileView('detail')
}

// Back button returns to list
const handleBack = () => {
  setMobileView('list')
}

// Layout
<div className="flex h-full">
  {/* List panel */}
  <div className={cn(
    'w-full md:w-72 border-r flex flex-col',
    mobileView === 'detail' && 'hidden md:flex'  // Hide on mobile when viewing detail
  )}>
    {/* List content */}
  </div>

  {/* Detail panel */}
  <div className={cn(
    'flex-1 flex flex-col',
    mobileView === 'list' && 'hidden md:flex'  // Hide on mobile when viewing list
  )}>
    {/* Mobile back button */}
    <div className="md:hidden border-b px-3 py-2">
      <Button variant="ghost" size="sm" onClick={handleBack}>
        <ArrowLeft className="h-4 w-4 mr-2" />
        Back
      </Button>
    </div>

    {/* Detail content */}
  </div>
</div>
```

### Responsive Button Labels

Hide text on mobile, show icon only:

```tsx
// Icon only on mobile, icon + text on desktop
<Button>
  <Plus className="h-4 w-4 md:mr-2" />
  <span className="hidden md:inline">New Item</span>
</Button>

// Or use size prop
<Button size="sm" className="md:size-default">
  <Plus className="h-4 w-4 md:mr-2" />
  <span className="hidden md:inline">New Item</span>
</Button>
```

## Sidebar State Management

```tsx
// features/settings/stores/settings.ts
interface SettingsState {
  // Desktop sidebar collapse
  sidebarCollapsed: boolean
  toggleSidebar: () => void

  // Mobile sidebar drawer
  mobileSidebarOpen: boolean
  setMobileSidebarOpen: (open: boolean) => void
  toggleMobileSidebar: () => void
}
```

## Testing Responsive Designs

1. **Browser DevTools**: Use responsive mode (Cmd+Shift+M in Chrome)
2. **Key widths to test**:
   - 375px (iPhone SE)
   - 430px (iPhone Pro Max)
   - 768px (iPad)
   - 1024px (Small laptop)
   - 1440px (Desktop)

## Checklist for New Pages

When creating a new page, ensure:

- [ ] Page uses `PageLayout` component
- [ ] Title uses `text-lg md:text-xl`
- [ ] Content padding is `p-4 md:p-6` (or use `noPadding` prop)
- [ ] Fixed-width sidebars use `w-full md:w-{size}`
- [ ] Two-panel layouts implement master-detail pattern
- [ ] Buttons have responsive labels where needed
- [ ] Text truncates properly with `truncate` or `line-clamp-*`
- [ ] Gaps use responsive classes `gap-3 md:gap-4`

## Animation Classes

For mobile drawer animations:

```tsx
// Slide in from left
className="animate-in slide-in-from-left duration-200"

// Fade in backdrop
className="animate-in fade-in duration-200"
```

## Z-Index Scale

| Z-Index | Usage |
|---------|-------|
| `z-40` | Sticky headers |
| `z-50` | Mobile drawer, modals |
| `z-60` | Tooltips, dropdowns |
