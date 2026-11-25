package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
)

// machineID returns a stable identifier for this machine, or "" if none is
// available. The raw machine-id is a host secret, so only a hash leaves the
// device.
func machineID() string {
	for _, path := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if raw := strings.TrimSpace(string(data)); raw != "" {
			sum := sha256.Sum256([]byte("gloski-machine:" + raw))
			return hex.EncodeToString(sum[:16])
		}
	}
	return ""
}
