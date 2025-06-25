package main

import (
	"context"
	"encoding/json"
	//"log"
	"time"

	"github.com/gofrs/uuid/v5"
	pgxuuid "github.com/jackc/pgx-gofrs-uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Message struct {
	ID       uuid.UUID `json:"id"`
	Room     string    `json:"room"`
	SenderID uuid.UUID `json:"sender_id"`
	Username string    `json:"username"`
	Body     string    `json:"body"`
	SentAt   time.Time `json:"sent_at"` // RFC3339
}

type User struct {
	id       uuid.UUID
	email    string
	username string
	hashed   string
}

// ------------ pool ------------

var db *pgxpool.Pool

func initDB(dsn string) error {
	var err error
	dbconf, err := pgxpool.ParseConfig(dsn)
	dbconf.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		pgxuuid.Register(conn.TypeMap())
		return nil
	}
	db, err = pgxpool.NewWithConfig(context.Background(), dbconf)

	return err
}

// ------------ users ------------

func CreateUser(email, username, password string) *User {
	hashed, err := hashPassword(password)
	if err != nil {
		return nil
	}
	uid, err := uuid.NewV4()
	if err != nil {
		return nil
	}
	user := User{
		id:       uid,
		email:    email,
		username: username,
		hashed:   hashed,
	}
	err = storeUser(context.Background(), user)
	if err != nil {
		return nil
	}

	return &user
}

func hashPassword(password string) (string, error) {
	var passwordBytes = []byte(password)

	hashedPasswordBytes, err := bcrypt.GenerateFromPassword(passwordBytes, bcrypt.MinCost)

	return string(hashedPasswordBytes), err
}

func verifyCredentials(email, password string) (*User, error) {
	user, err := fetchUserByEmail(context.Background(), email)
	if err != nil {
		return nil, err
	}
	err = bcrypt.CompareHashAndPassword(
		[]byte(user.hashed), []byte(password),
	)
	if err != nil {
		return nil, err
	}

	return &user, err
}

func storeUser(ctx context.Context, u User) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO users (id, email, username, hashed) VALUES ($1,$2,$3,$4)`,
		u.id, u.email, u.username, u.hashed)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func fetchUserByID(ctx context.Context, id uuid.UUID) (User, error) {
	var u User
	err := db.QueryRow(ctx,
		`SELECT id, email, username, hashed FROM users WHERE id=$1`,
		id).Scan(&u.id, &u.email, &u.username, &u.hashed)
	return u, err
}

func fetchUserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := db.QueryRow(ctx,
		`SELECT id, email, username, hashed FROM users WHERE email=$1`,
		email).Scan(&u.id, &u.email, &u.username, &u.hashed)
	return u, err
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
	if err != nil {
		return m, err
	}
	err = db.QueryRow(ctx,
		`SELECT username FROM users WHERE id = $1::uuid`, m.SenderID).
		Scan(&m.Username)
	return m, err
}
