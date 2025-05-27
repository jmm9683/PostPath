package handlers

import (
	"html/template"
	"log"
	"mininet/database"
	"net/http"
	"path"

	"github.com/gorilla/sessions"
)

var (
	tpl   *template.Template
	store *sessions.CookieStore
)

func SetupHelpers(templateGlob string, sessionKey []byte) {
	tpl = template.Must(template.ParseGlob(templateGlob))
	store = sessions.NewCookieStore(sessionKey)
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		Secure:   false, // Changed to true for HTTPS
		SameSite: http.SameSiteStrictMode,
	}
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	if isRootPath(r) {
		http.Redirect(w, r, "/home", http.StatusSeeOther)
		return
	}
	render(w, r, r.URL.Path, nil)
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

func getLoggedInUser(r *http.Request) string {
	session, err := store.Get(r, "session")
	if err != nil {
		return ""
	}

	if !isValidSession(session) {
		return ""
	}

	user, ok := session.Values["user"].(string)
	if ok && user != "" {
		return user
	}
	return ""
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

func getUsername(userId int) string {
	// Get User ID
	var user string
	err := database.DB().QueryRow("SELECT username FROM users WHERE id = ?", userId).Scan(&user)
	if err != nil {
		return ""
	}
	return user
}

func render(w http.ResponseWriter, r *http.Request, page string, data map[string]any) {
	tplName := page
	if data == nil {
		data = map[string]any{}
	}
	if isHTMX(r) {
		tplName += "HTMX"
	}
	log.Println("Rendering " + tplName)
	if tpl.Lookup(tplName) != nil {
		tpl.ExecuteTemplate(w, tplName, data)
	} else if !isHTMX(r) {
		http.Redirect(w, r, "/home", http.StatusSeeOther)
	}
}

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func isRootPath(r *http.Request) bool {
	return path.Clean(r.URL.Path) == "/"
}
