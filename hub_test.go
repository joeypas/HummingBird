package main

import (
	"testing"

	"github.com/gofrs/uuid/v5"
)

func TestHubJoinLeave(t *testing.T) {
	h := NewHub()
	uid, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("create uuid")
	}
	room_id, err := uuid.FromString("d1ed0b21-a6d9-4aa6-8f0c-b375207c303e")
	if err != nil {
		t.Fatalf("room uuid")
	}
	c := newClient(nil, room_id, uid)

	h.join(room_id, c)
	if _, ok := h.rooms[room_id]; !ok {
		t.Fatalf("room not created")
	}
	if !h.rooms[room_id][c] {
		t.Fatalf("client not joined")
	}

	h.leave(room_id, c)
	if _, ok := h.rooms[room_id]; ok {
		t.Fatalf("room not removed after leaving")
	}
}

func TestHubBroadcast(t *testing.T) {
	h := NewHub()
	uid, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("create uuid")
	}
	room_id, err := uuid.FromString("d1ed0b21-a6d9-4aa6-8f0c-b375207c303e")
	if err != nil {
		t.Fatalf("room uuid")
	}
	c := newClient(nil, room_id, uid)
	h.join(room_id, c)

	msg := []byte("hello")
	h.broadcast(room_id, msg)

	select {
	case got := <-c.send:
		if string(got) != string(msg) {
			t.Fatalf("expected %q, got %q", msg, got)
		}
	default:
		t.Fatalf("no message broadcasted")
	}
}

func TestNewClient(t *testing.T) {
	uid, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("create uuid")
	}
	room_id, err := uuid.FromString("d1ed0b21-a6d9-4aa6-8f0c-b375207c303e")
	if err != nil {
		t.Fatalf("room uuid")
	}
	c := newClient(nil, room_id, uid)
	if c.room != room_id {
		t.Fatalf("room not set")
	}
	if c.send == nil || cap(c.send) != 256 {
		t.Fatalf("send channel not initialized correctly")
	}
}
