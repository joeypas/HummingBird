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

// serveWs handles websocket requests from the peer.
func serveWs(w http.ResponseWriter, r *http.Request) {
	room := mux.Vars(r)["room"]
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := newClient(conn, room)
	hub.join(room, client)

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}

func main() {
	if err := initDB(os.Getenv("DATABASE_URL")); err != nil {
		log.Fatal(err)
	}

	go listenAndFan(context.Background(), hub)

	router := mux.NewRouter()
	router.HandleFunc("/ws/{room}", serveWs)
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static")))

	log.Println("listening on ", addr)
	log.Fatal(http.ListenAndServe(*addr, router))
}
