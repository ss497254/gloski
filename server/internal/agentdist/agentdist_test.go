package agentdist

import (
	"os"
	"os/exec"
	"path/filepath"
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
