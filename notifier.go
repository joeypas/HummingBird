package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
)

func listenAndFan(ctx context.Context, h *Hub) {
	conn, err := db.Acquire(ctx)
	if err != nil {
		log.Fatalf("pg acquire: %v", err)
	}
	defer conn.Release()

	_, err = conn.Exec(ctx, `LISTEN chat`)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	log.Println("LISTEN chat")

	for {
		notification, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			log.Printf("wait: %v (retrying in 1s)", err)
			time.Sleep(time.Second)
			continue
		}
		var meta struct {
			Room string    `json:"room"`
			ID   uuid.UUID `json:"id"`
		}
		if json.Unmarshal([]byte(notification.Payload), &meta) != nil {
			log.Fatal("json Unmarshal")
			continue
		}

		msg, err := fetchMessage(ctx, meta.ID)
		if err != nil {
			log.Fatal(err)
			continue
		}
		frame, _ := json.Marshal(struct {
			Type string  `json:"type"`
			Data Message `json:"data"`
		}{"message.new", msg})

		h.broadcast(meta.Room, frame) // mutex-guarded hub method
	}
}
