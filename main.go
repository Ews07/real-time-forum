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

	// simple health check
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	}).Methods("GET")

	r.HandleFunc("/register", RegisterHandler(db)).Methods("POST")

	r.HandleFunc("/login", LoginHandler(db)).Methods("POST")

	// Start server
	log.Println("Starting server on http://localhost:8080")
	err = http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
