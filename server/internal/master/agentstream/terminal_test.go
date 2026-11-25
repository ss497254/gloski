package agentstream

import (
	"bytes"
	"strings"
	"testing"
)

func TestTerminalOutputBatchesAndCloses(t *testing.T) {
	output := newTerminalOutput()
	output.push([]byte("one "))
	output.push([]byte("two"))

	data, ok := output.Next(nil)
	if !ok || string(data) != "one two" {
		t.Fatalf("Next = %q, %v", data, ok)
	}

	// Output pushed before the session ends is still delivered.
	output.push([]byte("last"))
	output.close()
	if data, ok := output.Next(nil); !ok || string(data) != "last" {
		t.Fatalf("Next before close = %q, %v", data, ok)
	}
	if _, ok := output.Next(nil); ok {
		t.Fatal("Next after a drained, closed session reported more output")
	}
}

func TestTerminalOutputSkipsWhenBrowserFallsBehind(t *testing.T) {
	output := newTerminalOutput()
	piece := bytes.Repeat([]byte("x"), 1<<20)
	for i := 0; i < 6; i++ {
		output.push(piece)
	}
	output.push([]byte("TAIL"))

	data, ok := output.Next(nil)
	if !ok {
		t.Fatal("no output")
	}
	if len(data) > maxTerminalBacklog+len(terminalSkipNotice) {
		t.Fatalf("backlog grew to %d bytes", len(data))
	}
	if !strings.HasPrefix(string(data), terminalSkipNotice) || !strings.HasSuffix(string(data), "TAIL") {
		t.Fatal("overflow should drop the oldest output, say so, and keep the newest")
	}
}

func TestTerminalOutputStops(t *testing.T) {
	output := newTerminalOutput()
	stop := make(chan struct{})
	close(stop)
	if _, ok := output.Next(stop); ok {
		t.Fatal("Next ignored stop")
	}
}
