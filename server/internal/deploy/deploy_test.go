package deploy

import (
	"strings"
	"testing"
)

func validTarget() Target {
	return Target{
		Name:      "worker",
		Host:      "100.64.0.2",
		User:      "root",
		SSHPort:   22,
		AgentPort: 18081,
		MasterURL: "https://tower.tailnet.ts.net",
		JoinToken: "join-token",
		DeployID:  "deploy-id",
	}
}

func TestTargetValidate(t *testing.T) {
	target := validTarget()
	if err := target.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if target.RemoteBase != "/tmp" {
		t.Fatalf("RemoteBase = %q, want /tmp", target.RemoteBase)
	}

	bad := validTarget()
	bad.Host = "-oProxyCommand=bad"
	if err := bad.Validate(); err == nil {
		t.Fatal("expected invalid host to fail")
	}

	bad = validTarget()
	bad.MasterURL = "file:///etc/passwd"
	if err := bad.Validate(); err == nil {
		t.Fatal("expected invalid master URL to fail")
	}

	bad = validTarget()
	bad.User = "root;rm -rf /"
	if err := bad.Validate(); err == nil {
		t.Fatal("expected invalid user to fail")
	}
}

func TestStartScriptContainsEnvironment(t *testing.T) {
	target := validTarget()
	script := startScript(target)
	for _, want := range []string{
		"GLOSKI_MODE='agent'",
		"GLOSKI_AGENT_MASTER_URL='https://tower.tailnet.ts.net'",
		"GLOSKI_AGENT_JOIN_TOKEN='join-token'",
		"GLOSKI_AGENT_DEPLOY_ID='deploy-id'",
		"/tmp/gloski-agent",
		"/tmp/gloski-agent.pid",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("start script missing %q:\n%s", want, script)
		}
	}
}

func TestStartScriptQuotesName(t *testing.T) {
	target := validTarget()
	target.Name = `worker's "box"`
	script := startScript(target)
	if !strings.Contains(script, `GLOSKI_AGENT_NAME='worker'\''s "box"'`) {
		t.Fatalf("name was not shell-quoted:\n%s", script)
	}
}

func TestStopScriptUsesPidFile(t *testing.T) {
	script := stopScript(validTarget())
	for _, want := range []string{"/tmp/gloski-agent.pid", "echo stopped", "echo not-running"} {
		if !strings.Contains(script, want) {
			t.Fatalf("stop script missing %q:\n%s", want, script)
		}
	}
}

func TestLastLine(t *testing.T) {
	if got := lastLine("starting\n1234\n\n"); got != "1234" {
		t.Fatalf("lastLine() = %q", got)
	}
	if got := lastLine(""); got != "" {
		t.Fatalf("lastLine() = %q", got)
	}
}
