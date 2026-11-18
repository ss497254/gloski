package fleet

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ss497254/gloski/internal/logger"
)

const (
	wsWriteTimeout = 10 * time.Second
	wsPongTimeout  = 60 * time.Second
	wsPingInterval = 25 * time.Second
)

// EventHub fans fleet events out to connected dashboards.
type EventHub struct {
	mu      sync.RWMutex
	clients map[*wsClient]struct{}
}

type wsClient struct {
	conn *websocket.Conn
	send chan []byte
}

// NewHub creates a fleet event hub.
func NewEventHub() *EventHub {
	return &EventHub{clients: make(map[*wsClient]struct{})}
}

// Broadcast sends an event to all connected dashboards. Slow clients are
// dropped rather than blocking the heartbeat path.
func (h *EventHub) Broadcast(ev Event) {
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients {
		select {
		case client.send <- data:
		default:
			// The read loop owns closing the send channel after the
			// connection fails, so only evict and close the socket here.
			delete(h.clients, client)
			client.conn.Close()
		}
	}
}

// ServeWS upgrades an HTTP request and streams fleet events.
func (h *EventHub) ServeWS(w http.ResponseWriter, r *http.Request, snapshot func() ([]Device, error)) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 4096,
		// Origin is enforced by the master's httpapi.SameOrigin middleware.
		CheckOrigin: func(*http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Debug("Fleet websocket upgrade failed: %v", err)
		return
	}

	client := &wsClient{conn: conn, send: make(chan []byte, 32)}

	// Send the current fleet immediately so a new dashboard does not wait for
	// the next heartbeat.
	if list, err := snapshot(); err == nil {
		if data, err := json.Marshal(Event{Type: "snapshot", Devices: list}); err == nil {
			client.send <- data
		}
	}

	h.mu.Lock()
	h.clients[client] = struct{}{}
	h.mu.Unlock()

	go h.writeLoop(client)
	h.readLoop(client)
}

func (h *EventHub) writeLoop(client *wsClient) {
	ticker := time.NewTicker(wsPingInterval)
	defer ticker.Stop()
	// Unblock the read loop if writing fails first.
	defer client.conn.Close()

	for {
		select {
		case data, ok := <-client.send:
			if !ok {
				return
			}
			client.conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
			if err := client.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (h *EventHub) readLoop(client *wsClient) {
	defer func() {
		h.mu.Lock()
		delete(h.clients, client)
		close(client.send)
		h.mu.Unlock()
		client.conn.Close()
	}()

	client.conn.SetReadDeadline(time.Now().Add(wsPongTimeout))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(wsPongTimeout))
		return nil
	})

	for {
		if _, _, err := client.conn.ReadMessage(); err != nil {
			return
		}
	}
}
