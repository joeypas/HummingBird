package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Message struct {
	ID       uuid.UUID `json:"id"`
	Room     string    `json:"room"`
	SenderID uuid.UUID `json:"sender_id"`
	Body     string    `json:"body"`
	SentAt   time.Time `json:"sent_at"` // RFC3339
}

// ------------ pool ------------

var db *pgxpool.Pool

func initDB(dsn string) error {
	var err error
	db, err = pgxpool.New(context.Background(), dsn)
	return err
}

// ------------ insert + notify (atomic) ------------

func storeAndNotify(ctx context.Context, m Message) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // safe even after commit

	_, err = tx.Exec(ctx,
		`INSERT INTO messages (id, room, sender_id, body)
		   VALUES ($1,$2,$3,$4)`,
		m.ID, m.Room, m.SenderID, m.Body)
	log.Println("INSERT: ", m.ID, ", ", m.Room, ", ", m.SenderID, ", ", m.Body)
	if err != nil {
		return err
	}

	// small JSON payload keeps us under 8 kB NOTIFY limit
	payload, _ := json.Marshal(struct {
		Room string    `json:"room"`
		ID   uuid.UUID `json:"id"`
	}{m.Room, m.ID})

	_, err = tx.Exec(ctx, `SELECT pg_notify('chat', $1)`, string(payload))
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// ------------ fetch full row by id ------------

func fetchMessage(ctx context.Context, id uuid.UUID) (Message, error) {
	var m Message
	err := db.QueryRow(ctx,
		`SELECT id, room, sender_id, body, sent_at
		   FROM messages WHERE id=$1`, id).
		Scan(&m.ID, &m.Room, &m.SenderID, &m.Body, &m.SentAt)
	return m, err
}
