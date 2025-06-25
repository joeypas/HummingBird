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
	c := newClient(nil, "room1", uid)

	h.join("room1", c)
	if _, ok := h.rooms["room1"]; !ok {
		t.Fatalf("room not created")
	}
	if !h.rooms["room1"][c] {
		t.Fatalf("client not joined")
	}

	h.leave("room1", c)
	if _, ok := h.rooms["room1"]; ok {
		t.Fatalf("room not removed after leaving")
	}
}

func TestHubBroadcast(t *testing.T) {
	h := NewHub()
	uid, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("create uuid")
	}
	c := newClient(nil, "room1", uid)
	h.join("room1", c)

	msg := []byte("hello")
	h.broadcast("room1", msg)

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
	c := newClient(nil, "room1", uid)
	if c.room != "room1" {
		t.Fatalf("room not set")
	}
	if c.send == nil || cap(c.send) != 256 {
		t.Fatalf("send channel not initialized correctly")
	}
}
