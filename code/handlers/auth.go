package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/mail"
	"postpath/database"
	"regexp"
	"strings"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

// Define custom context keys
type contextKey string

const (
	userContextKey   contextKey = "user"
	userIDContextKey contextKey = "userID"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session")

		// Check if user is already authenticated
		if isValidSession(session) {
			username, ok := session.Values["user"].(string)
			if ok && username != "" {
				// If accessing public paths while logged in, redirect to home
				publicPaths := map[string]bool{
					"/login":    true,
					"/register": true,
					"/":         true,
				}

				if publicPaths[r.URL.Path] {
					http.Redirect(w, r, "/home", http.StatusSeeOther)
					return
				}

				// Continue with authenticated context
				userID := getUserId(username)
				ctx := context.WithValue(r.Context(), userContextKey, username)
				ctx = context.WithValue(ctx, userIDContextKey, userID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// Handle non-authenticated requests
		publicPaths := map[string]bool{
			"/login":    true,
			"/register": true,
			"/":         true,
		}

		if publicPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
}

// Helper functions to get user info from context
func GetUserFromContext(r *http.Request) (string, int) {
	ctx := r.Context()
	username, _ := ctx.Value(userContextKey).(string)
	userID, _ := ctx.Value(userIDContextKey).(int)
	return username, userID
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		email := r.FormValue("email")
		password := r.FormValue("password")

		data := map[string]any{
			"Username": username,
			"Email":    email,
		}

		// Basic validations
		if username == "" || email == "" || password == "" {
			data["Error"] = "All fields are required."
			render(w, r, "register", data)
			return
		}

		matched, _ := regexp.MatchString(`^[a-zA-Z0-9]+$`, username)
		if !matched {
			data["Error"] = "Username can only contain letters and numbers."
			render(w, r, "register", data)
			return
		}

		if err := validateEmail(email); err != nil {
			data["Error"] = "Invalid email format."
			render(w, r, "register", data)
			return
		}

		if len(password) < 6 {
			data["Error"] = "Password must be at least 6 characters."
			render(w, r, "register", data)
			return
		}

		// Hash password
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			data["Error"] = "Internal error creating user."
			render(w, r, "register", data)
			return
		}

		// Insert into DB
		_, err = database.DB().Exec(
			"INSERT INTO users (username, email, password) VALUES (?, ?, ?)",
			username, email, hash,
		)

		if err != nil {
			if isUniqueViolation(err) {
				data["Error"] = "Email or username already exists."
			} else {
				data["Error"] = "Error registering user."
			}
			render(w, r, "register", data)
			return
		}

		// On success, do a full-page redirect (not HTMX)
		session, _ := store.Get(r, "session")
		session.AddFlash("Registration successful. You can now log in.")
		session.Save(r, w)

		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	render(w, r, "register", nil)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	flashes := session.Flashes()
	_ = session.Save(r, w)

	data := map[string]any{}
	if len(flashes) > 0 {
		data["Flash"] = flashes[0]
	}

	if r.Method == http.MethodPost {
		email := r.FormValue("email")
		password := r.FormValue("password")

		var id int
		var username, hash string

		err := database.DB().QueryRow("SELECT id, username, password FROM users WHERE email = ?", email).Scan(&id, &username, &hash)
		if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
			render(w, r, "login", map[string]any{
				"Error": "Invalid email or password",
				"Email": email,
			})
			return
		}

		// Set session values
		session.Values["user"] = username
		session.Values["user_id"] = id
		session.Values["authenticated"] = true

		if err := session.Save(r, w); err != nil {
			http.Error(w, "Error saving session", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/home", http.StatusSeeOther)
		return
	}

	render(w, r, "login", data)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	session.Options.MaxAge = -1
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func validateEmail(email string) error {
	if _, err := mail.ParseAddress(email); err != nil {
		return errors.New("invalid email format")
	}
	return nil
}

func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "duplicate")
}

func isValidSession(session *sessions.Session) bool {
	// Check if session exists
	if session == nil {
		return false
	}

	// Check if session has required values
	if session.IsNew {
		return false
	}

	// Check if session has not expired
	if session.Options != nil && session.Options.MaxAge < 0 {
		return false
	}

	return true
}

func getUserId(user string) int {
	// Get User ID
	var userId int
	err := database.DB().QueryRow("SELECT id FROM users WHERE username = ?", user).Scan(&userId)
	if err != nil {
		return -1
	}
	return userId
}
