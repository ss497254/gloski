package controltower

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

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

// agentHTTPHandler serves plain HTTP on the agent port.
func (t *Tower) agentHTTPHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})

	return mux
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
