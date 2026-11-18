package controltower

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/ss497254/gloski/internal/logger"
)

func (t *Tower) handleTerminalWS(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	cwd := r.URL.Query().Get("cwd")
	connection, err := deviceWSUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer connection.Close()

	output, id, err := t.agentService.OpenTerminal(deviceID, cwd, 80, 24)
	if err != nil {
		writeAgentError(w, err)
		return
	}
	defer func() { _ = t.agentService.CloseTerminal(deviceID, id) }()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for data := range output {
			if err := connection.WriteMessage(websocket.BinaryMessage, data); err != nil {
				return
			}
		}
	}()

	connection.SetReadLimit(1 << 20)
	for {
		_, message, err := connection.ReadMessage()
		if err != nil {
			return
		}
		if err := t.agentService.TerminalInput(deviceID, id, message); err != nil {
			logger.Warn("Terminal input failed for %s: %v", deviceID, err)
			return
		}
	}
}
