package main

import (
	"context"
	"encoding/json"
	"log"

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
	RoomID   uuid.UUID `json:"room"`
	SenderID uuid.UUID `json:"sender_id"`
	Username string    `json:"username"`
	Body     string    `json:"body"`
	SentAt   time.Time `json:"sent_at"`
}

type User struct {
	id       uuid.UUID
	email    string
	username string
	hashed   string
	rooms    []Room
}

type Room struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
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

// ------------ rooms ------------

func createRoom(ctx context.Context, name string) (Room, error) {
	var room Room
	uid, err := uuid.NewV4()
	if err != nil {
		return room, err
	}
	room = Room{
		ID:   uid,
		Name: name,
	}
	err = storeRoom(ctx, room)

	return room, err
}

func storeRoom(ctx context.Context, r Room) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO rooms (id, name) VALUES ($1,$2)`,
		r.ID, r.Name)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func fetchRoomByID(ctx context.Context, id uuid.UUID) (Room, error) {
	var r Room
	err := db.QueryRow(ctx,
		`SELECT id, name FROM rooms WHERE id=$1::uuid`,
		id).Scan(&r.ID, &r.Name)
	return r, err
}

func fetchAllRooms(ctx context.Context) ([]Room, error) {
	var rooms []Room

	rows, err := db.Query(ctx, "SELECT id::uuid, name FROM rooms WHERE private=false")
	if err != nil {
		log.Fatal("No rows?")
		return rooms, err
	}

	for rows.Next() {
		var room Room
		err := rows.Scan(&room.ID, &room.Name)
		if err != nil {
			return rooms, err
		}
		rooms = append(rooms, room)
	}

	return rooms, err
}

// ------------ users ------------

func createUser(ctx context.Context, email, username, password string) (User, error) {
	var user User
	hashed, err := hashPassword(password)
	if err != nil {
		return user, err
	}
	uid, err := uuid.NewV4()
	if err != nil {
		return user, err
	}
	user = User{
		id:       uid,
		email:    email,
		username: username,
		hashed:   hashed,
		rooms:    make([]Room, 0),
	}
	err = storeUser(ctx, user)

	return user, err
}

func (u User) addRoom(ctx context.Context, room_id uuid.UUID) error {
	var room Room
	err := db.QueryRow(ctx, `SELECT id, name FROM rows WHERE id=$`, room_id).Scan(&room.ID, &room.Name)
	if err != nil {
		return err
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO user_rows (user_id, room_id)
		   VALUES ($1,$2)`,
		u.id, room.ID)
	if err != nil {
		return err
	}

	u.rooms = append(u.rooms, room)
	return tx.Commit(ctx)
}

func hashPassword(password string) (string, error) {
	var passwordBytes = []byte(password)

	hashedPasswordBytes, err := bcrypt.GenerateFromPassword(passwordBytes, bcrypt.MinCost)

	return string(hashedPasswordBytes), err
}

func verifyCredentials(ctx context.Context, email, password string) (User, error) {
	user, err := fetchUserByEmail(ctx, email)
	if err != nil {
		return user, err
	}
	err = bcrypt.CompareHashAndPassword(
		[]byte(user.hashed), []byte(password),
	)
	if err != nil {
		return user, err
	}

	return user, err
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
	if err != nil {
		return u, err
	}

	rows, err := db.Query(ctx, `SELECT room_id FROM user_rows WHERE user_id=$1`, u.id)
	u.rooms, err = pgx.CollectRows(rows, pgx.RowTo[Room])
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
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO messages (id, room_id, sender_id, body)
		   VALUES ($1,$2,$3,$4)`,
		m.ID, m.RoomID, m.SenderID, m.Body)
	if err != nil {
		return err
	}

	payload, _ := json.Marshal(struct {
		Room uuid.UUID `json:"room"`
		ID   uuid.UUID `json:"id"`
	}{m.RoomID, m.ID})

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
		`SELECT id, room_id, sender_id, body, sent_at
		   FROM messages WHERE id=$1`, id).
		Scan(&m.ID, &m.RoomID, &m.SenderID, &m.Body, &m.SentAt)
	if err != nil {
		return m, err
	}
	err = db.QueryRow(ctx,
		`SELECT username FROM users WHERE id = $1::uuid`, m.SenderID).
		Scan(&m.Username)
	return m, err
}
