package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
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

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return origin == "http://localhost:8080"
	},
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

type LoginRequest struct {
	Identifier string `json:"identifier"` // email or nickname
	Password   string `json:"password"`
}

func LoginHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Basic input validation
		if req.Identifier == "" || req.Password == "" {
			http.Error(w, "Missing credentials", http.StatusBadRequest)
			return
		}

		// Get user by email or nickname
		userUUID, hashedPassword, err := GetUserByEmailOrNickname(db, req.Identifier)
		if err != nil {
			http.Error(w, "Invalid email/nickname or password", http.StatusUnauthorized)
			return
		}

		// Compare password
		if !CheckPasswordHash(hashedPassword, req.Password) {
			http.Error(w, "Invalid email/nickname or password", http.StatusUnauthorized)
			return
		}

		// Create session UUID and expiry
		sessionUUID := uuid.New().String()
		expiresAt := time.Now().Add(24 * time.Hour)

		// Save session in DB
		err = CreateSession(db, sessionUUID, userUUID, expiresAt)
		if err != nil {
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		// Set cookie with session UUID
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    sessionUUID,
			Expires:  expiresAt,
			HttpOnly: true,
			Path:     "/",
		})

		w.Write([]byte("Login successful"))
	}
}

func LogoutHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			http.Error(w, "No session found", http.StatusUnauthorized)
			return
		}

		sessionToken := cookie.Value

		// Delete session from DB
		err = DeleteSession(db, sessionToken)
		if err != nil {
			http.Error(w, "Error logging out", http.StatusInternalServerError)
			return
		}

		// Expire the session cookie
		expiredCookie := &http.Cookie{
			Name:     "session_token",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		}
		http.SetCookie(w, expiredCookie)

		w.Write([]byte("Logged out successfully"))
	}
}

type CreatePostRequest struct {
	Title      string   `json:"title"`
	Content    string   `json:"content"`
	Categories []string `json:"categories"`
}

func CreatePostHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userUUID, ok := UserUUIDFromContext(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req CreatePostRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		req.Title = strings.TrimSpace(req.Title)
		req.Content = strings.TrimSpace(req.Content)

		if req.Title == "" || req.Content == "" {
			http.Error(w, "Title and content are required", http.StatusBadRequest)
			return
		}

		postUUID := uuid.New().String()
		now := time.Now()

		err := InsertPost(db, postUUID, userUUID, req.Title, req.Content, now)
		if err != nil {
			http.Error(w, "Failed to insert post", http.StatusInternalServerError)
			return
		}

		err = InsertPostCategories(db, postUUID, req.Categories)
		if err != nil {
			http.Error(w, "Failed to insert categories", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("Post created successfully"))
	}
}

func WebSocketHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userUUID, ok := UserUUIDFromContext(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("WebSocket upgrade error:", err)
			return
		}

		client := &Client{
			Conn:     conn,
			UserUUID: userUUID,
			Send:     make(chan []byte),
		}

		clients[userUUID] = client
		onlineUsers[userUUID] = &UserPresence{
			UserUUID:    userUUID,
			IsOnline:    true,
			LastMessage: "",
		}

		go writePump(client)
		readPump(db, client)
	}
}

func GetMessagesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userUUID, ok := UserUUIDFromContext(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		otherUser := r.URL.Query().Get("with")
		offsetStr := r.URL.Query().Get("offset")
		offset, _ := strconv.Atoi(offsetStr)

		messages, err := LoadMessages(db, userUUID, otherUser, 10, offset)
		if err != nil {
			http.Error(w, "Failed to fetch messages", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messages)
	}
}
