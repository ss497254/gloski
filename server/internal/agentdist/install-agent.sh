#!/bin/sh
# Gloski agent installer. The master renders this and runs it over SSH. It is
# safe to run repeatedly: it replaces the agent only when the binary or its
# settings changed, and starts it only when it is not running.
set -eu

DIR=@@DIR@@
FORCE=@@FORCE@@
SHA_amd64=@@SHA_AMD64@@
SHA_arm64=@@SHA_ARM64@@
SHA_arm=@@SHA_ARM@@

fail() {
  code=$1
  shift
  echo "$*" >&2
  exit "$code"
}

[ "$(uname -s)" = "Linux" ] || fail 40 "only Linux is supported (this device runs $(uname -s))"
case "$(uname -m)" in
  x86_64 | amd64) ARCH=amd64 ;;
  aarch64 | arm64) ARCH=arm64 ;;
  armv7* | armv6* | armhf | arm) ARCH=arm ;;
  *) fail 40 "unsupported CPU architecture $(uname -m)" ;;
esac
eval "WANT=\$SHA_$ARCH"
[ -n "$WANT" ] || fail 41 "the master has no agent build for linux/$ARCH (run 'make build' on the master)"

mkdir -p "$DIR" || fail 43 "cannot create $DIR on the device"
cd "$DIR"
BIN="$DIR/gloski-agent"

sha_of() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | cut -d' ' -f1
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | cut -d' ' -f1
  else
    # No checksum tool: fall back to the hash recorded at the last install.
    cat "$1.sha256" 2>/dev/null || echo unknown
  fi
}

CHANGED=$FORCE
CURRENT=none
[ -f "$BIN" ] && CURRENT=$(sha_of "$BIN")
if [ "$CURRENT" != "$WANT" ]; then
  if [ ! -f "$BIN.upload" ]; then
    # Ask the master to copy the right build here, then run this again.
    echo "GLOSKI_NEED_UPLOAD arch=$ARCH"
    exit 42
  fi
  chmod 755 "$BIN.upload"
  if ! "$BIN.upload" -v >/dev/null 2>&1; then
    rm -f "$BIN.upload"
    fail 43 "the device cannot run programs from $DIR (it is mounted noexec, or the CPU type was detected wrongly)"
  fi
  CHANGED=1
fi

umask 077
cat > "$DIR/agent.json.new" <<'GLOSKI_CONFIG_END'
@@CONFIG@@
GLOSKI_CONFIG_END
umask 022
if ! cmp -s "$DIR/agent.json.new" "$DIR/agent.json" 2>/dev/null; then
  CHANGED=1
fi

running_pid() {
  pid=$(cat "$1" 2>/dev/null || true)
  if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
    echo "$pid"
  fi
}

stop_pid_file() {
  pid=$(running_pid "$1")
  if [ -n "$pid" ]; then
    kill "$pid" 2>/dev/null || true
    i=0
    while kill -0 "$pid" 2>/dev/null && [ "$i" -lt 100 ]; do
      sleep 0.1 2>/dev/null || sleep 1
      i=$((i + 1))
    done
    kill -9 "$pid" 2>/dev/null || true
  fi
  rm -f "$1"
}

# Remove an agent installed by an older Gloski, which lived directly in /tmp.
if [ -f /tmp/gloski-agent.pid ] || [ -f /tmp/gloski-agent.env ]; then
  stop_pid_file /tmp/gloski-agent.pid
  rm -rf /tmp/gloski-agent /tmp/gloski-agent.env /tmp/gloski-agent.log /tmp/gloski-agent-data /tmp/gloski-agent.upload
fi

if [ "$CHANGED" = 1 ]; then
  stop_pid_file "$DIR/agent.pid"
  if [ -f "$BIN.upload" ]; then
    mv "$BIN.upload" "$BIN"
    echo "$WANT" > "$BIN.sha256"
  fi
fi
mv "$DIR/agent.json.new" "$DIR/agent.json"

STARTED=no
PID=$(running_pid "$DIR/agent.pid")
if [ -z "$PID" ]; then
  # Keep one previous log so the file cannot grow without limit.
  if [ -f "$DIR/agent.log" ] && [ "$(wc -c < "$DIR/agent.log")" -gt 5000000 ]; then
    mv "$DIR/agent.log" "$DIR/agent.log.1"
  fi
  GLOSKI_AGENT_INSTALL_DIR="$DIR" GLOSKI_DATA_DIR="$DIR/data" \
    nohup "$BIN" -c "$DIR/agent.json" >> "$DIR/agent.log" 2>&1 < /dev/null &
  PID=$!
  echo "$PID" > "$DIR/agent.pid"
  STARTED=yes
  sleep 1
  if ! kill -0 "$PID" 2>/dev/null; then
    rm -f "$DIR/agent.pid"
    echo "the agent started and exited at once. Last lines of its log:" >&2
    tail -n 5 "$DIR/agent.log" >&2 || true
    exit 45
  fi
fi

echo "GLOSKI_INSTALLED pid=$PID arch=$ARCH started=$STARTED"
