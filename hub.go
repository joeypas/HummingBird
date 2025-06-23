package main

import "sync"

type Hub struct {
	rooms map[string]map[*Client]bool
	mu    sync.Mutex
}

func NewHub() *Hub {
	return &Hub{rooms: map[string]map[*Client]bool{}}
}

func (h *Hub) join(room string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	set, ok := h.rooms[room]
	if !ok {
		set = make(map[*Client]bool)
		h.rooms[room] = set
	}
	set[c] = true
}

func (h *Hub) leave(room string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.rooms[room]; ok {
		delete(set, c)
		if len(set) == 0 {
			delete(h.rooms, room)
		}
	}
}

func (h *Hub) broadcast(room string, msg []byte) {
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
