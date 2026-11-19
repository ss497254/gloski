#!/bin/sh
# Gloski agent installer. Rendered by the master; safe to run repeatedly.
set -eu

DIR=@@DIR@@
MASTER_HTTP=@@MASTER_HTTP@@
MASTER_URL=@@MASTER_URL@@
JOIN_TOKEN=@@JOIN_TOKEN@@
AGENT_NAME=@@AGENT_NAME@@
DEPLOY_ID=@@DEPLOY_ID@@
REINSTALL_URL=@@REINSTALL_URL@@
SHA_amd64=@@SHA_AMD64@@
SHA_arm64=@@SHA_ARM64@@
SHA_arm=@@SHA_ARM@@

fail() {
  code=$1
  shift
  echo "gloski install failed: $*" >&2
  exit "$code"
}

[ "$(uname -s)" = "Linux" ] || fail 40 "only Linux is supported (this is $(uname -s))"
case "$(uname -m)" in
  x86_64 | amd64) ARCH=amd64 ;;
  aarch64 | arm64) ARCH=arm64 ;;
  armv7* | armv6* | armhf | arm) ARCH=arm ;;
  *) fail 40 "unsupported CPU architecture $(uname -m)" ;;
esac
eval "WANT=\$SHA_$ARCH"
[ -n "$WANT" ] || fail 41 "the master has no agent build for linux/$ARCH (run 'make build' on the master)"

mkdir -p "$DIR" || fail 43 "cannot create $DIR"
cd "$DIR"
BIN="$DIR/gloski-agent"

sha_of() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | cut -d' ' -f1
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | cut -d' ' -f1
  else
    echo unknown
  fi
}

fetch() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL --connect-timeout 10 -o "$2" "$1"
  elif command -v wget >/dev/null 2>&1; then
    wget -q -T 10 -O "$2" "$1"
  else
    return 42
  fi
}

CHANGED=0
CURRENT=none
[ -f "$BIN" ] && CURRENT=$(sha_of "$BIN")
if [ "$CURRENT" != "$WANT" ]; then
  if [ -f "$BIN.upload" ]; then
    # Placed here by the master over SSH because this device cannot download.
    mv "$BIN.upload" "$BIN.new"
  else
    rc=0
    fetch "$MASTER_HTTP/agent/linux/$ARCH" "$BIN.new" || rc=$?
    if [ "$rc" = 42 ]; then
      echo "GLOSKI_NEED_UPLOAD arch=$ARCH"
      fail 42 "neither curl nor wget is installed"
    fi
    [ "$rc" = 0 ] || fail 44 "could not download the agent from $MASTER_HTTP (is the master reachable from this device?)"
  fi
  GOT=$(sha_of "$BIN.new")
  if [ "$GOT" != unknown ] && [ "$GOT" != "$WANT" ]; then
    rm -f "$BIN.new"
    fail 44 "downloaded agent is corrupt (checksum mismatch)"
  fi
  chmod 755 "$BIN.new"
  if ! "$BIN.new" -v >/dev/null 2>&1; then
    rm -f "$BIN.new"
    fail 43 "cannot run programs from $DIR (mounted noexec, or wrong architecture). Install into another directory by adding dir=/some/path to the install URL"
  fi
  CHANGED=1
fi

umask 077
cat > "$DIR/agent.env.new" <<GLOSKI_ENV
GLOSKI_AGENT_MASTER_URL=$MASTER_URL
GLOSKI_AGENT_JOIN_TOKEN=$JOIN_TOKEN
GLOSKI_AGENT_NAME=$AGENT_NAME
GLOSKI_AGENT_DEPLOY_ID=$DEPLOY_ID
GLOSKI_DATA_DIR=$DIR/data
GLOSKI_ENV
if ! cmp -s "$DIR/agent.env.new" "$DIR/agent.env" 2>/dev/null; then
  CHANGED=1
fi
umask 022

stop_pid_file() {
  [ -f "$1" ] || return 0
  pid=$(cat "$1" 2>/dev/null || true)
  if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
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
  [ -f "$BIN.new" ] && mv "$BIN.new" "$BIN"
  mv "$DIR/agent.env.new" "$DIR/agent.env"
else
  rm -f "$DIR/agent.env.new"
fi
rm -f "$DIR/stopped"

"$BIN" start -dir "$DIR" || fail 45 "the agent did not start; see $DIR/agent.log"

# Self-heal: every 5 minutes start the agent if it is not running, and if its
# files are gone (a reboot wiped them) fetch this installer again.
SELFHEAL=none
if command -v crontab >/dev/null 2>&1; then
  LINE="*/5 * * * * $BIN start -dir $DIR >/dev/null 2>&1"
  if [ -n "$REINSTALL_URL" ]; then
    if command -v curl >/dev/null 2>&1; then
      LINE="$LINE || curl -fsSL --connect-timeout 10 '$REINSTALL_URL' | sh >/dev/null 2>&1"
    elif command -v wget >/dev/null 2>&1; then
      LINE="$LINE || wget -q -T 10 -O - '$REINSTALL_URL' | sh >/dev/null 2>&1"
    fi
  fi
  LINE="$LINE # gloski-agent $DIR"
  if { crontab -l 2>/dev/null | grep -v "# gloski-agent $DIR\$" || true; echo "$LINE"; } | crontab - 2>/dev/null; then
    SELFHEAL=cron
  fi
fi
if [ "$SELFHEAL" = none ]; then
  echo "warning: no usable crontab on this device; the agent restarts after a crash but not after a reboot" >&2
fi

PID=$(cat "$DIR/agent.pid" 2>/dev/null || echo 0)
echo "GLOSKI_INSTALLED pid=$PID arch=$ARCH selfheal=$SELFHEAL dir=$DIR"
