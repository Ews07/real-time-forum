package main

import (
	"context"
	"database/sql"
	"net/http"
)

// Session Middleware for Authentication
func AuthMiddleware(db *sql.DB, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			http.Error(w, "Unauthorized: missing session token", http.StatusUnauthorized)
			return
		}

		session, err := GetSession(db, cookie.Value)
		if err != nil {
			http.Error(w, "Unauthorized: invalid or expired session", http.StatusUnauthorized)
			return
		}

		// Add user UUID to request context
		ctx := context.WithValue(r.Context(), userContextKey, session.UserUUID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
