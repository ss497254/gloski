# Frontend Architecture

The Gloski frontend is a React 19 application built with TypeScript, Vite, and TailwindCSS v4.

## Tech Stack

| Technology | Purpose |
|------------|---------|
| React 19 | UI framework |
| TypeScript | Type safety |
| Vite | Build tool & dev server |
| TailwindCSS v4 | Styling |
| shadcn/ui | UI component library |
| Zustand | State management |
| xterm.js | Terminal emulator |
| Lucide React | Icons |

## Directory Structure

```
web/src/
├── app/                    # Application configuration
│   ├── feature-registry.ts # Central feature definitions
│   └── routes.tsx          # Route configuration
│
├── features/               # Feature modules (isolated)
│   ├── dashboard/
│   ├── servers/
│   ├── files/
│   ├── terminal/
│   ├── jobs/
│   ├── search/
│   ├── downloads/
│   ├── bookmarks/
│   ├── notes/
│   ├── snippets/
│   ├── messages/
│   ├── activity/
│   └── settings/
│
├── shared/                 # Shared across features
│   ├── components/         # Reusable UI components
│   ├── hooks/              # Reusable hooks
│   ├── lib/                # Utilities
│   └── services/           # API client
│
├── layouts/                # Layout components
│   ├── AppLayout.tsx       # Main app shell
│   ├── PageLayout.tsx      # Page wrapper
│   └── Sidebar/            # Navigation sidebar
│
└── ui/                     # Pure UI components (shadcn)
```

## Feature Module Pattern

Each feature is self-contained with its own components, hooks, pages, and stores.

```
features/{name}/
├── components/     # Feature-specific components
├── hooks/          # Feature-specific hooks
├── pages/          # Page components
├── stores/         # Zustand stores
├── lib/            # Feature-specific utilities (optional)
└── index.ts        # Public exports
```

### Example: Files Feature

```
features/files/
├── components/
│   ├── FileEntryItem.tsx    # Single file/folder row (context menu: pin, VS Code)
│   ├── FilePreview.tsx      # File preview router
│   ├── FileDialogs.tsx      # Create/Rename/Delete dialogs
│   ├── FilesToolbar.tsx     # Toolbar with actions
│   ├── FilesSidebar.tsx     # Quick access & pinned folders sidebar
│   ├── CodeEditor.tsx       # Native code editor
│   ├── DropZone.tsx         # Drag-drop upload
│   └── previews/
│       ├── ImagePreview.tsx
│       ├── TextPreview.tsx
│       ├── PdfPreview.tsx
│       └── BinaryPreview.tsx
├── hooks/
│   └── useFilesKeyboard.ts  # Keyboard shortcuts
├── stores/
│   └── files.ts             # Zustand store (syncs pinned folders with server)
├── lib/
│   └── file-types.ts        # File type detection
├── pages/
│   └── FilesPage.tsx        # Main page component
└── index.ts
```

## Feature Registry

Features are registered in `app/feature-registry.ts`:

```typescript
export interface FeatureDefinition {
  id: string
  name: string
  icon: LucideIcon
  path: string
  component: LazyExoticComponent<ComponentType>
  section: 'main' | 'workspace' | 'server'
  serverScoped?: boolean  // Requires server context
  shortcut?: string       // Keyboard shortcut (e.g., 'g d')
}
```

### Feature Sections

- **main**: Dashboard, Servers, Settings
- **workspace**: Bookmarks, Notes, Snippets, Messages, Activity
- **server**: Files, Terminal, Jobs, Search, Downloads (require server selection)

## State Management (Zustand)

Each feature has its own Zustand store in `stores/`.

### Store Pattern

```typescript
// features/notes/stores/notes.ts
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface Note {
  id: string
  title: string
  content: string
  folder: string
  pinned: boolean
  createdAt: Date
  updatedAt: Date
}

interface NotesState {
  notes: Note[]
  addNote: (note: Omit<Note, 'id' | 'createdAt' | 'updatedAt'>) => void
  updateNote: (id: string, updates: Partial<Note>) => void
  deleteNote: (id: string) => void
}

export const useNotesStore = create<NotesState>()(
  persist(
    (set) => ({
      notes: [],
      addNote: (note) =>
        set((state) => ({
          notes: [
            ...state.notes,
            {
              ...note,
              id: generateId(),
              createdAt: new Date(),
              updatedAt: new Date(),
            },
          ],
        })),
      updateNote: (id, updates) =>
        set((state) => ({
          notes: state.notes.map((n) =>
            n.id === id ? { ...n, ...updates, updatedAt: new Date() } : n
          ),
        })),
      deleteNote: (id) =>
        set((state) => ({
          notes: state.notes.filter((n) => n.id !== id),
        })),
    }),
    { name: 'gloski-notes' }
  )
)
```

### Key Stores

| Store | Location | Purpose |
|-------|----------|---------|
| `useServersStore` | `features/servers/` | Server connections & status |
| `useSettingsStore` | `features/settings/` | Theme, sidebar, preferences |
| `useNotesStore` | `features/notes/` | Notes data |
| `useBookmarksStore` | `features/bookmarks/` | Bookmarks data |
| `useMessagesStore` | `features/messages/` | Notifications |
| `useFilesStore` | `features/files/` | Pinned folders (server-synced), view prefs |

## API Client

The API client is in `shared/services/api.ts`.

### Usage

```typescript
import { createServerApi } from '@/shared/services/api'

// In a component or hook
const server = useServersStore((s) => s.servers.find(s => s.id === serverId))
const api = createServerApi(server)

// Make API calls
const stats = await api.stats()
const files = await api.listFiles('/home')
const content = await api.readFile('/etc/hosts')
```

### SDK Client Usage

The frontend uses `@gloski/sdk` client stored per-server:

```typescript
// Get client for current server
const client = server.getClient()

// System
const stats = await client.system.getStats()
const processes = await client.system.getProcesses(100)

// Files
const { path, entries } = await client.files.list('/home/user')
const { content } = await client.files.read('/etc/hosts')
await client.files.write('/tmp/test.txt', 'content')
await client.files.mkdir('/tmp/folder')
await client.files.delete('/tmp/test.txt')
await client.files.rename('/tmp/old.txt', '/tmp/new.txt')
await client.files.upload('/tmp', file)
const url = client.files.getDownloadUrl('/tmp/file.zip')

// Pinned folders (server-synced)
const { folders, home_dir } = await client.files.pinned.list()
const pinned = await client.files.pinned.pin('/home/user/projects', 'Projects')
await client.files.pinned.unpin(pinned.id)

// Jobs
const jobs = await client.jobs.list()
const job = await client.jobs.start('bun run build', '/home/user/project')
const logs = await client.jobs.getLogs(job.id)
await client.jobs.stop(job.id)

// Systemd
const units = await client.systemd.list(true) // user mode
await client.systemd.start('my-service.service')
const logs = await client.systemd.getLogs('my-service.service')

// Terminal
const term = client.terminal.connect({ cwd: '/home/user' })
```

## Shared Components

Located in `shared/components/`:

| Component | Purpose |
|-----------|---------|
| `SearchInput` | Search input with icon and clear button |
| `FilterSidebar` | Sidebar with filter navigation |
| `EmptyState` | Empty state placeholder with icon |
| `StatCard` | Stats display card |
| `ProgressBar` | Progress indicators |
| `PageLoader` | Full-page loading spinner |
| `CommandPalette` | Global command palette (Cmd+K) |

### Example Usage

```tsx
import { SearchInput, FilterSidebar, EmptyState } from '@/shared/components'

function MyPage() {
  const [search, setSearch] = useState('')
  const [filter, setFilter] = useState('all')
  
  return (
    <div className="flex">
      <div className="w-64 border-r">
        <SearchInput
          value={search}
          onChange={setSearch}
          placeholder="Search..."
        />
        <FilterSidebar
          selected={filter}
          onSelect={setFilter}
          allItem={{ label: 'All Items', icon: Folder, count: 10 }}
          sections={[
            { title: 'Categories', items: [...] }
          ]}
        />
      </div>
      <div className="flex-1">
        {items.length === 0 ? (
          <EmptyState
            icon={FileText}
            title="No items"
            description="Create your first item"
          />
        ) : (
          // ... render items
        )}
      </div>
    </div>
  )
}
```

## Shared Hooks

Located in `shared/hooks/`:

| Hook | Purpose |
|------|---------|
| `useFilter` | Generic filtering, searching, sorting |
| `useDialog` | Dialog open/close state |
| `useSelection` | Master-detail selection |
| `useAsync` | Async operation state |
| `useInterval` | Polling with cleanup |
| `useServer` | Get current server from route |

### useFilter Example

```typescript
const {
  items: filteredNotes,
  search,
  setSearch,
  filter,
  setFilter,
  sort,
  setSort,
} = useFilter(notes, {
  searchFields: ['title', 'content'],
  filterFn: (note, filter) => filter === 'all' || note.folder === filter,
  sortOptions: {
    'date-desc': (a, b) => b.updatedAt - a.updatedAt,
    'date-asc': (a, b) => a.updatedAt - b.updatedAt,
    'title': (a, b) => a.title.localeCompare(b.title),
  },
  defaultSort: 'date-desc',
})
```

## Layouts

### AppLayout

The main application shell with sidebar and mobile navigation.

```tsx
// layouts/AppLayout.tsx
export function AppLayout() {
  return (
    <div className="h-screen flex flex-col md:flex-row">
      {/* Mobile header (hidden on desktop) */}
      <header className="md:hidden ...">
        <Logo />
        <HamburgerButton />
      </header>
      
      {/* Desktop sidebar (hidden on mobile) */}
      <Sidebar className="hidden md:flex" />
      
      {/* Mobile drawer */}
      <MobileSidebar />
      
      {/* Main content */}
      <main className="flex-1 overflow-hidden">
        <Outlet />
      </main>
    </div>
  )
}
```

### PageLayout

Wrapper for consistent page structure.

```tsx
// layouts/PageLayout.tsx
interface PageLayoutProps {
  title: string
  description?: string
  actions?: ReactNode
  children: ReactNode
  noPadding?: boolean
}

// Usage
<PageLayout
  title="Files"
  description="/home/user"
  actions={<Button>Upload</Button>}
>
  {/* Page content */}
</PageLayout>
```

## Styling

### TailwindCSS v4

The project uses TailwindCSS v4 with the new CSS-first configuration.

```css
/* index.css */
@import "tailwindcss";

@theme {
  --color-background: oklch(1 0 0);
  --color-foreground: oklch(0.145 0 0);
  /* ... more theme variables */
}
```

### Common Patterns

```tsx
// Responsive padding
className="p-4 md:p-6"

// Responsive text
className="text-lg md:text-xl"

// Hide on mobile/desktop
className="hidden md:flex"  // Hidden on mobile
className="md:hidden"       // Hidden on desktop

// Responsive widths
className="w-full md:w-72"

// Flex direction change
className="flex flex-col md:flex-row"
```

### Custom Scrollbars

```css
/* Thin scrollbar */
.scrollbar-thin { /* 6px */ }

/* Hide scrollbar */
.scrollbar-none { /* hidden */ }

/* Auto-hide scrollbar */
.scrollbar-auto-hide { /* visible on hover */ }
```

## Adding a New Feature

1. **Create feature folder:**
   ```
   web/src/features/my-feature/
   ├── components/
   ├── hooks/
   ├── pages/
   │   └── MyFeaturePage.tsx
   ├── stores/
   │   └── my-feature.ts
   └── index.ts
   ```

2. **Create the page:**
   ```tsx
   // pages/MyFeaturePage.tsx
   import { PageLayout } from '@/layouts'
   
   export function MyFeaturePage() {
     return (
       <PageLayout title="My Feature">
         {/* Content */}
       </PageLayout>
     )
   }
   ```

3. **Export from index:**
   ```typescript
   // index.ts
   export { MyFeaturePage as default } from './pages/MyFeaturePage'
   ```

4. **Register in feature-registry:**
   ```typescript
   // app/feature-registry.ts
   {
     id: 'my-feature',
     name: 'My Feature',
     icon: Star,
     path: '/my-feature',
     component: lazy(() => import('@/features/my-feature')),
     section: 'workspace',
     shortcut: 'g f',
   }
   ```

5. **Add route:**
   ```tsx
   // app/routes.tsx
   <Route path="my-feature" element={<MyFeaturePage />} />
   ```

## Type Definitions

Common types are in `shared/lib/types.ts`:

```typescript
// API response types
interface SystemStats { cpu, memory, disks, network, ... }
interface FileEntry { name, path, size, isDir, modTime, ... }
interface Task { id, command, status, pid, ... }

// Generic response wrapper
interface APIResponse {
  success?: boolean
  error?: string
}
```

## Building

```bash
cd web

# Development
bun run dev

# Type checking
bun run typecheck

# Linting
bun run lint

# Production build
bun run build
```
