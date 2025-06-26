package main

import (
	"sync"

	"github.com/gofrs/uuid/v5"
)

type Hub struct {
	rooms map[uuid.UUID]map[*Client]bool
	mu    sync.Mutex
}

func NewHub() *Hub {
	return &Hub{rooms: map[uuid.UUID]map[*Client]bool{}}
}

func (h *Hub) join(room uuid.UUID, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	set, ok := h.rooms[room]
	if !ok {
		set = make(map[*Client]bool)
		h.rooms[room] = set
	}
	set[c] = true
}

func (h *Hub) leave(room uuid.UUID, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.rooms[room]; ok {
		delete(set, c)
		if len(set) == 0 {
			delete(h.rooms, room)
		}
	}
}

func (h *Hub) broadcast(room uuid.UUID, msg []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.rooms[room] {
		select {
		case c.send <- msg:
		default:
			go c.conn.Close()
			delete(h.rooms[room], c)
		}
	}
}
