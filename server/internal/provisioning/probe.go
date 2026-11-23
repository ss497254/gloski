package provisioning

import (
	"context"
	"strings"
)

// AgentState is what a read-only probe of a device's install directory found.
// A false value means "not there" or "not usable", never "the device is bad".
type AgentState struct {
	Dir      bool
	Binary   bool
	Runnable bool
	Running  bool
	PID      string
}

// Probe connects over SSH and inspects the agent install directory. It does
// not change anything on the device, so it is safe to run at any time.
func (i *Installer) Probe(ctx context.Context, target Target) (AgentState, error) {
	if err := target.Validate(); err != nil {
		return AgentState{}, err
	}
	output, err := i.runCommand(ctx, target, "sh -s", strings.NewReader(probeScript(target.RemoteBase)))
	if err != nil {
		return AgentState{}, err
	}
	return AgentState{
		Dir:      field(output, "GLOSKI_PROBE", "dir") == "yes",
		Binary:   field(output, "GLOSKI_PROBE", "binary") == "yes",
		Runnable: field(output, "GLOSKI_PROBE", "runnable") == "yes",
		Running:  field(output, "GLOSKI_PROBE", "running") == "yes",
		PID:      field(output, "GLOSKI_PROBE", "pid"),
	}, nil
}

// probeScript reports only facts. The version check is what catches a noexec
// mount, which "test -x" cannot see.
func probeScript(dir string) string {
	return `set -u
DIR=` + quote(dir) + `
if [ -d "$DIR" ]; then echo "GLOSKI_PROBE dir=yes"; else echo "GLOSKI_PROBE dir=no"; fi
if [ -x "$DIR/gloski-agent" ]; then echo "GLOSKI_PROBE binary=yes"; else echo "GLOSKI_PROBE binary=no"; fi
RUNNABLE=no
if [ -x "$DIR/gloski-agent" ]; then
  if command -v timeout >/dev/null 2>&1; then
    timeout 5 "$DIR/gloski-agent" -v >/dev/null 2>&1 && RUNNABLE=yes
  else
    "$DIR/gloski-agent" -v >/dev/null 2>&1 && RUNNABLE=yes
  fi
fi
echo "GLOSKI_PROBE runnable=$RUNNABLE"
pid=$(cat "$DIR/agent.pid" 2>/dev/null || true)
if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
  echo "GLOSKI_PROBE running=yes pid=$pid"
else
  echo "GLOSKI_PROBE running=no"
fi
`
}
