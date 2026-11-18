package controltower

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ss497254/gloski/internal/controltower/agents"
	"github.com/ss497254/gloski/internal/logger"
)

const (
	terminalWriteTimeout = 10 * time.Second
	// Idle WebSockets are cut by most reverse proxies after about a minute.
	terminalPingInterval = 25 * time.Second
	// resizeFrame is the first byte of the browser's 5-byte binary resize
	// frame: [0x01, cols_hi, cols_lo, rows_hi, rows_lo]. Keystrokes arrive as
	// text frames, so the two cannot be confused.
	resizeFrame = 0x01
)

func (t *Tower) handleTerminalWS(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	// Fail with a normal HTTP error while that is still possible; after the
	// upgrade the response belongs to the WebSocket.
	if !t.agentService.Connected(deviceID) {
		writeAgentError(w, agents.ErrNoAgent)
		return
	}
	query := r.URL.Query()
	cols, rows := terminalSize(query.Get("cols"), 80), terminalSize(query.Get("rows"), 24)

	connection, err := deviceWSUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer connection.Close()

	output, id, err := t.agentService.OpenTerminal(deviceID, query.Get("cwd"), cols, rows)
	if err != nil {
		closeTerminalSocket(connection, websocket.CloseTryAgainLater, "agent is not connected")
		return
	}
	defer func() { _ = t.agentService.CloseTerminal(deviceID, id) }()

	stop := make(chan struct{})
	defer close(stop)

	// Writer: the only goroutine that sends data frames. Closing the socket
	// when the shell ends unblocks the read loop below.
	go func() {
		defer connection.Close()
		for {
			data, ok := output.Next(stop)
			if !ok {
				closeTerminalSocket(connection, websocket.CloseNormalClosure, "session ended")
				return
			}
			_ = connection.SetWriteDeadline(time.Now().Add(terminalWriteTimeout))
			if err := connection.WriteMessage(websocket.BinaryMessage, data); err != nil {
				return
			}
		}
	}()

	// Pings keep the socket alive through proxies. WriteControl is safe to
	// call alongside the writer.
	go func() {
		ticker := time.NewTicker(terminalPingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				if err := connection.WriteControl(websocket.PingMessage, nil, time.Now().Add(terminalWriteTimeout)); err != nil {
					return
				}
			}
		}
	}()

	connection.SetReadLimit(1 << 20)
	for {
		messageType, message, err := connection.ReadMessage()
		if err != nil {
			return
		}
		if messageType == websocket.BinaryMessage && len(message) == 5 && message[0] == resizeFrame {
			cols := uint32(message[1])<<8 | uint32(message[2])
			rows := uint32(message[3])<<8 | uint32(message[4])
			err = t.agentService.ResizeTerminal(deviceID, id, cols, rows)
		} else {
			err = t.agentService.TerminalInput(deviceID, id, message)
		}
		if err != nil {
			logger.Warn("Terminal input failed for %s: %v", deviceID, err)
			return
		}
	}
}

func terminalSize(raw string, fallback uint32) uint32 {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 || value > 0xffff {
		return fallback
	}
	return uint32(value)
}

func closeTerminalSocket(connection *websocket.Conn, code int, reason string) {
	_ = connection.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), time.Now().Add(terminalWriteTimeout))
}
