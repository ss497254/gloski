package provisioning

import "testing"

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
	if target.RemoteBase != "/tmp/gloski" {
		t.Fatalf("RemoteBase = %q, want /tmp/gloski", target.RemoteBase)
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

func TestField(t *testing.T) {
	output := "warning: something\nGLOSKI_INSTALLED pid=1234 arch=arm64 selfheal=cron dir=/tmp/gloski\n"
	if got := field(output, "GLOSKI_INSTALLED", "arch"); got != "arm64" {
		t.Fatalf("arch = %q", got)
	}
	if got := field(output, "GLOSKI_INSTALLED", "missing"); got != "" {
		t.Fatalf("missing = %q", got)
	}
	if got := field(output, "GLOSKI_NEED_UPLOAD", "arch"); got != "" {
		t.Fatalf("absent marker = %q", got)
	}
}
