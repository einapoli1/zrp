package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WSEvent is the payload broadcast to all connected WebSocket clients.
type WSEvent struct {
	Type   string `json:"type"`   // e.g. "eco_updated", "part_created"
	ID     any    `json:"id"`     // resource identifier (int or string)
	Action string `json:"action"` // "create", "update", "delete"
}

// client wraps a WebSocket connection with a mutex for thread-safe writes.
type client struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

// Hub maintains connected WebSocket clients and broadcasts events.
type Hub struct {
	mu      sync.RWMutex
	clients map[*client]struct{}
}

var wsHub = &Hub{
	clients: make(map[*client]struct{}),
}

func (h *Hub) register(c *client) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
	c.conn.Close()
}

// Broadcast sends an event to all connected clients.
func (h *Hub) Broadcast(evt WSEvent) {
	data, err := json.Marshal(evt)
	if err != nil {
		log.Printf("ws: marshal error: %v", err)
		return
	}
	h.mu.RLock()
	clients := make([]*client, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	for _, c := range clients {
		c.mu.Lock()
		_ = c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		err := c.conn.WriteMessage(websocket.TextMessage, data)
		c.mu.Unlock()
		
		if err != nil {
			h.unregister(c)
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// handleWebSocket upgrades the connection and keeps it alive with pings.
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws: upgrade error: %v", err)
		return
	}

	// Register the client wrapper
	c := &client{conn: conn}
	wsHub.register(c)
	
	wsHub.mu.RLock()
	clientCount := len(wsHub.clients)
	wsHub.mu.RUnlock()
	
	log.Printf("ws: client connected (%d total)", clientCount)

	// Keep-alive: read loop (handles pongs and detects disconnects)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Ping ticker
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			c.mu.Lock()
			err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
			c.mu.Unlock()
			if err != nil {
				return
			}
		}
	}()

	// Read loop — just discard messages, detect close
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
	wsHub.unregister(c)
	log.Printf("ws: client disconnected")
}

// broadcast is a convenience helper used by handlers.
func broadcast(resourceType, action string, id any) {
	wsHub.Broadcast(WSEvent{
		Type:   resourceType + "_" + action + "d",
		ID:     id,
		Action: action,
	})
}
