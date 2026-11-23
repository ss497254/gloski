// Package agentdist holds what the master hands to devices: agent builds for
// each architecture and the install and uninstall scripts.
package agentdist

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

//go:embed install-agent.sh
var installScript string

//go:embed uninstall-agent.sh
var uninstallScript string

// DefaultDir is where agents are installed on a device.
const DefaultDir = "/tmp/gloski"

// Arches lists the Linux architectures the master can serve.
var Arches = []string{"amd64", "arm64", "arm"}

var safeDir = regexp.MustCompile(`^/[A-Za-z0-9._/-]*[A-Za-z0-9._-]$`)

// ValidDir reports whether dir is an acceptable install directory. It is
// written into shell scripts and a crontab line, so the set is kept small.
func ValidDir(dir string) bool {
	return safeDir.MatchString(dir) && !strings.Contains(dir, "..") && !strings.Contains(dir, "//")
}

// Build is one agent binary.
type Build struct {
	Data   []byte
	SHA256 string
}

// Dist serves agent builds. Embedded builds win; fallbackPath (the agent
// binary next to the master) covers the master's own architecture when the
// master was built without embedded agents, as with "go run".
type Dist struct {
	builds       fs.FS
	fallbackPath string

	mu     sync.Mutex
	loaded map[string]*Build
}

// New creates a Dist. builds may be nil.
func New(builds fs.FS, fallbackPath string) *Dist {
	return &Dist{builds: builds, fallbackPath: fallbackPath, loaded: make(map[string]*Build)}
}

// Build returns the agent for linux/arch, or nil when the master has none.
func (d *Dist) Build(arch string) *Build {
	d.mu.Lock()
	defer d.mu.Unlock()
	if build, ok := d.loaded[arch]; ok {
		return build
	}
	var build *Build
	var data []byte
	err := fs.ErrNotExist
	if d.builds != nil {
		data, err = fs.ReadFile(d.builds, "gloski-agent-linux-"+arch)
	}
	if err != nil && arch == runtime.GOARCH && runtime.GOOS == "linux" && d.fallbackPath != "" {
		data, err = os.ReadFile(d.fallbackPath)
	}
	if err == nil && len(data) > 0 {
		sum := sha256.Sum256(data)
		build = &Build{Data: data, SHA256: hex.EncodeToString(sum[:])}
	}
	d.loaded[arch] = build
	return build
}

// Available reports whether any agent build can be served.
func (d *Dist) Available() bool {
	for _, arch := range Arches {
		if d.Build(arch) != nil {
			return true
		}
	}
	return false
}

// InstallParams fills the install script.
type InstallParams struct {
	Dir       string
	MasterURL string // gRPC address the agent connects to
	JoinToken string
	Name      string
	DeployID  string
	// ConfigJSON, when set, is written verbatim instead of generating one from
	// the fields above. The master uses it to keep operator-supplied settings
	// across a reinstall.
	ConfigJSON string
	// Force restarts the agent even when nothing changed.
	Force bool
}

// InstallScript renders the installer for one device.
func (d *Dist) InstallScript(p InstallParams) string {
	if p.Dir == "" {
		p.Dir = DefaultDir
	}
	sha := func(arch string) string {
		if build := d.Build(arch); build != nil {
			return build.SHA256
		}
		return ""
	}
	config := []byte(p.ConfigJSON)
	if len(config) == 0 {
		config, _ = json.MarshalIndent(map[string]any{
			"agent": map[string]any{
				"master_url": p.MasterURL,
				"join_token": p.JoinToken,
				"name":       oneLine(p.Name),
				"deploy_id":  p.DeployID,
			},
		}, "", "  ")
	}
	force := "0"
	if p.Force {
		force = "1"
	}
	return strings.NewReplacer(
		"@@DIR@@", quote(p.Dir),
		"@@FORCE@@", force,
		"@@CONFIG@@", string(config),
		"@@SHA_AMD64@@", quote(sha("amd64")),
		"@@SHA_ARM64@@", quote(sha("arm64")),
		"@@SHA_ARM@@", quote(sha("arm")),
	).Replace(installScript)
}

// UninstallScript renders the uninstaller. With stopOnly it stops the agent
// and its self-heal but leaves the files.
func UninstallScript(dir string, stopOnly bool) string {
	if dir == "" {
		dir = DefaultDir
	}
	mode := "remove"
	if stopOnly {
		mode = "stop"
	}
	return strings.NewReplacer("@@DIR@@", quote(dir), "@@MODE@@", mode).Replace(uninstallScript)
}

func oneLine(value string) string {
	return strings.TrimSpace(strings.NewReplacer("\n", " ", "\r", " ").Replace(value))
}

func quote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

// Builds lists the architectures that can be served.
func (d *Dist) Builds() []string {
	available := []string{}
	for _, arch := range Arches {
		if d.Build(arch) != nil {
			available = append(available, arch)
		}
	}
	return available
}
