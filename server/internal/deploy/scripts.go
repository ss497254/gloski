// Remote shell scripts used to install and control the agent binary under /tmp.

package deploy

import (
	"fmt"
	"os"
	"path"
	"strings"
)

type remotePaths struct {
	binary  string
	dataDir string
	envFile string
	pidFile string
	logFile string
	upload  string
}

func targetPaths(target Target) remotePaths {
	base := target.RemoteBase
	if base == "" {
		base = defaultRemoteDir
	}
	binary := path.Join(base, "gloski-agent")
	return remotePaths{
		binary:  binary,
		dataDir: path.Join(base, "gloski-agent-data"),
		envFile: path.Join(base, "gloski-agent.env"),
		pidFile: path.Join(base, "gloski-agent.pid"),
		logFile: path.Join(base, "gloski-agent.log"),
		upload:  binary + ".upload",
	}
}

func startScript(target Target) string {
	paths := targetPaths(target)
	var env strings.Builder
	writeEnv := func(key, value string) {
		env.WriteString(key)
		env.WriteString("=")
		env.WriteString(quote(value))
		env.WriteString("\n")
	}
	writeEnv("GLOSKI_MODE", "agent")
	writeEnv("GLOSKI_AGENT_MASTER_URL", target.MasterURL)
	writeEnv("GLOSKI_AGENT_JOIN_TOKEN", target.JoinToken)
	writeEnv("GLOSKI_AGENT_NAME", target.Name)
	writeEnv("GLOSKI_AGENT_DEPLOY_ID", target.DeployID)
	writeEnv("GLOSKI_DATA_DIR", paths.dataDir)

	return fmt.Sprintf(`set -eu
mkdir -p %s
umask 077
cat > %s <<'GLOSKI_ENV'
%sGLOSKI_ENV
chmod 600 %s
if [ -f %s ]; then
  pid=$(cat %s 2>/dev/null || true)
  if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
    echo "$pid"
    exit 0
  fi
  rm -f %s
fi
nohup sh -c "set -a; . %s; exec %s" >> %s 2>&1 < /dev/null &
pid=$!
echo "$pid" > %s
echo "$pid"
`,
		quote(paths.dataDir),
		quote(paths.envFile),
		env.String(),
		quote(paths.envFile),
		quote(paths.pidFile),
		quote(paths.pidFile),
		quote(paths.pidFile),
		quote(paths.envFile),
		quote(paths.binary),
		quote(paths.logFile),
		quote(paths.pidFile),
	)
}

func stopScript(target Target) string {
	paths := targetPaths(target)
	return fmt.Sprintf(`set +e
pid=""
if [ -f %s ]; then
  pid=$(cat %s 2>/dev/null)
fi
if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
  kill "$pid" 2>/dev/null
  i=0
  while kill -0 "$pid" 2>/dev/null && [ "$i" -lt 5 ]; do
    sleep 1
    i=$((i+1))
  done
  kill -9 "$pid" 2>/dev/null
  echo stopped
else
  echo not-running
fi
rm -f %s
`,
		quote(paths.pidFile),
		quote(paths.pidFile),
		quote(paths.pidFile),
	)
}

func quote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func lastLine(value string) string {
	lines := strings.Split(strings.TrimSpace(value), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			return line
		}
	}
	return ""
}

func currentUser() string {
	if user := strings.TrimSpace(os.Getenv("USER")); user != "" {
		return user
	}
	return strings.TrimSpace(os.Getenv("USERNAME"))
}
