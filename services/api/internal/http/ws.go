package httpapi

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

type WebSocketHub struct {
	mu    sync.RWMutex
	conns map[string]map[*websocket.Conn]struct{}
}

type EventMessage struct {
	Event   string `json:"event"`
	Payload any    `json:"payload"`
}

func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{conns: map[string]map[*websocket.Conn]struct{}{}}
}

func (h *WebSocketHub) Register(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.conns[userID] == nil {
		h.conns[userID] = map[*websocket.Conn]struct{}{}
	}
	h.conns[userID][conn] = struct{}{}
}

func (h *WebSocketHub) Unregister(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.conns[userID] != nil {
		delete(h.conns[userID], conn)
		if len(h.conns[userID]) == 0 {
			delete(h.conns, userID)
		}
	}

	_ = conn.Close()
}

func (h *WebSocketHub) PublishToUsers(userIDs []string, event string, payload any) {
	message, err := json.Marshal(EventMessage{Event: event, Payload: payload})
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, userID := range userIDs {
		for conn := range h.conns[userID] {
			_ = conn.WriteMessage(websocket.TextMessage, message)
		}
	}
}
