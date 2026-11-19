// Package builds embeds the agent binaries the master hands to devices.
// "make build-agents" writes gloski-agent-linux-{amd64,arm64,arm} into this
// directory before the master is compiled; they are not committed. Only the
// master imports this package, so agents never embed other agents.
package builds

import "embed"

// FS holds whatever agent builds were present at compile time.
//
//go:embed *
var FS embed.FS
