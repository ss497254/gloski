package provisioning

import (
	"os"
	"strings"
)

// field finds a line that starts with marker and returns the value of its
// key=value word, or "" when either is missing.
func field(output, marker, key string) string {
	for _, line := range strings.Split(output, "\n") {
		words := strings.Fields(line)
		if len(words) == 0 || words[0] != marker {
			continue
		}
		for _, word := range words[1:] {
			if value, ok := strings.CutPrefix(word, key+"="); ok {
				return value
			}
		}
	}
	return ""
}

func quote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func currentUser() string {
	if user := strings.TrimSpace(os.Getenv("USER")); user != "" {
		return user
	}
	return strings.TrimSpace(os.Getenv("USERNAME"))
}
