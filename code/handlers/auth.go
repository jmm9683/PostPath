package handlers

import (
	"errors"
	"mininet/database"
	"net/http"
	"net/mail"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

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
	if getLoggedInUser(r) != "" {
		http.Redirect(w, r, "/home", http.StatusSeeOther)
		return
	}

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
			data := map[string]any{
				"Error": "Invalid email or password",
				"Email": email,
			}
			render(w, r, "login", data)
			return
		}

		session, _ := store.Get(r, "session")
		session.Values["user"] = username
		session.Save(r, w)

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

func ProfileHandler(w http.ResponseWriter, r *http.Request) {
	user := getLoggedInUser(r)
	if user == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	data := map[string]any{
		"Username": user,
		"LoggedIn": user != "",
	}
	render(w, r, "profile", data)
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
