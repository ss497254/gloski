import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export interface Note {
  id: string
  title: string
  content: string
  folder?: string
  tags: string[]
  pinned: boolean
  createdAt: string
  updatedAt: string
}

interface NotesState {
  notes: Note[]
  folders: string[]
  selectedNoteId: string | null
  addNote: (note: Omit<Note, 'id' | 'createdAt' | 'updatedAt' | 'pinned'>) => string
  updateNote: (id: string, data: Partial<Note>) => void
  deleteNote: (id: string) => void
  togglePin: (id: string) => void
  selectNote: (id: string | null) => void
  addFolder: (name: string) => void
  deleteFolder: (name: string) => void
}

// Mock data
const initialNotes: Note[] = [
  {
    id: '1',
    title: 'Server Setup Guide',
    content: `# Server Setup Guide

## Prerequisites
- Ubuntu 22.04 or later
- SSH access
- Root privileges

## Steps

### 1. Update system packages
\`\`\`bash
sudo apt update && sudo apt upgrade -y
\`\`\`

### 2. Install required dependencies
\`\`\`bash
sudo apt install -y curl wget git
\`\`\`

### 3. Configure firewall
\`\`\`bash
sudo ufw allow 22
sudo ufw allow 80
sudo ufw allow 443
sudo ufw enable
\`\`\`

## Notes
- Always backup before major changes
- Document any custom configurations
`,
    folder: 'Documentation',
    tags: ['server', 'setup', 'linux'],
    pinned: true,
    createdAt: '2024-01-15T10:00:00Z',
    updatedAt: '2024-01-18T14:30:00Z',
  },
  {
    id: '2',
    title: 'Meeting Notes - Jan 20',
    content: `# Team Standup - January 20

## Attendees
- Alice, Bob, Charlie

## Updates

### Alice
- Completed API refactoring
- Starting on frontend integration

### Bob
- Fixed critical bug in auth flow
- Reviewing PR #234

### Charlie
- Deployed staging environment
- Setting up monitoring

## Action Items
- [ ] Review deployment checklist
- [ ] Update documentation
- [ ] Schedule demo for Friday
`,
    folder: 'Meetings',
    tags: ['meeting', 'team'],
    pinned: false,
    createdAt: '2024-01-20T09:00:00Z',
    updatedAt: '2024-01-20T09:45:00Z',
  },
  {
    id: '3',
    title: 'API Endpoints Reference',
    content: `# API Endpoints

## Authentication
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/auth/login | User login |
| POST | /api/auth/logout | User logout |
| GET | /api/auth/status | Check auth status |

## Files
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/files | List files |
| POST | /api/files/upload | Upload file |
| DELETE | /api/files | Delete file |

## System
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/system/stats | Get system stats |
| GET | /api/system/processes | List processes |
`,
    folder: 'Documentation',
    tags: ['api', 'reference'],
    pinned: true,
    createdAt: '2024-01-10T15:00:00Z',
    updatedAt: '2024-01-19T11:20:00Z',
  },
  {
    id: '4',
    title: 'Quick Commands',
    content: `# Useful Commands

## Docker
\`\`\`bash
# List containers
docker ps -a

# Stop all containers
docker stop $(docker ps -q)

# Remove all stopped containers
docker container prune
\`\`\`

## System
\`\`\`bash
# Check disk space
df -h

# Check memory
free -h

# Find large files
find / -size +100M -type f 2>/dev/null
\`\`\`

## Git
\`\`\`bash
# Undo last commit (keep changes)
git reset --soft HEAD~1

# Clean untracked files
git clean -fd
\`\`\`
`,
    folder: 'Reference',
    tags: ['commands', 'cheatsheet'],
    pinned: false,
    createdAt: '2024-01-05T12:00:00Z',
    updatedAt: '2024-01-17T08:15:00Z',
  },
  {
    id: '5',
    title: 'Project Ideas',
    content: `# Project Ideas

## Short-term
1. **Dashboard improvements**
   - Add real-time updates
   - Better visualizations
   
2. **File manager enhancements**
   - Drag and drop upload
   - Bulk operations

## Long-term
1. **Multi-user support**
   - Role-based access
   - Audit logging

2. **Mobile app**
   - React Native
   - Push notifications

## Research
- WebSocket alternatives
- Container orchestration
- Edge computing
`,
    tags: ['ideas', 'planning'],
    pinned: false,
    createdAt: '2024-01-08T16:00:00Z',
    updatedAt: '2024-01-16T10:00:00Z',
  },
]

const initialFolders = ['Documentation', 'Meetings', 'Reference']

function generateId(): string {
  return Math.random().toString(36).substring(2, 10)
}

export const useNotesStore = create<NotesState>()(
  persist(
    (set) => ({
      notes: initialNotes,
      folders: initialFolders,
      selectedNoteId: null,

      addNote: (note) => {
        const id = generateId()
        const now = new Date().toISOString()
        set((state) => ({
          notes: [
            {
              ...note,
              id,
              pinned: false,
              createdAt: now,
              updatedAt: now,
            },
            ...state.notes,
          ],
          selectedNoteId: id,
        }))
        return id
      },

      updateNote: (id, data) =>
        set((state) => ({
          notes: state.notes.map((n) =>
            n.id === id
              ? { ...n, ...data, updatedAt: new Date().toISOString() }
              : n
          ),
        })),

      deleteNote: (id) =>
        set((state) => ({
          notes: state.notes.filter((n) => n.id !== id),
          selectedNoteId:
            state.selectedNoteId === id ? null : state.selectedNoteId,
        })),

      togglePin: (id) =>
        set((state) => ({
          notes: state.notes.map((n) =>
            n.id === id ? { ...n, pinned: !n.pinned } : n
          ),
        })),

      selectNote: (id) => set({ selectedNoteId: id }),

      addFolder: (name) =>
        set((state) => ({
          folders: state.folders.includes(name)
            ? state.folders
            : [...state.folders, name],
        })),

      deleteFolder: (name) =>
        set((state) => ({
          folders: state.folders.filter((f) => f !== name),
          notes: state.notes.map((n) =>
            n.folder === name ? { ...n, folder: undefined } : n
          ),
        })),
    }),
    {
      name: 'gloski-notes',
      partialize: (state) => ({
        notes: state.notes,
        folders: state.folders,
      }),
    }
  )
)
