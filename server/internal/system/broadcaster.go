package system

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ss497254/gloski/internal/logger"
)

// Client represents a WebSocket client connection
type Client struct {
	Conn      *websocket.Conn
	Send      chan []byte
	Hub       *Hub
	closeOnce sync.Once
}

// Hub maintains active WebSocket connections and broadcasts stats updates
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan *Stats
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	store      *Store
}

// NewHub creates a new WebSocket hub
func NewHub(store *Store) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan *Stats, 10),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		store:      store,
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			logger.Debug("WebSocket client connected (total: %d)", len(h.clients))

			// Send current stats immediately on connect
			if stats := h.store.Get(); stats != nil {
				h.sendToClient(client, stats)
			}

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
			logger.Debug("WebSocket client disconnected (total: %d)", len(h.clients))

		case stats := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- h.encodeStats(stats):
				default:
					// Client send buffer full, disconnect
					go func(c *Client) {
						h.unregister <- c
					}(client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends stats to all connected clients
func (h *Hub) Broadcast(stats *Stats) {
	select {
	case h.broadcast <- stats:
	default:
		// Broadcast channel full, skip this update
		logger.Warn("WebSocket broadcast channel full, dropping update")
	}
}

// Register adds a client to the hub
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// ClientCount returns the number of connected clients
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// sendToClient sends stats to a specific client
func (h *Hub) sendToClient(client *Client, stats *Stats) {
	select {
	case client.Send <- h.encodeStats(stats):
	default:
		// Client send buffer full, skip
	}
}

// encodeStats encodes stats to JSON
func (h *Hub) encodeStats(stats *Stats) []byte {
	data, err := json.Marshal(stats)
	if err != nil {
		logger.Error("Failed to encode stats: %v", err)
		return []byte("{}")
	}
	return data
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Hub closed the channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.Hub.Unregister(c)
		c.close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Debug("WebSocket error: %v", err)
			}
			break
		}
	}
}

// close closes the client connection
func (c *Client) close() {
	c.closeOnce.Do(func() {
		c.Conn.Close()
	})
}

// Start starts the client's read and write pumps
func (c *Client) Start() {
	go c.writePump()
	go c.readPump()
}
