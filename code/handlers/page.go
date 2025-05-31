package handlers

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"postpath/database"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

const (
	MaxInputLength = 500
)

var HomePageID, ProfilePageID int

type PageTitle struct {
	ID    int
	Title string
}

type PageText struct {
	PageID       int
	TextID       int
	Text         string
	LinkID       int
	Path         []int
	UserID       int
	User         string
	CreatedAtStr string
	Edited       int
	SourcePath   string
	Source       int
	SourceTitle  string
}

func HandlerInit() {
	rows, cancel, err := database.QueryWithTimeout(`SELECT id, title FROM pages WHERE title IN (?, ?)`, "Home", "Profile")
	if err != nil {
		log.Printf("Query error: %v", err)
		log.Fatalf("failed to query page IDs: %v", err)
	}
	defer cancel()
	defer rows.Close()

	foundHome := false
	foundProfile := false

	for rows.Next() {
		var id int
		var title string
		if err := rows.Scan(&id, &title); err != nil {
			log.Printf("Scan error: %v", err)
			log.Fatalf("failed to scan row: %v", err)
		}
		switch title {
		case "Home":
			HomePageID = id
			foundHome = true
		case "Profile":
			ProfilePageID = id
			foundProfile = true
		}
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		log.Printf("Row iteration error: %v", err)
		log.Fatalf("error during row iteration: %v", err)
	}

	// Verify we found both pages
	if !foundHome || !foundProfile {
		log.Fatalf("missing required pages: home=%d, profile=%d", HomePageID, ProfilePageID)
	}

	log.Printf("HandlerInit complete: HomePageID=%d, ProfilePageID=%d", HomePageID, ProfilePageID)
}

func PageHandler(w http.ResponseWriter, r *http.Request) {
	user, userId := GetUserFromContext(r)

	path := getPath(r)
	if path == nil {
		path = []int{HomePageID}
	}
	if path[len(path)-1] == ProfilePageID {
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}
	renderPage(w, r, path, user, userId, true, false)
}

func ProfilePageHandler(w http.ResponseWriter, r *http.Request) {
	user, userId := GetUserFromContext(r)

	path := getPath(r)
	var profileId int
	if path == nil {
		profileId = userId
	} else {
		profileId = path[len(path)-1]
		user = getUsername(profileId)
	}
	renderPage(w, r, []int{ProfilePageID}, user, profileId, userId == profileId, true)
}

func AddTextHandler(w http.ResponseWriter, r *http.Request) {
	user, userId := GetUserFromContext(r)

	if err := r.ParseForm(); err != nil {
		htmxError(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	text := r.FormValue("text")
	text = strings.TrimSpace(text)

	// Add length validation
	if len(text) > MaxInputLength {
		htmxError(w, "Text exceeds maximum length of 500 characters", http.StatusBadRequest)
		return
	}

	path := getPath(r)
	source := path[len(path)-1]
	pageId := path[len(path)-1]
	if len(path) > 1 {
		source = path[len(path)-2]
	}
	sourcePath := getSourcePath(r)

	if text == "" {
		htmxError(w, "Text cannot be empty", http.StatusBadRequest)
		return
	}

	lText := strings.ToLower(text)
	log.Printf("Adding text to page ID: %v with sourcePath %s and source %v", pageId, sourcePath, source)

	if len(strings.Fields(text)) == 1 && len(path) > 0 && path[0] != ProfilePageID {
		// One word: Handle as a link
		text = lText
		var linkID int
		row, cancel := database.QueryRowWithTimeout(`SELECT id FROM pages WHERE title = ?`, lText)
		defer cancel()
		err := row.Scan(&linkID)
		if err == sql.ErrNoRows {
			// Link does not exist yet, insert it
			result, err := database.DB().Exec(`INSERT INTO pages (title) VALUES (?)`, lText)
			if err != nil {
				htmxError(w, "Failed to insert into pages", http.StatusInternalServerError)
				return
			}
			lastInsertId, err := result.LastInsertId()
			if err != nil {
				htmxError(w, "Failed to get inserted link id", http.StatusInternalServerError)
				return
			}
			linkID = int(lastInsertId)
		} else if err != nil {
			htmxError(w, "Failed to query pages", http.StatusInternalServerError)
			return
		}
		_, err = database.DB().Exec(`INSERT INTO pagetext (page_id, user_id, text, link_id, created_at, path, source) VALUES (?, ?, ?, ?, ?, ?, ?)`, pageId, userId, text, linkID, time.Now(), sourcePath, source)
		if err != nil {
			htmxError(w, "Failed to insert text into pagetext", http.StatusInternalServerError)
			return
		}

		data := map[string]any{"PageID": pageId, "Text": text, "LinkID": linkID, "Path": path, "User": user, "CreatedAtStr": time.Now().Format("2006-01-02 15:04")}
		render(w, r, "addlink", data)
	} else {
		// More than one word: Normal text

		result, err := database.DB().Exec(`INSERT INTO pagetext (page_id, user_id, text, link_id, created_at, path, source) VALUES (?, ?, ?, NULL, ?, ?, ?)`, pageId, userId, text, time.Now(), sourcePath, source)
		if err != nil {
			http.Error(w, "Failed to insert text", http.StatusInternalServerError)
			return
		}
		textId, err := result.LastInsertId()
		if err != nil {
			http.Error(w, "Failed to get inserted link id", http.StatusInternalServerError)
			return
		}
		data := map[string]any{"PageID": pageId, "Text": text, "TextID": textId, "User": user, "CreatedAtStr": time.Now().Format("2006-01-02 15:04"), "Edited": 0}
		render(w, r, "addtext", data)
	}
}

func EditTextHandler(w http.ResponseWriter, r *http.Request) {
	_, userId := GetUserFromContext(r)

	pageId := getPageId(r)
	textId := getTextId(r)
	if textId == -1 {
		htmxError(w, "TextID missing", http.StatusNotFound)
		return
	}

	var text string
	row, cancel := database.QueryRowWithTimeout(`SELECT text FROM pagetext WHERE id = ? and user_id = ?`, textId, userId)
	defer cancel()
	err := row.Scan(&text)
	if err == sql.ErrNoRows {
		htmxError(w, "You are not allowed to edit this text.", http.StatusNotFound)
		return
	} else if err != nil {
		htmxError(w, "Database error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"PageID": pageId,
		"TextID": textId,
		"Text":   text,
	}
	render(w, r, "edittext", data)
}

func EditTextCancelHandler(w http.ResponseWriter, r *http.Request) {
	pageId := getPageId(r)
	textId := getTextId(r)
	if textId == -1 {
		htmxError(w, "TextID missing", http.StatusNotFound)
		return
	}

	rows, cancel, err := database.QueryWithTimeout(`
		SELECT 
			pagetext.page_id, 
			pagetext.id, 
			pagetext.text, 
			users.username, 
			pagetext.created_at,
			pagetext.is_edited,
			pagetext.path,
			pagetext.source,
			pages.title AS source_title
		FROM pagetext
		INNER JOIN users ON pagetext.user_id = users.id
		LEFT JOIN pages ON pagetext.source = pages.id
		WHERE pagetext.page_id = ? AND pagetext.id = ?
		LIMIT 1
	`, pageId, textId)

	var text PageText
	if err == nil {
		defer cancel()
		defer rows.Close()
		for rows.Next() {
			var pt PageText
			var createdAt time.Time

			if err := rows.Scan(
				&pt.PageID, &pt.TextID, &pt.Text,
				&pt.User, &createdAt, &pt.Edited, &pt.SourcePath, &pt.Source, &pt.SourceTitle,
			); err == nil {
				pt.CreatedAtStr = createdAt.Format("2006-01-02 15:04")
				text = pt
				log.Println(text)
			}
		}
	}

	data := map[string]any{
		"PageID":       pageId,
		"TextID":       textId,
		"Text":         text.Text,
		"User":         text.User,
		"CreatedAtStr": text.CreatedAtStr,
		"Edited":       text.Edited,
		"Source":       text.Source,
		"SourcePath":   text.SourcePath,
		"SourceTitle":  text.SourceTitle,
	}
	render(w, r, "addtext", data)
}

func UpdateTextHandler(w http.ResponseWriter, r *http.Request) {
	user, _ := GetUserFromContext(r)

	pageId := getPageId(r)
	textId := getTextId(r)
	if textId == -1 {
		htmxError(w, "TextID missing", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		htmxError(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	text := strings.TrimSpace(r.FormValue("text"))

	// Add length validation
	if len(text) > MaxInputLength {
		htmxError(w, "Text exceeds maximum length of 500 characters", http.StatusBadRequest)
		return
	}

	if text == "" {
		// If empty, delete the text
		_, err := database.DB().Exec(`DELETE FROM pagetext WHERE id = ? and page_id = ?`, textId, pageId)
		if err != nil {
			htmxError(w, "Failed to delete text", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	_, err := database.DB().Exec(`UPDATE pagetext SET text = ?, is_edited = 1 WHERE id = ? and page_id = ?`, text, textId, pageId)
	if err != nil {
		htmxError(w, "Failed to update text", http.StatusInternalServerError)
		return
	}

	data := map[string]any{"PageID": pageId, "Text": text, "TextID": textId, "User": user, "CreatedAtStr": "Just Now", "Edited": 1}
	render(w, r, "addtext", data)
}

func DeleteTextHandler(w http.ResponseWriter, r *http.Request) {
	pageId := getPageId(r)
	textId := getTextId(r)
	if textId == -1 {
		htmxError(w, "TextID missing", http.StatusNotFound)
		return
	}

	_, err := database.DB().Exec(`DELETE FROM pagetext WHERE id = ? and page_id = ?`, textId, pageId)
	if err != nil {
		htmxError(w, "Failed to delete text", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Helper Functions
func renderPage(w http.ResponseWriter, r *http.Request, path []int, user string, userId int, editable bool, filtered bool) {
	pageId := path[len(path)-1]
	var pageTitles []PageTitle

	for _, id := range path {
		var title string
		row, cancel := database.QueryRowWithTimeout(`SELECT title FROM pages WHERE id = ?`, id)
		defer cancel()

		err := row.Scan(&title)
		if err != nil {
			http.Redirect(w, r, "/home", http.StatusSeeOther)
			return
		}

		pageTitles = append(pageTitles, PageTitle{ID: id, Title: title})
	}
	log.Println(pageTitles)
	log.Println("Loading text for " + pageTitles[len(pageTitles)-1].Title)
	var rows *sql.Rows
	var cancel context.CancelFunc
	var err error
	if !filtered {
		rows, cancel, err = database.QueryWithTimeout(`
		SELECT 
			pagetext.page_id, 
			pagetext.id, 
			pagetext.text, 
			pagetext.link_id, 
			users.id,
			users.username, 
			pagetext.created_at,
			pagetext.is_edited,
			pagetext.path,
			pagetext.source,
			pages.title AS source_title
		FROM pagetext
		INNER JOIN users ON pagetext.user_id = users.id
		LEFT JOIN pages ON pagetext.source = pages.id
		WHERE pagetext.page_id = ?
		ORDER BY pagetext.created_at ASC
		`, pageId)
	} else {
		rows, cancel, err = database.QueryWithTimeout(`
		SELECT 
			pagetext.page_id, 
			pagetext.id, 
			pagetext.text, 
			pagetext.link_id, 
			users.id,
			users.username, 
			pagetext.created_at,
			pagetext.is_edited,
			pagetext.path,
			pagetext.source,
			pages.title AS source_title
		FROM pagetext
		INNER JOIN users ON pagetext.user_id = users.id
		LEFT JOIN pages ON pagetext.source = pages.id
		WHERE pagetext.page_id = ? and pagetext.user_id = ?
		ORDER BY pagetext.created_at ASC
		`, pageId, userId)
	}
	var texts []PageText
	if err == nil {
		defer cancel()
		defer rows.Close()
		for rows.Next() {
			var pt PageText
			var linkId sql.NullInt64
			var createdAt time.Time

			if err := rows.Scan(
				&pt.PageID, &pt.TextID, &pt.Text, &linkId, &pt.UserID,
				&pt.User, &createdAt, &pt.Edited, &pt.SourcePath, &pt.Source, &pt.SourceTitle,
			); err == nil {
				if linkId.Valid {
					pt.LinkID = int(linkId.Int64)
				}
				pt.Path = path
				pt.CreatedAtStr = createdAt.Format("2006-01-02 15:04")
				texts = append(texts, pt)
			}
		}
	}

	data := map[string]any{
		"Username":   user,
		"LoggedIn":   user != "",
		"PageID":     pageId,
		"PageTitles": pageTitles,
		"Texts":      texts,
		"Editable":   editable,
	}

	render(w, r, "home", data)

}

func getPageId(r *http.Request) int {
	vars := mux.Vars(r)
	idStr := vars["pageId"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		id = HomePageID //default to home
	}
	return id
}

func getTextId(r *http.Request) int {
	vars := mux.Vars(r)
	idStr := vars["textId"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		id = -1
	}
	return id
}

func getPath(r *http.Request) []int {
	vars := mux.Vars(r)
	rawPath := vars["path"]
	segments := strings.Split(rawPath, "/")

	var ids []int
	for _, s := range segments {
		id, err := strconv.Atoi(s)
		if err != nil {
			return nil
		}
		ids = append(ids, id)
	}
	return ids
}

func getSourcePath(r *http.Request) string {
	vars := mux.Vars(r)
	rawPath := vars["path"]
	segments := strings.Split(rawPath, "/")

	if len(segments) <= 1 {
		return rawPath
	}

	// Join all but the last segment
	return strings.Join(segments[:len(segments)-1], "/")
}

func getUsername(userId int) string {
	var username string

	row, cancel := database.QueryRowWithTimeout("SELECT username FROM users WHERE id = ?", userId)
	defer cancel()

	err := row.Scan(&username)
	if err != nil {
		return ""
	}

	return username
}

func htmxError(w http.ResponseWriter, message string, code int) {
	w.Header().Add("HX-Reswap", "none")
	http.Error(w, message, code)
}
