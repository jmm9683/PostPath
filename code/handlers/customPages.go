package handlers

import (
	"database/sql"
	"mininet/database"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type ProfileEntry struct {
	Text         string
	IsLink       bool
	LinkID       int
	CreatedAtStr string
}

func ProfilesHandler(w http.ResponseWriter, r *http.Request) {
	loggedInUser := getLoggedInUser(r)

	vars := mux.Vars(r)
	requestedUsername := vars["username"]

	if requestedUsername == "" {
		http.Error(w, "Username not provided", http.StatusBadRequest)
		return
	}

	var userId int
	err := database.DB().QueryRow(
		"SELECT id FROM users WHERE username = ?", requestedUsername,
	).Scan(&userId)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	rows, err := database.DB().Query(`
		SELECT text, is_link, link_id, created_at
		FROM profiles
		WHERE user_id = ?
		ORDER BY created_at ASC
	`, userId)
	if err != nil {
		http.Error(w, "Error loading profile data", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var profileEntries []ProfileEntry
	for rows.Next() {
		var entry ProfileEntry
		var createdAt time.Time
		var linkId sql.NullInt64

		if err := rows.Scan(&entry.Text, &entry.IsLink, &linkId, &createdAt); err == nil {
			if linkId.Valid {
				entry.LinkID = int(linkId.Int64)
			}
			entry.CreatedAtStr = createdAt.Format("2006-01-02 15:04")
			profileEntries = append(profileEntries, entry)
		}
	}

	data := map[string]any{
		"Username": requestedUsername,
		"Editable": loggedInUser == requestedUsername,
		"Entries":  profileEntries,
		"LoggedIn": loggedInUser != "",
	}

	render(w, r, "profile", data)
}
