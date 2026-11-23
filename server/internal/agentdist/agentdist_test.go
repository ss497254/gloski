package agentdist

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"
)

func TestBuildPrefersEmbedded(t *testing.T) {
	dist := New(fstest.MapFS{"gloski-agent-linux-arm64": {Data: []byte("arm build")}}, "")
	build := dist.Build("arm64")
	if build == nil || string(build.Data) != "arm build" || len(build.SHA256) != 64 {
		t.Fatalf("Build(arm64) = %+v", build)
	}
	if dist.Build("arm") != nil {
		t.Fatal("Build(arm) should be nil when nothing is embedded")
	}
	if got := dist.Builds(); len(got) != 1 || got[0] != "arm64" {
		t.Fatalf("Builds() = %v", got)
	}
}

func TestValidDir(t *testing.T) {
	for _, dir := range []string{"/tmp/gloski", "/opt/gloski-agent", "/home/pi/.gloski"} {
		if !ValidDir(dir) {
			t.Errorf("ValidDir(%q) = false", dir)
		}
	}
	for _, dir := range []string{"", "tmp", "/tmp/../etc", "/tmp/a b", "/tmp/$(id)", "/tmp//x", "/tmp/x/"} {
		if ValidDir(dir) {
			t.Errorf("ValidDir(%q) = true", dir)
		}
	}
}

// The scripts must at least parse, with awkward values in every field.
func TestScriptsAreValidShell(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh")
	}
	dist := New(nil, "")
	scripts := map[string]string{
		"install": dist.InstallScript(InstallParams{
			MasterURL: "http://tower:9090",
			JoinToken: "tok'en", Name: `it's "a" $HOME box`, DeployID: "id",
		}),
		"uninstall": UninstallScript("", false),
		"stop":      UninstallScript("/opt/gloski", true),
	}
	for name, script := range scripts {
		if strings.Contains(script, "@@") {
			t.Errorf("%s script has an unfilled placeholder", name)
		}
		path := filepath.Join(t.TempDir(), name+".sh")
		if err := os.WriteFile(path, []byte(script), 0600); err != nil {
			t.Fatal(err)
		}
		if output, err := exec.Command("sh", "-n", path).CombinedOutput(); err != nil {
			t.Errorf("%s script does not parse: %v\n%s", name, err, output)
		}
	}
}

func TestInstallScriptUsesSuppliedConfig(t *testing.T) {
	dist := New(nil, "")
	merged := `{"agent":{"join_token":"kept","port":18099}}`
	script := dist.InstallScript(InstallParams{ConfigJSON: merged})
	if !strings.Contains(script, `"port":18099`) || !strings.Contains(script, `"join_token":"kept"`) {
		t.Fatalf("script did not use the merged config: %s", script)
	}
	if strings.Contains(script, `"master_url"`) {
		t.Fatalf("generated config should have been replaced: %s", script)
	}
}

func TestInstallRollsBackOnChecksumFailure(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh")
	}
	dir := t.TempDir()
	oldBinary := []byte("previous agent binary")
	oldConfig := []byte(`{"agent":{"port":19999}}`)
	for name, content := range map[string][]byte{
		"gloski-agent":        oldBinary,
		"agent.json":          oldConfig,
		"gloski-agent.upload": []byte("#!/bin/sh\nexit 0\n"),
	} {
		mode := os.FileMode(0600)
		if name != "agent.json" {
			mode = 0755
		}
		if err := os.WriteFile(filepath.Join(dir, name), content, mode); err != nil {
			t.Fatal(err)
		}
	}

	script := New(nil, "").InstallScript(InstallParams{Dir: dir, ConfigJSON: string(oldConfig)})
	script = regexp.MustCompile(`(?m)^SHA_(amd64|arm64|arm)=.*$`).ReplaceAllString(script, "SHA_$1='deadbeef'")
	scriptPath := filepath.Join(t.TempDir(), "install.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command("sh", scriptPath).CombinedOutput()
	if err == nil {
		t.Fatalf("a bad checksum must fail the install: %s", output)
	}
	if !strings.Contains(string(output), "replace:") {
		t.Fatalf("failure did not name its phase: %s", output)
	}
	if !strings.Contains(string(output), "restored the previous agent files") {
		t.Fatalf("rollback was not reported: %s", output)
	}

	gotBinary, err := os.ReadFile(filepath.Join(dir, "gloski-agent"))
	if err != nil || !bytes.Equal(gotBinary, oldBinary) {
		t.Fatalf("previous binary was not restored: %q, %v", gotBinary, err)
	}
	gotConfig, err := os.ReadFile(filepath.Join(dir, "agent.json"))
	if err != nil || !bytes.Equal(gotConfig, oldConfig) {
		t.Fatalf("previous config was not restored: %q, %v", gotConfig, err)
	}
	for _, leftover := range []string{"gloski-agent.bak", "agent.json.bak", "gloski-agent.upload"} {
		if _, err := os.Stat(filepath.Join(dir, leftover)); !os.IsNotExist(err) {
			t.Errorf("rollback left %s behind: %v", leftover, err)
		}
	}
	staged, err := filepath.Glob(filepath.Join(dir, ".gloski-agent.*"))
	if err != nil || len(staged) != 0 {
		t.Errorf("rollback left staging files behind: %v, %v", staged, err)
	}
}

func TestInstallRollsBackFreshInstall(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "gloski-agent.upload"), []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}

	script := New(nil, "").InstallScript(InstallParams{Dir: dir})
	script = regexp.MustCompile(`(?m)^SHA_(amd64|arm64|arm)=.*$`).ReplaceAllString(script, "SHA_$1='deadbeef'")
	scriptPath := filepath.Join(t.TempDir(), "install.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command("sh", scriptPath).CombinedOutput()
	if err == nil {
		t.Fatalf("a bad checksum must fail the install: %s", output)
	}
	if !strings.Contains(string(output), "removed the partial install") {
		t.Fatalf("fresh-install rollback was not reported: %s", output)
	}
	for _, leftover := range []string{
		"gloski-agent", "agent.json", "gloski-agent.sha256", "gloski-agent.bak",
		"agent.json.bak", "gloski-agent.sha256.bak", "gloski-agent.upload", "agent.json.new",
	} {
		if _, err := os.Stat(filepath.Join(dir, leftover)); !os.IsNotExist(err) {
			t.Errorf("failed fresh install left %s behind: %v", leftover, err)
		}
	}
}

func TestInstallScriptNamesPhases(t *testing.T) {
	script := New(nil, "").InstallScript(InstallParams{})
	for _, phase := range []string{"PHASE=platform", "PHASE=prepare", "PHASE=binary", "PHASE=config", "PHASE=replace", "PHASE=start"} {
		if !strings.Contains(script, phase) {
			t.Errorf("install script is missing %s", phase)
		}
	}
}
