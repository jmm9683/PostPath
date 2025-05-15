package handlers

import (
	"html/template"
	"log"
	"mininet/database"
	"net/http"

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
		Secure:   false,
	}
}

func getLoggedInUser(r *http.Request) string {
	session, _ := store.Get(r, "session")
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
