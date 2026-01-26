# Gloski JavaScript SDK

The `@gloski/sdk` package provides a TypeScript client for interacting with Gloski servers.

## Installation

The SDK is included as a workspace dependency. In the `web/` package, it's already configured:

```json
{
  "dependencies": {
    "@gloski/sdk": "*"
  }
}
```

## Quick Start

```typescript
import { GloskiClient } from '@gloski/sdk'

// Create a client
const client = new GloskiClient({
  url: 'https://server.example.com',
  apiKey: 'your-api-key',
})

// Get system stats
const stats = await client.system.getStats()
console.log(`CPU: ${stats.cpu.usage_percent}%`)

// List files
const { entries } = await client.files.list('/home')

// Connect to terminal
const term = client.terminal.connect()
term.on('data', (data) => console.log(data))
term.write('ls -la\n')
```

## Client Configuration

```typescript
interface GloskiClientConfig {
  /** Server URL (e.g., "https://server.example.com") */
  url: string

  /** API key for authentication */
  apiKey?: string

  /** JWT token for authentication */
  token?: string

  /** Request timeout in milliseconds (default: 30000) */
  timeout?: number

  /** API path prefix (default: "/api") */
  apiPrefix?: string

  /** Called when server returns 401 Unauthorized */
  onUnauthorized?: () => void

  /** Called when server is unreachable (network error) */
  onOffline?: () => void

  /** Called when server comes back online */
  onOnline?: () => void
}
```

### Custom API Prefix

If your server uses a different API path prefix, you can configure it:

```typescript
const client = new GloskiClient({
  url: 'https://server.example.com',
  apiKey: 'your-api-key',
  apiPrefix: '/v1', // Instead of default "/api"
})
```

## Resources

The client provides namespaced access to different API resources:

| Resource           | Description                               |
| ------------------ | ----------------------------------------- |
| `client.auth`      | Authentication status                     |
| `client.system`    | System stats and monitoring               |
| `client.files`     | File operations (includes pinned folders) |
| `client.jobs`      | Job/process management                    |
| `client.systemd`   | Systemd service management                |
| `client.search`    | File search                               |
| `client.terminal`  | Terminal WebSocket connections            |
| `client.packages`  | Package management (optional)             |
| `client.cron`      | Cron job management (optional)            |
| `client.downloads` | Download manager                          |

### System Resource

```typescript
// Get current stats
const stats = await client.system.getStats()

// Get stats history (last 5 minutes)
const history = await client.system.getHistory('5m')

// Get system info
const info = await client.system.getInfo()

// Get running processes
const processes = await client.system.getProcesses(100)
```

### Files Resource

```typescript
// List directory (also returns home_dir from pinned response)
const { path, entries } = await client.files.list('/home/user')

// Read file
const { content } = await client.files.read('/etc/hosts')

// Pinned folders (sub-resource of files)
const { folders, home_dir } = await client.files.pinned.list()
const pinned = await client.files.pinned.pin('/home/user/projects', 'Projects')
await client.files.pinned.unpin(pinned.id)

// Read large file with progress
const result = await client.files.readWithProgress('/var/log/large.log', (loaded, total) => {
  console.log(`Progress: ${loaded}/${total}`)
})

// Write file
await client.files.write('/tmp/test.txt', 'Hello World')

// Create directory
await client.files.mkdir('/tmp/new-folder')

// Delete file or directory
await client.files.delete('/tmp/test.txt')

// Rename/move
await client.files.rename('/tmp/old.txt', '/tmp/new.txt')

// Upload file
await client.files.upload('/tmp', file)

// Get download URL
const url = client.files.getDownloadUrl('/tmp/file.zip')
```

### Jobs Resource

```typescript
// List jobs
const jobs = await client.jobs.list()

// Start a job
const job = await client.jobs.start('bun run build', '/home/user/project')

// Get job details
const job = await client.jobs.get('job-id')

// Get job logs
const logs = await client.jobs.getLogs('job-id')

// Stop job
await client.jobs.stop('job-id')

// Delete job
await client.jobs.delete('job-id')
```

### Systemd Resource

```typescript
// List units
const units = await client.systemd.list(true) // true = user mode

// Start/stop/restart
await client.systemd.start('my-service.service')
await client.systemd.stop('my-service.service')
await client.systemd.restart('my-service.service')

// Enable/disable
await client.systemd.enable('my-service.service')
await client.systemd.disable('my-service.service')

// Get logs
const logs = await client.systemd.getLogs('my-service.service', true, 100)
```

### Search Resource

```typescript
// Search by filename
const results = await client.search.byName('/home', 'config')

// Search in file contents
const results = await client.search.byContent('/home', 'TODO')

// Full search options
const results = await client.search.search({
  path: '/home',
  query: 'config',
  content: true,
  limit: 50,
})
```

### Terminal Resource

The terminal resource provides WebSocket connections with auto-reconnect:

```typescript
// Connect to terminal
const term = client.terminal.connect({
  cwd: '/home/user',
  autoReconnect: true,
  maxReconnectAttempts: 5,
  reconnectDelay: 1000,
})

// Event handlers
term.on('open', () => {
  console.log('Connected')
  term.resize(120, 40)
})

term.on('data', (data) => {
  // Write to xterm.js or console
  console.log(data)
})

term.on('close', (event) => {
  console.log('Disconnected:', event.code)
})

term.on('error', (error) => {
  console.error('Error:', error)
})

term.on('reconnecting', (attempt) => {
  console.log(`Reconnecting (attempt ${attempt})...`)
})

term.on('reconnected', () => {
  console.log('Reconnected!')
})

// Send user input
term.write('ls -la\n')

// Resize terminal
term.resize(120, 40)

// Close connection
term.close()

// Check state
console.log(term.state) // 'connecting' | 'open' | 'reconnecting' | 'closed'
console.log(term.isOpen) // true/false
```

### Packages Resource

```typescript
// Get package manager info
const info = await client.packages.info()
console.log(info.manager) // 'apt', 'dnf', etc.

// List installed packages
const packages = await client.packages.listInstalled()

// Check for upgrades
const upgrades = await client.packages.checkUpgrades()

// Search packages
const results = await client.packages.search('nginx')

// Get package details
const pkg = await client.packages.get('nginx')
```

### Cron Resource

```typescript
// List cron jobs
const jobs = await client.cron.list()

// Add job
await client.cron.add({
  schedule: '0 * * * *',
  command: '/usr/local/bin/backup.sh',
})

// Remove job
await client.cron.remove('job-id')
```

## Error Handling

```typescript
import { GloskiClient, GloskiError } from '@gloski/sdk'

try {
  await client.files.read('/etc/shadow')
} catch (error) {
  if (error instanceof GloskiError) {
    console.log('Status:', error.status)
    console.log('Message:', error.message)

    if (error.isUnauthorized) {
      // Handle 401
    } else if (error.isForbidden) {
      // Handle 403
    } else if (error.isNotFound) {
      // Handle 404
    } else if (error.isNetworkError) {
      // Handle network error (status 0)
    } else if (error.isServerError) {
      // Handle 5xx errors
    }
  }
}
```

## Static Methods

```typescript
// Health check (no authentication required)
const health = await GloskiClient.checkHealth('https://server.example.com')
console.log(health.status) // 'ok'
console.log(health.version) // '1.0.0'
console.log(health.features) // { packages: true, cron: true, ... }
```

## TypeScript Types

All types are exported from the SDK:

```typescript
import type {
  // Client config
  GloskiClientConfig,

  // System types
  SystemStats,
  CPUStats,
  MemoryStats,
  ProcessInfo,
  SystemInfo,

  // File types
  FileEntry,
  ListResponse,
  ReadResponse,
  SearchResult,
  PinnedFolder,
  PinnedFoldersResponse,

  // Job types
  Job,
  JobStatus,

  // Systemd types
  SystemdUnit,
  SystemdAction,

  // Terminal types
  TerminalOptions,
  TerminalState,
  TerminalEvents,

  // Download types
  Download,
  DownloadStatus,
} from '@gloski/sdk'
```

## Architecture

```
@gloski/sdk
├── GloskiClient          # Main client class
├── GloskiError           # Error class
├── EventEmitter          # Base class for events
└── Resources
    ├── AuthResource
    ├── SystemResource
    ├── FilesResource
    │   └── PinnedSubResource  # files.pinned.*
    ├── JobsResource
    ├── SystemdResource
    ├── SearchResource
    ├── TerminalResource
    ├── PackagesResource
    ├── CronResource
    └── DownloadsResource
```

## Building the SDK

```bash
cd js-sdk

# Type check
bun run typecheck
```

The build outputs:

- `dist/index.js` - ESM bundle
- `dist/index.cjs` - CommonJS bundle
- `dist/index.d.ts` - TypeScript declarations
