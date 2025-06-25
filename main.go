package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

var addr = flag.String("addr", ":8080", "http service address")

func serveWs(w http.ResponseWriter, r *http.Request) {
	room := mux.Vars(r)["room"]
	auth := r.URL.Query().Get("token")
	uid, err := parseToken(auth)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := newClient(conn, room, uid)
	hub.join(room, client)

	go client.writePump()
	go client.readPump()
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
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static")))

	log.Println("listening on ", addr)
	log.Fatal(http.ListenAndServe(*addr, router))
}
