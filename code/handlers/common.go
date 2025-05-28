package handlers

import (
	"html/template"
	"log"
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
		MaxAge:   3600 * 24, // 1 day
		HttpOnly: true,
		Secure:   false, // Changed to true for HTTPS
		SameSite: http.SameSiteStrictMode,
	}
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	render(w, r, "landing", nil)
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
