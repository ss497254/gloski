package master

import (
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestSplitListenerRoutesByProtocol(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	grpcSide, httpSide := splitListener(listener)
	defer grpcSide.Close()

	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "plain http")
	})}
	go func() { _ = server.Serve(httpSide) }()

	gotPreface := make(chan string, 1)
	go func() {
		conn, err := grpcSide.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buffer := make([]byte, 24)
		_, _ = io.ReadFull(conn, buffer)
		gotPreface <- string(buffer)
	}()

	response, err := http.Get("http://" + listener.Addr().String() + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if string(body) != "plain http" {
		t.Fatalf("HTTP request got %q", body)
	}

	const preface = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"
	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_, _ = io.WriteString(conn, preface)
	select {
	case got := <-gotPreface:
		if got != preface {
			t.Fatalf("gRPC side read %q; the peeked bytes were lost", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("HTTP/2 connection was not routed to the gRPC side")
	}

	_ = grpcSide.Close()
	if _, err := httpSide.Accept(); err == nil {
		t.Fatal("Accept() should fail after close")
	}
}
