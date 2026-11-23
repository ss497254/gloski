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

# Every failure names the phase it happened in. The master records the message
# on the deployment, and the diagnose view shows it as the last recorded error.
PHASE=platform
fail() {
  code=$1
  shift
  echo "$PHASE: $*" >&2
  exit "$code"
}

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

start_agent() {
  STARTED=no
  PID=$(running_pid "$DIR/agent.pid")
  if [ -n "$PID" ]; then
    return 0
  fi
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
    echo "start: the agent started and exited at once. Last lines of its log:" >&2
    tail -n 5 "$DIR/agent.log" >&2 || true
    return 1
  fi
  return 0
}

# A failed replacement restores the previous binary and config, and restarts
# the agent if it was running before. A device is never left offline because
# an install went wrong.
STAGE=
BACKUP=
CONFIG_BACKUP=
SHA_BACKUP=
HAD_BINARY=no
HAD_CONFIG=no
HAD_SHA=no
WAS_RUNNING=no
rollback() {
  status=$?
  trap - EXIT HUP INT TERM
  if [ "$status" -ne 0 ]; then
    restored=no
    # Restore each file that existed before, and remove each one that did not,
    # so a failed first install leaves no half-installed agent behind.
    if [ "$HAD_BINARY" = yes ] && [ -f "$BACKUP" ]; then
      mv -f "$BACKUP" "$BIN" 2>/dev/null || true
      restored=yes
    elif [ "$HAD_BINARY" = no ]; then
      rm -f "$BIN" 2>/dev/null || true
    fi
    if [ "$HAD_CONFIG" = yes ] && [ -f "$CONFIG_BACKUP" ]; then
      mv -f "$CONFIG_BACKUP" "$DIR/agent.json" 2>/dev/null || true
      restored=yes
    elif [ "$HAD_CONFIG" = no ]; then
      rm -f "$DIR/agent.json" 2>/dev/null || true
    fi
    if [ "$HAD_SHA" = yes ] && [ -f "$SHA_BACKUP" ]; then
      mv -f "$SHA_BACKUP" "$BIN.sha256" 2>/dev/null || true
      restored=yes
    elif [ "$HAD_SHA" = no ]; then
      rm -f "$BIN.sha256" 2>/dev/null || true
    fi
    rm -f "$DIR/agent.json.new" 2>/dev/null || true
    if [ "$restored" = yes ] && [ "$WAS_RUNNING" = yes ]; then
      if start_agent; then
        echo "replace: install failed; restored and restarted the previous agent" >&2
      else
        echo "replace: install failed; restored the previous files but could not restart the agent" >&2
      fi
    elif [ "$restored" = yes ]; then
      echo "replace: install failed; restored the previous agent files" >&2
    else
      echo "replace: install failed; removed the partial install" >&2
    fi
  fi
  if [ -n "$STAGE" ]; then rm -f "$STAGE" 2>/dev/null || true; fi
  if [ -n "$BACKUP" ]; then rm -f "$BACKUP" 2>/dev/null || true; fi
  if [ -n "$CONFIG_BACKUP" ]; then rm -f "$CONFIG_BACKUP" 2>/dev/null || true; fi
  if [ -n "$SHA_BACKUP" ]; then rm -f "$SHA_BACKUP" 2>/dev/null || true; fi
}
trap rollback EXIT
# Signals must exit nonzero so the EXIT trap performs the rollback rather than
# just deleting the backup files.
trap 'exit 1' HUP INT TERM

[ "$(uname -s)" = "Linux" ] || fail 40 "only Linux is supported (this device runs $(uname -s))"
case "$(uname -m)" in
  x86_64 | amd64) ARCH=amd64 ;;
  aarch64 | arm64) ARCH=arm64 ;;
  armv7* | armv6* | armhf | arm) ARCH=arm ;;
  *) fail 40 "unsupported CPU architecture $(uname -m)" ;;
esac
eval "WANT=\$SHA_$ARCH"
[ -n "$WANT" ] || fail 41 "the master has no agent build for linux/$ARCH (run 'make build' on the master)"

PHASE=prepare
mkdir -p "$DIR" || fail 43 "cannot create $DIR on the device"
cd "$DIR"
BIN="$DIR/gloski-agent"
if [ -f "$BIN" ]; then HAD_BINARY=yes; fi
if [ -f "$DIR/agent.json" ]; then HAD_CONFIG=yes; fi
if [ -f "$BIN.sha256" ]; then HAD_SHA=yes; fi
if [ -n "$(running_pid "$DIR/agent.pid")" ]; then
  WAS_RUNNING=yes
fi

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

PHASE=binary
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

PHASE=config
# The master has already merged its fields into this file, so operator settings
# such as a custom port survive a reinstall.
umask 077
cat > "$DIR/agent.json.new" <<'GLOSKI_CONFIG_END'
@@CONFIG@@
GLOSKI_CONFIG_END
umask 022
if ! cmp -s "$DIR/agent.json.new" "$DIR/agent.json" 2>/dev/null; then
  CHANGED=1
  if [ -f "$DIR/agent.json" ]; then
    CONFIG_BACKUP="$DIR/agent.json.bak"
    cp "$DIR/agent.json" "$CONFIG_BACKUP" || fail 43 "cannot back up the current agent config"
  fi
fi

# Remove an agent installed by an older Gloski, which lived directly in /tmp.
if [ -f /tmp/gloski-agent.pid ] || [ -f /tmp/gloski-agent.env ]; then
  stop_pid_file /tmp/gloski-agent.pid
  rm -rf /tmp/gloski-agent /tmp/gloski-agent.env /tmp/gloski-agent.log /tmp/gloski-agent-data /tmp/gloski-agent.upload
fi

PHASE=replace
if [ "$CHANGED" = 1 ]; then
  stop_pid_file "$DIR/agent.pid"
  if [ -f "$BIN.upload" ]; then
    STAGE=$(mktemp "$DIR/.gloski-agent.XXXXXX") || fail 43 "cannot stage the agent binary in $DIR"
    cp "$BIN.upload" "$STAGE" || fail 43 "cannot stage the agent binary in $DIR"
    if [ ! -s "$STAGE" ]; then
      rm -f "$BIN.upload"
      fail 43 "the uploaded agent binary is empty"
    fi
    GOT=$(sha_of "$STAGE")
    if [ "$GOT" != unknown ] && [ "$GOT" != "$WANT" ]; then
      # A rejected upload must not be reused on the next attempt.
      rm -f "$BIN.upload"
      fail 43 "the uploaded agent binary failed its checksum (got $GOT, want $WANT)"
    fi
    if [ "$HAD_BINARY" = yes ]; then
      BACKUP="$DIR/gloski-agent.bak"
      cp "$BIN" "$BACKUP" || fail 43 "cannot back up the current agent binary"
    fi
    if [ "$HAD_SHA" = yes ]; then
      SHA_BACKUP="$DIR/gloski-agent.sha256.bak"
      cp "$BIN.sha256" "$SHA_BACKUP" || fail 43 "cannot back up the current checksum"
    fi
    mv "$STAGE" "$BIN"
    STAGE=
    rm -f "$BIN.upload"
    chmod 755 "$BIN" || fail 43 "cannot make the agent binary executable"
    echo "$WANT" > "$BIN.sha256"
  fi
  mv "$DIR/agent.json.new" "$DIR/agent.json"
else
  rm -f "$DIR/agent.json.new"
fi

PHASE=start
start_agent || exit 45

echo "GLOSKI_INSTALLED pid=$PID arch=$ARCH started=$STARTED"
