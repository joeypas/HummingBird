package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/gofrs/uuid/v5"
	"github.com/gorilla/mux"
)

var addr = flag.String("addr", ":8080", "http service address")

func serveWs(w http.ResponseWriter, r *http.Request) {
	room_name := mux.Vars(r)["room"]
	auth := r.URL.Query().Get("token")
	uid, err := parseToken(auth)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	room_id, err := uuid.FromString(room_name)
	if err != nil {
		log.Println(err)
		return
	}
	ctx := context.Background()
	room, err := fetchRoomByID(ctx, room_id)
	if err != nil {
		log.Println(err)
		return
	}

	header := http.Header{}

	header.Add("room_name", room.Name)

	conn, err := upgrader.Upgrade(w, r, header)
	if err != nil {
		log.Println(err)
		return
	}

	client := newClient(conn, room.ID, uid)
	hub.join(room.ID, client)

	go client.writePump()
	go client.readPump()
}

func getRooms(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	rooms, err := fetchAllRooms(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	json.NewEncoder(w).Encode(map[string][]Room{"rooms": rooms})
}

func main() {
	if err := initDB(os.Getenv("DATABASE_URL")); err != nil {
		log.Fatal(err)
	}

	go listenAndFan(context.Background(), hub)

	router := mux.NewRouter()
	router.HandleFunc("/register", registerHandler).Methods("POST")
	router.HandleFunc("/login", loginHandler).Methods("POST")
	router.HandleFunc("/ws/{room}", serveWs)
	router.HandleFunc("/rooms", getRooms).Methods("GET")
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static")))

	log.Println("listening on ", addr)
	log.Fatal(http.ListenAndServe(*addr, router))
}
