# Gloski JavaScript Client

The Gloski client is a TypeScript library for interacting with Gloski servers and agents. It lives inside the web app at `web/src/shared/api` and is imported through the `@/shared/api` path alias.

## Installation

There is no separate package to install. The client is part of the `web/`
workspace and shares its dependencies:

```typescript
import { GloskiClient } from '@/shared/api'
```

## Quick Start

```typescript
import { GloskiClient } from '@/shared/api'

// Create a client
const client = new GloskiClient({
  url: 'https://server.example.com',
  apiKey: 'your-api-key',
})

// All resource methods return Result<T> = { data, error }
const { data: stats, error } = await client.system.getStats()
if (error) {
  console.error(error.message)
} else {
  console.log(`CPU: ${stats.cpu.usage_percent}%`)
}

// List files
const { data: listing } = await client.files.list('/home')

// Connect to terminal (WebSocket — not wrapped in Result)
const term = client.terminal.connect()
term.on('data', (data) => console.log(data))
term.write('ls -la\n')

// Dispose when done
client.dispose()
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

### Cleanup

Call `dispose()` when the client is no longer needed to clear pending request references:

```typescript
client.dispose()
```

## Result Type

All resource methods return `Promise<Result<T>>` instead of throwing errors:

```typescript
type Result<T> =
  | { data: T; error: null }
  | { data: null; error: GloskiError }
```

This allows safe destructuring without try/catch:

```typescript
const { data, error } = await client.system.getStats()
if (error) {
  console.error(error.message)
  return
}
// data is typed as SystemStats
console.log(data.cpu.usage_percent)
```

## Resources

The client provides namespaced access to different API resources:

| Resource           | Description                               |
| ------------------ | ----------------------------------------- |
| `client.auth`      | Authentication status                     |
| `client.system`    | System stats and monitoring               |
| `client.files`     | File operations (includes pinned folders) |
| `client.jobs`      | Job/process management                    |
| `client.search`    | File search                               |
| `client.terminal`  | Terminal WebSocket connections            |
| `client.packages`  | Package management (optional)             |
| `client.cron`      | Cron job management (optional)            |
| `client.downloads` | Download manager                          |

### System Resource

```typescript
// Get current stats
const { data: stats } = await client.system.getStats()

// Get stats history (last 5 minutes)
const { data: history } = await client.system.getHistory('5m')

// Get system info
const { data: info } = await client.system.getInfo()

// Get running processes
const { data: processes } = await client.system.getProcesses(100)

// Real-time stats via WebSocket
const statsConn = client.system.connectStats()
statsConn.on('stats', (stats) => {
  console.log('CPU:', stats.cpu.usage_percent)
})
statsConn.on('error', (err) => console.error(err))
// Later: statsConn.close()
```

### Files Resource

```typescript
// List directory
const { data: listing } = await client.files.list('/home/user')

// Read file
const { data: file } = await client.files.read('/etc/hosts')

// Pinned folders (sub-resource of files)
const { data: pinned } = await client.files.pinned.list()
const { data: folder } = await client.files.pinned.pin('/home/user/projects', 'Projects')
await client.files.pinned.unpin(folder.id)

// Read large file with progress
const { data: result } = await client.files.readWithProgress('/var/log/large.log', (loaded, total) => {
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

// Get download URL (synchronous, returns string)
const url = client.files.getDownloadUrl('/tmp/file.zip')
```

### Jobs Resource

```typescript
// List jobs
const { data: jobs } = await client.jobs.list()

// Start a job
const { data: job } = await client.jobs.start('bun run build', '/home/user/project')

// Get job details
const { data: job } = await client.jobs.get('job-id')

// Get job logs
const { data: logs } = await client.jobs.getLogs('job-id')

// Stop job
await client.jobs.stop('job-id')

// Delete job
await client.jobs.delete('job-id')
```

### Search Resource

```typescript
// Search by filename
const { data: results } = await client.search.byName('/home', 'config')

// Search in file contents
const { data: results } = await client.search.byContent('/home', 'TODO')

// Full search options
const { data: results } = await client.search.search({
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
const { data: info } = await client.packages.info()
console.log(info.manager) // 'apt', 'dnf', etc.

// List installed packages
const { data: packages } = await client.packages.listInstalled()

// Check for upgrades
const { data: upgrades } = await client.packages.checkUpgrades()

// Search packages
const { data: results } = await client.packages.search('nginx')

// Get package details
const { data: pkg } = await client.packages.get('nginx')
```

### Cron Resource

```typescript
// List cron jobs
const { data: jobs } = await client.cron.list()

// Add job
await client.cron.add({
  schedule: '0 * * * *',
  command: '/usr/local/bin/backup.sh',
})

// Remove job (by schedule + command)
await client.cron.remove('0 * * * *', '/usr/local/bin/backup.sh')
```

### Downloads Resource

```typescript
// List all downloads
const { data: downloads } = await client.downloads.list()

// Add a new download
const { data: download } = await client.downloads.add(
  'https://example.com/file.tar.gz',
  '/home/user/downloads'
)

// Get download by ID
const { data: dl } = await client.downloads.get('download-id')

// Control downloads
await client.downloads.pause('download-id')
await client.downloads.resume('download-id')
await client.downloads.cancel('download-id')
await client.downloads.retry('download-id')

// Delete download (optionally delete file too)
await client.downloads.delete('download-id', true)

// Get authenticated download URL (synchronous)
const url = client.downloads.getDownloadUrl('download-id')

// Share links
const { data: link } = await client.downloads.createShareLink('download-id', {
  expires_in: 3600, // seconds, or omit for no expiry
})
await client.downloads.revokeShareLink('download-id', link.token)
```

## Error Handling

All resource methods return `Result<T>` — errors are returned, not thrown:

```typescript
const { data, error } = await client.files.read('/etc/shadow')

if (error) {
  // error is a GloskiError instance
  console.log('Status:', error.status)   // HTTP status (0 for network errors)
  console.log('Message:', error.message)

  // Convenience getters
  if (error.isUnauthorized) { /* 401 */ }
  if (error.isForbidden) { /* 403 */ }
  if (error.isNotFound) { /* 404 */ }
  if (error.isNetworkError) { /* status 0, server unreachable */ }
  if (error.isServerError) { /* 5xx */ }
  if (error.isClientError) { /* 4xx */ }
  return
}

// data is typed and non-null here
console.log(data.content)
```

## Static Methods

```typescript
// Health check (no authentication required)
const health = await GloskiClient.checkHealth('https://server.example.com')
console.log(health.status) // 'ok'
```

## TypeScript Types

All types are exported from the SDK:

```typescript
import type {
  // Client config
  GloskiClientConfig,
  Result,
  ApiResponseEnvelope,

  // System types
  SystemStats,
  CPUStats,
  MemoryStats,
  SwapStats,
  DiskStats,
  NetworkStats,
  LoadAvg,
  ProcessInfo,
  SystemInfo,
  ServerHealthReport,
  StatsSample,
  StatsHistoryResponse,

  // File types
  FileEntry,
  DirectoryListing,
  FileContent,
  FileSearchMatch,
  FileSearchOptions,
  FileSearchResponse,
  PinnedFolder,
  PinnedFoldersResponse,
  ChunkedUploadInit,
  ChunkedUploadInfo,

  // Job types
  Job,
  JobStatus,

  // Terminal types
  TerminalOptions,
  TerminalEvents,

  // WebSocket types
  WebSocketConnectionOptions,
  WebSocketConnectionState,
  StatsConnectionEvents,

  // Download types
  Download,
  DownloadStatus,
  NewDownloadRequest,
  ShareLink,
  CreateShareOptions,

  // Package types
  SystemPackage,
  PackageManagerInfo,
  InstalledPackagesResponse,
  UpgradeInfo,

  // Cron types
  CronJob,
  NewCronJob,
  CronScope,
} from '@/shared/api'
```

## Architecture

```
web/src/shared/api
├── GloskiClient          # Main client class
├── GloskiError           # Error class
├── safe()                # Wraps promises into Result<T>
├── EventEmitter          # Base class for events
└── Resources
    ├── AuthResource
    ├── SystemResource
    ├── FilesResource
    │   └── PinnedFoldersResource  # files.pinned.*
    ├── JobsResource
    ├── SearchResource
    ├── TerminalResource
    ├── PackagesResource
    ├── CronResource
    └── DownloadsResource
```

## Type Checking and Tests

The client is source-only and has no build step. It is consumed directly by the
web app through the `@/shared/api` alias.

```bash
cd web

# Type check the frontend
bun run type-check

# Run the client tests
bun test src/shared/api
```
