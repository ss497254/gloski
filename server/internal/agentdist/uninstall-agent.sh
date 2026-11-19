#!/bin/sh
# Gloski agent uninstaller. MODE=remove deletes everything; MODE=stop stops the
# agent and its self-heal but keeps the files so it can be started again.
set -u

DIR=@@DIR@@
MODE=@@MODE@@
[ "${1:-}" = "--stop" ] && MODE=stop
[ "${1:-}" = "--remove" ] && MODE=remove

if command -v crontab >/dev/null 2>&1; then
  if crontab -l 2>/dev/null | grep -q "# gloski-agent $DIR\$"; then
    crontab -l 2>/dev/null | grep -v "# gloski-agent $DIR\$" | crontab - 2>/dev/null
  fi
fi

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

if [ "$MODE" = stop ]; then
  # Tells the supervisor and cron-less restarts that this stop is deliberate.
  [ -d "$DIR" ] && : > "$DIR/stopped"
fi
stop_pid_file "$DIR/agent.pid"
stop_pid_file /tmp/gloski-agent.pid

if [ "$MODE" = remove ]; then
  rm -rf /tmp/gloski-agent /tmp/gloski-agent.env /tmp/gloski-agent.log /tmp/gloski-agent-data /tmp/gloski-agent.upload
  case "$DIR" in
    "" | / | /tmp | /home | /root | /usr | /etc | /var) echo "refusing to remove $DIR" >&2 ;;
    *) rm -rf "$DIR" ;;
  esac
  echo "GLOSKI_UNINSTALLED state=$STATE dir=$DIR"
else
  echo "GLOSKI_STOPPED state=$STATE dir=$DIR"
fi
