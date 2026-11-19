package controltower

import (
	"bufio"
	"crypto/subtle"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ss497254/gloski/internal/agentdist"
	"github.com/ss497254/gloski/internal/logger"
)

// The agent port carries two protocols. Agents speak gRPC over HTTP/2, which
// always opens with the bytes "PRI "; anything else is a plain HTTP request
// for the install script or an agent binary. splitListener looks at the first
// bytes of each connection and hands it to the right server.
func splitListener(listener net.Listener) (grpcSide, httpSide net.Listener) {
	closed := make(chan struct{})
	var once sync.Once
	closeAll := func() error {
		once.Do(func() { close(closed) })
		return listener.Close()
	}
	grpcConns := make(chan net.Conn)
	httpConns := make(chan net.Conn)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				once.Do(func() { close(closed) })
				return
			}
			go func() {
				_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
				reader := bufio.NewReader(conn)
				head, err := reader.Peek(4)
				if err != nil {
					_ = conn.Close()
					return
				}
				_ = conn.SetReadDeadline(time.Time{})
				target := httpConns
				if string(head) == "PRI " {
					target = grpcConns
				}
				select {
				case target <- &peekedConn{Conn: conn, reader: reader}:
				case <-closed:
					_ = conn.Close()
				}
			}()
		}
	}()

	return &chanListener{conns: grpcConns, closed: closed, close: closeAll, addr: listener.Addr()},
		&chanListener{conns: httpConns, closed: closed, close: closeAll, addr: listener.Addr()}
}

type peekedConn struct {
	net.Conn
	reader *bufio.Reader
}

func (c *peekedConn) Read(p []byte) (int, error) { return c.reader.Read(p) }

type chanListener struct {
	conns  chan net.Conn
	closed chan struct{}
	close  func() error
	addr   net.Addr
}

func (l *chanListener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.conns:
		return conn, nil
	case <-l.closed:
		return nil, net.ErrClosed
	}
}

func (l *chanListener) Close() error   { return l.close() }
func (l *chanListener) Addr() net.Addr { return l.addr }

// agentHTTPHandler serves what a device needs to install an agent.
func (t *Tower) agentHTTPHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})

	mux.HandleFunc("GET /install-agent.sh", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		token := t.fleetSvc.JoinToken()
		if subtle.ConstantTimeCompare([]byte(query.Get("token")), []byte(token)) != 1 {
			logger.Warn("Install script request from %s rejected: wrong token", r.RemoteAddr)
			http.Error(w, "wrong or missing token; copy the install command from the dashboard again", http.StatusForbidden)
			return
		}
		dir := query.Get("dir")
		if dir == "" {
			dir = agentdist.DefaultDir
		}
		if !agentdist.ValidDir(dir) {
			http.Error(w, "dir must be an absolute path of letters, digits, '.', '_', '-' and '/'", http.StatusBadRequest)
			return
		}
		// The device reached the master at r.Host, so that address is known
		// to work from there; the agent uses it for gRPC too.
		master := "http://" + r.Host
		w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
		fmt.Fprint(w, t.dist.InstallScript(agentdist.InstallParams{
			Dir:        dir,
			MasterHTTP: master,
			MasterURL:  master,
			JoinToken:  token,
			Name:       strings.TrimSpace(query.Get("name")),
			DeployID:   query.Get("deploy_id"),
		}))
	})

	mux.HandleFunc("GET /uninstall-agent.sh", func(w http.ResponseWriter, r *http.Request) {
		dir := r.URL.Query().Get("dir")
		if dir == "" {
			dir = agentdist.DefaultDir
		}
		if !agentdist.ValidDir(dir) {
			http.Error(w, "invalid dir", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
		fmt.Fprint(w, agentdist.UninstallScript(dir, r.URL.Query().Get("mode") == "stop"))
	})

	mux.HandleFunc("GET /agent/linux/{arch}", func(w http.ResponseWriter, r *http.Request) {
		build := t.dist.Build(r.PathValue("arch"))
		if build == nil {
			http.Error(w, "the master has no agent build for this architecture", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", strconv.Itoa(len(build.Data)))
		w.Header().Set("X-Checksum-Sha256", build.SHA256)
		_, _ = w.Write(build.Data)
	})

	return mux
}

// installCommand is the one-liner a person pastes on a device.
func (t *Tower) installCommand() string {
	return fmt.Sprintf("curl -fsSL '%s/install-agent.sh?token=%s' | sh", t.agentGRPCURL(), t.fleetSvc.JoinToken())
}

// localAgentBinary returns the agent the master runs for its own machine: the
// build it serves to devices, written into the data directory, so the master
// and its own agent can never be different versions.
func (t *Tower) localAgentBinary() string {
	build := t.dist.Build(runtime.GOARCH)
	if build == nil || runtime.GOOS != "linux" {
		return t.agentBinary
	}
	path := filepath.Join(t.cfg.DataDir, "gloski-agent")
	temp := path + ".new"
	if err := os.WriteFile(temp, build.Data, 0755); err != nil {
		logger.Error("Cannot write local agent binary: %v", err)
		return ""
	}
	if err := os.Rename(temp, path); err != nil {
		logger.Error("Cannot place local agent binary: %v", err)
		return ""
	}
	return path
}
