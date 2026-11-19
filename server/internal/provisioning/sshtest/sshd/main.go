// Command sshd runs the test SSH server by itself, so a master under manual
// test has a "device" to install agents on. It runs commands as the current
// user with no authentication; never expose it beyond localhost.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/ss497254/gloski/internal/provisioning/sshtest"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:2222", "Address to listen on")
	flag.Parse()

	server, err := sshtest.Start(*addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("test SSH server listening on", server.Addr())
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt
}
