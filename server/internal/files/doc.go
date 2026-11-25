// Package files implements the filesystem operations the agent exposes to the
// master: path validation, listing, reading, writing, deleting, renaming, and
// stat. Every operation resolves and validates its path first; paths outside
// the configured allowed roots are refused.
//
// The package is shared by the agent's gRPC handlers and the master's local
// helpers. It does not speak gRPC itself.
package files
