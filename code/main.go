package main

import (
	"crypto/rand"
	"log"
	"net/http"
	"postpath/database"
	"postpath/handlers"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Generate a proper key for production
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		log.Fatal(err)
	}

	handlers.SetupHelpers("templates/*.gohtml", key)

	database.InitDB("./postpath.db")
	defer database.DB().Close()

	handlers.HandlerInit()

	mux := mux.NewRouter()

	// Static files
	fs := http.FileServer(http.Dir("static"))
	mux.PathPrefix("/styles/").Handler(http.StripPrefix("/styles/", fs))
	mux.PathPrefix("/images/").Handler(http.StripPrefix("/images/", fs))

	// Protected routes
	protected := mux.NewRoute().Subrouter()
	protected.Use(handlers.AuthMiddleware)
	protected.HandleFunc("/", handlers.IndexHandler)
	protected.HandleFunc("/register", handlers.RegisterHandler)
	protected.HandleFunc("/login", handlers.LoginHandler)
	protected.HandleFunc("/logout", handlers.LogoutHandler)
	protected.HandleFunc("/profile", handlers.ProfilePageHandler).Methods("GET")
	protected.HandleFunc("/profile/{path:.*}", handlers.ProfilePageHandler).Methods("GET")
	protected.HandleFunc("/home", handlers.PageHandler).Methods("GET")
	protected.HandleFunc("/page/{path:.*}", handlers.PageHandler).Methods("GET")
	protected.HandleFunc("/addText/{path:.*}", handlers.AddTextHandler).Methods("POST")
	protected.HandleFunc("/editText/{pageId:[0-9]+}/{textId:[0-9]+}", handlers.EditTextHandler).Methods("GET")
	protected.HandleFunc("/editText/{pageId:[0-9]+}/{textId:[0-9]+}/cancel", handlers.EditTextCancelHandler).Methods("GET")
	protected.HandleFunc("/editText/{pageId:[0-9]+}/{textId:[0-9]+}", handlers.UpdateTextHandler).Methods("PUT")
	protected.HandleFunc("/editText/{pageId:[0-9]+}/{textId:[0-9]+}", handlers.DeleteTextHandler).Methods("DELETE")

	// Handle 404
	mux.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/", http.StatusFound)
	})

	log.Println("Server running at http://localhost:8080")
	http.ListenAndServe(":8080", mux)
}
