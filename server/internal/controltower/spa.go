package controltower

import (
	"fmt"
	"net/http"
	"os"
	"path"

	"github.com/ss497254/gloski/internal/api/handlers"
	"path/filepath"
	"strings"
)

func (t *Tower) serveSPA(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		handlers.NotFound(w, "endpoint not found")
		return
	}

	clean := path.Clean("/" + r.URL.Path)
	if clean == "/" {
		t.writeIndex(w)
		return
	}

	if t.webDir != "" {
		file := filepath.Join(t.webDir, filepath.FromSlash(strings.TrimPrefix(clean, "/")))
		if info, err := os.Stat(file); err == nil && !info.IsDir() {
			if strings.HasPrefix(clean, "/assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			http.ServeFile(w, r, file)
			return
		}
	}

	// Client-side routes fall back to index.html; missing assets return 404.
	if path.Ext(clean) != "" {
		http.NotFound(w, r)
		return
	}
	t.writeIndex(w)
}

func (t *Tower) writeIndex(w http.ResponseWriter) {
	// The dashboard can be rebuilt while the master is running. Always read
	// index.html from disk so asset hashes stay in sync with web/dist; the
	// startup copy is only a fallback.
	index := t.indexHTML
	if t.webDir != "" {
		if data, err := os.ReadFile(filepath.Join(t.webDir, "index.html")); err == nil && len(data) > 0 {
			index = data
		}
	}
	if len(index) == 0 {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, placeholderHTML)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(index)
}

func resolveWebDir(configured string) string {
	candidates := []string{configured}
	if configured == "" || configured == "web/dist" {
		if exe, err := os.Executable(); err == nil {
			candidates = append(candidates,
				filepath.Join(filepath.Dir(exe), "web", "dist"),
				filepath.Join(filepath.Dir(exe), "dist"),
			)
		}
		candidates = append(candidates, "web/dist", "../web/dist", "../../web/dist")
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if info, err := os.Stat(filepath.Join(candidate, "index.html")); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

func displayPath(dir string) string {
	if dir == "" {
		return "(dashboard not bundled; run make build-web)"
	}
	return dir
}

const placeholderHTML = `<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>Gloski Tower</title>
<style>
body{font:14px/1.6 ui-monospace,SFMono-Regular,Menlo,monospace;background:#0b0f14;color:#c9d1d9;display:grid;place-items:center;height:100vh;margin:0}
main{max-width:42rem;padding:2rem}
h1{color:#7ee787;font-size:1.2rem}code{color:#79c0ff}
</style></head>
<body><main>
<h1>Gloski control tower is running</h1>
<p>The API is online, but the dashboard bundle was not found.</p>
<p>Build it with <code>make build-web</code> (or <code>cd web &amp;&amp; bun run build</code>) and restart the master, or point <code>master.web_dir</code> at <code>web/dist</code>.</p>
<p>API: <code>/api/tower/info</code> · <code>/api/devices</code></p>
</main></body></html>`
