package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	db, err := InitDB("forum.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	defer db.Close()

	r := mux.NewRouter()

	r.HandleFunc("/register", RegisterHandler(db)).Methods("POST")
	r.HandleFunc("/login", LoginHandler(db)).Methods("POST")
	r.Handle("/logout", AuthMiddleware(db, LogoutHandler(db))).Methods("POST")
	r.HandleFunc("/feed", PostFeedHandler(db)).Methods("GET")
	r.Handle("/posts", AuthMiddleware(db, CreatePostHandler(db))).Methods("POST")
	r.Handle("/ws", AuthMiddleware(db, WebSocketHandler(db))).Methods("GET")
	r.Handle("/messages", AuthMiddleware(db, GetMessagesHandler(db))).Methods("GET")

	go handleMessages()
	// Start server
	log.Println("Starting server on http://localhost:8080")
	err = http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
