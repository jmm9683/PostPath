package handlers

import (
	"database/sql"
	"log"
	"mininet/database"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	render(w, r, r.URL.Path, nil)
}

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
	User         string
	CreatedAtStr string
	Edited       int
	SourcePath   string
	Source       int
	SourceTitle  string
}

func PageHandler(w http.ResponseWriter, r *http.Request) {
	user := getLoggedInUser(r)
	if user == "" {
		render(w, r, "landing", nil)
		return
	}

	userId := getUserId(user)
	if userId == -1 {
		http.Error(w, "User not found", http.StatusInternalServerError)
		return
	}

	path := getPath(r)
	pageId := path[len(path)-1]
	var pageTitles []PageTitle

	for _, id := range path {
		var title string
		err := database.DB().QueryRow(`SELECT title FROM pages WHERE id = ?`, id).Scan(&title)
		if err != nil {
			http.Error(w, "Page not found", http.StatusNotFound)
			return
		}

		pageTitles = append(pageTitles, PageTitle{ID: id, Title: title})
	}
	log.Println(pageTitles)
	log.Println("Loading text for " + pageTitles[len(pageTitles)-1].Title)

	rows, err := database.DB().Query(`
		SELECT 
		pagetext.page_id, 
		pagetext.id, 
		pagetext.text, 
		pagetext.link_id, 
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

	var texts []PageText
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var pt PageText
			var linkId sql.NullInt64
			var createdAt time.Time

			if err := rows.Scan(
				&pt.PageID, &pt.TextID, &pt.Text, &linkId,
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
	}

	render(w, r, "home", data)
}

func AddTextHandler(w http.ResponseWriter, r *http.Request) {
	user := getLoggedInUser(r)
	if user == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	userId := getUserId(user)
	if userId == -1 {
		http.Error(w, "User not found", http.StatusInternalServerError)
		return
	}

	text := r.FormValue("text")
	text = strings.TrimSpace(text)
	path := getPath(r)
	source := path[len(path)-1]
	pageId := path[len(path)-1]
	if len(path) > 1 {
		source = path[len(path)-2]
	}
	sourcePath := getSourcePath(r)

	if text == "" {
		http.Error(w, "Text cannot be empty", http.StatusBadRequest)
		return
	}

	lText := strings.ToLower(text)
	log.Printf("Adding text to page ID: %v with sourcePath %s and source %v", pageId, sourcePath, source)

	if len(strings.Fields(text)) == 1 {
		// One word: Handle as a link

		var linkID int
		err := database.DB().QueryRow(`SELECT id FROM pages WHERE title = ?`, lText).Scan(&linkID)
		if err == sql.ErrNoRows {
			// Link does not exist yet, insert it
			result, err := database.DB().Exec(`INSERT INTO pages (title) VALUES (?)`, lText)
			if err != nil {
				http.Error(w, "Failed to insert into pages", http.StatusInternalServerError)
				return
			}
			lastInsertId, err := result.LastInsertId()
			if err != nil {
				http.Error(w, "Failed to get inserted link id", http.StatusInternalServerError)
				return
			}
			linkID = int(lastInsertId)
		} else if err != nil {
			http.Error(w, "Failed to query pages", http.StatusInternalServerError)
			return
		}
		_, err = database.DB().Exec(`INSERT INTO pagetext (page_id, user_id, text, link_id, created_at, path, source) VALUES (?, ?, ?, ?, ?, ?, ?)`, pageId, userId, text, linkID, time.Now(), sourcePath, source)
		if err != nil {
			http.Error(w, "Failed to insert text into pagetext", http.StatusInternalServerError)
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
	user := getLoggedInUser(r)
	if user == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userId := getUserId(user)
	if userId == -1 {
		http.Error(w, "User not found", http.StatusInternalServerError)
		return
	}

	pageId := getPageId(r)
	textId := getTextId(r)
	if textId == -1 {
		http.Error(w, "TextID missing", http.StatusNotFound)
		return
	}

	var text string
	err := database.DB().QueryRow(`SELECT text FROM pagetext WHERE id = ? and user_id = ?`, textId, userId).Scan(&text)
	if err == sql.ErrNoRows {
		http.Error(w, "You are not allowed to edit this text.", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
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
	user := getLoggedInUser(r)
	if user == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userId := getUserId(user)
	if userId == -1 {
		http.Error(w, "User not found", http.StatusInternalServerError)
		return
	}

	pageId := getPageId(r)
	textId := getTextId(r)
	if textId == -1 {
		http.Error(w, "TextID missing", http.StatusNotFound)
		return
	}

	rows, err := database.DB().Query(`
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
	user := getLoggedInUser(r)
	if user == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	pageId := getPageId(r)
	textId := getTextId(r)
	if textId == -1 {
		http.Error(w, "TextID missing", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	text := strings.TrimSpace(r.FormValue("text"))

	if text == "" {
		// If empty, delete the text
		_, err := database.DB().Exec(`DELETE FROM pagetext WHERE id = ? and page_id = ?`, textId, pageId)
		if err != nil {
			http.Error(w, "Failed to delete text", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	_, err := database.DB().Exec(`UPDATE pagetext SET text = ?, is_edited = 1 WHERE id = ? and page_id = ?`, text, textId, pageId)
	if err != nil {
		http.Error(w, "Failed to update text", http.StatusInternalServerError)
		return
	}

	data := map[string]any{"PageID": pageId, "Text": text, "TextID": textId, "User": user, "CreatedAtStr": "Just Now", "Edited": 1}
	render(w, r, "addtext", data)
}

func DeleteTextHandler(w http.ResponseWriter, r *http.Request) {
	user := getLoggedInUser(r)
	if user == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	pageId := getPageId(r)
	textId := getTextId(r)
	if textId == -1 {
		http.Error(w, "TextID missing", http.StatusNotFound)
		return
	}

	_, err := database.DB().Exec(`DELETE FROM pagetext WHERE id = ? and page_id = ?`, textId, pageId)
	if err != nil {
		http.Error(w, "Failed to delete text", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Helper Functions
func getPageId(r *http.Request) int {
	vars := mux.Vars(r)
	idStr := vars["pageId"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		id = 0 //default to home
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
			ids = []int{0}
			return ids
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
