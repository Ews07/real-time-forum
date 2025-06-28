package main

import (
	"database/sql"
	"errors"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var ErrUserExists = errors.New("user already exists")

func InitDB(dbFile string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return nil, err
	}

	// Ping to check connection
	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Read schema.sql
	schema, err := os.ReadFile("schema.sql")
	if err != nil {
		return nil, err
	}

	// Execute schema to create table if not exist
	_, err = db.Exec(string(schema))
	if err != nil {
		return nil, err
	}
	return db, nil
}

// Check if email or nickname already exists
func UserExists(db *sql.DB, email, nickname string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = ? OR nickname = ?)`
	err := db.QueryRow(query, email, nickname).Scan(&exists)
	return exists, err
}

/*
  for security we can add:
  	// Only allow "email" or "nickname" as column names
	validColumns := map[string]bool{
		"email": true,
		"nickname": true,
	}
	
	if !validColumns[input] {
		fmt.Println("Invalid column name")
		return false, nil
	}
*/

// Insert user with all fields
func InsertUserFull(db *sql.DB, uuid, nickname, email, passwordHash string, age int, gender, firstName, lastName string) error {
	stmt := `INSERT INTO users (uuid, nickname, email, password_hash, age, gender, first_name, last_name) 
             VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := db.Exec(stmt, uuid, nickname, email, passwordHash, age, gender, firstName, lastName)
	return err
}

// GetUserByEmailOrNickname fetches user with matching email OR nickname
func GetUserByEmailOrNickname(db *sql.DB, identifier string) (uuid, hashedPassword string, err error) {
	query := `SELECT uuid, password_hash FROM users WHERE email = ? OR nickname = ?`
	return getUserAuth(db, query, identifier)
}

func getUserAuth(db *sql.DB, query, id string) (string, string, error) {
	var uuid, hash string
	err := db.QueryRow(query, id, id).Scan(&uuid, &hash)
	return uuid, hash, err
}

// CreateSession inserts a session for a user
func CreateSession(db *sql.DB, sessionUUID, userUUID string, expiresAt time.Time) error {
	stmt := `INSERT INTO sessions (session_uuid, user_uuid, expires_at) VALUES (?, ?, ?)`
	_, err := db.Exec(stmt, sessionUUID, userUUID, expiresAt)
	return err
}

var ErrSessionNotFound = errors.New("session not found or expired")

type Session struct {
	SessionUUID string
	UserUUID string
	ExpiresAt time.Time
}

//GetSession returns session info if session exists and valid
func GetSession(db *sql.DB, sessionUUID string) (*Session, error) {
	var s Session
	query := "SELECT session_uuid, user_uuid, expires_at FROM sessions WHERE session_uuid = ?"
	err := db.QueryRow(query, sessionUUID).Scan(&s.SessionUUID, &s.UserUUID, &s.ExpiresAt)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	if time.Now().After(s.ExpiresAt) {
		return nil, ErrSessionNotFound
	}

	return &s, nil
}

func DeleteSession(db *sql.DB, sessionUUID string) error {
    stmt := "DELETE FROM sessions WHERE session_uuid = ?"
    _, err := db.Exec(stmt, sessionUUID)
    return err
}

func InsertPost(db *sql.DB, postUUID, userUUID, title, content string, createdAt time.Time) error {
    stmt := "INSERT INTO posts (uuid, user_uuid, title, content, created_at) VALUES (?, ?, ?, ?, ?)"
    _, err := db.Exec(stmt, postUUID, userUUID, title, content, createdAt)
    return err
}

func InsertPostCategories(db *sql.DB, postUUID string, categories []string) error {
    stmt := "INSERT INTO post_categories (post_uuid, category) VALUES (?, ?)"
    for _, cat := range categories {
        _, err := db.Exec(stmt, postUUID, cat)
        if err != nil {
            return err
        }
    }
    return nil
}