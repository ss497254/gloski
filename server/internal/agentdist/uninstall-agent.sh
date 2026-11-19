#!/bin/sh
# Gloski agent uninstaller. MODE=remove deletes everything the installer put
# on the device; MODE=stop only stops the agent.
set -u

DIR=@@DIR@@
MODE=@@MODE@@

STATE=not-running
stop_pid_file() {
  [ -f "$1" ] || return 0
  pid=$(cat "$1" 2>/dev/null)
  if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
    kill "$pid" 2>/dev/null
    i=0
    while kill -0 "$pid" 2>/dev/null && [ "$i" -lt 100 ]; do
      sleep 0.1 2>/dev/null || sleep 1
      i=$((i + 1))
    done
    kill -9 "$pid" 2>/dev/null
    STATE=stopped
  fi
  rm -f "$1"
}

stop_pid_file "$DIR/agent.pid"
stop_pid_file /tmp/gloski-agent.pid

if [ "$MODE" = remove ]; then
  rm -rf /tmp/gloski-agent /tmp/gloski-agent.env /tmp/gloski-agent.log /tmp/gloski-agent-data /tmp/gloski-agent.upload
  case "$DIR" in
    "" | / | /tmp | /home | /root | /usr | /etc | /var) echo "refusing to remove $DIR" >&2 ;;
    *) rm -rf "$DIR" ;;
  esac
  echo "GLOSKI_UNINSTALLED state=$STATE"
else
  echo "GLOSKI_STOPPED state=$STATE"
fi
