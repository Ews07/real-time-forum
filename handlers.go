package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type RegisterRequest struct {
	Nickname  string `json:"nickname"`
	Age       int    `json:"age"`
	Gender    string `json:"gender"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

func RegisterHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RegisterRequest

		// Decode JSON body
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Basic validation
		req.Nickname = strings.TrimSpace(req.Nickname)
		req.Email = strings.TrimSpace(req.Email)
		req.Password = strings.TrimSpace(req.Password)

		if req.Nickname == "" || req.Email == "" || req.Password == "" || req.Age <= 0 {
			http.Error(w, "Missing required fields", http.StatusBadRequest)
			return
		}

		// Check if user already exists
		exists, err := UserExists(db, req.Email, req.Nickname)
		if err != nil {
			log.Printf("DB error checking user existence: %v", err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}
		if exists {
			http.Error(w, "Email or Nickname already taken", http.StatusConflict)
			return
		}

		// Hash password
		hashedPass, err := HashPassword(req.Password)
		if err != nil {
			log.Printf("Error hashing password: %v", err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		// Create UUID for user
		userUUID := uuid.New().String()

		// Insert user in DB
		err = InsertUserFull(db, userUUID, req.Nickname, req.Email, hashedPass, req.Age, req.Gender, req.FirstName, req.LastName)
		if err != nil {
			log.Printf("Error inserting user: %v", err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("User registered successfully"))
	}
}
