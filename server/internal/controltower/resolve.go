package controltower

import (
	"os"
	"path/filepath"
)

func resolveAgentBinary(configured string) string {
	candidates := []string{configured}
	if executable, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(executable), "gloski-agent"))
	}
	candidates = append(candidates,
		filepath.Join("bin", "gloski-agent"),
		filepath.Join("..", "bin", "gloski-agent"),
		filepath.Join("..", "..", "bin", "gloski-agent"),
	)

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return configured
}
