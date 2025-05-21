package main

import (
	"log"
	"mininet/database"
	"mininet/handlers"
	"net/http"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	handlers.SetupHelpers("templates/*.gohtml", []byte("very-secret-key"))

	database.InitDB("./mininetDatabase.db")
	defer database.DB().Close()

	handlers.HandlerInit()

	mux := mux.NewRouter()

	// Routes
	mux.PathPrefix("/styles/").Handler(http.StripPrefix("/styles/", http.FileServer(http.Dir("styles"))))
	mux.HandleFunc("/", handlers.IndexHandler)
	mux.HandleFunc("/register", handlers.RegisterHandler)
	mux.HandleFunc("/login", handlers.LoginHandler)
	mux.HandleFunc("/logout", handlers.LogoutHandler)
	mux.HandleFunc("/profile", handlers.ProfilePageHandler).Methods("GET")
	mux.HandleFunc("/profile/{path:.*}", handlers.ProfilePageHandler).Methods("GET")
	mux.HandleFunc("/home", handlers.PageHandler).Methods("GET")
	mux.HandleFunc("/page/{path:.*}", handlers.PageHandler).Methods("GET")
	mux.HandleFunc("/addText/{path:.*}", handlers.AddTextHandler).Methods("POST")
	mux.HandleFunc("/editText/{pageId:[0-9]+}/{textId:[0-9]+}", handlers.EditTextHandler).Methods("GET")
	mux.HandleFunc("/editText/{pageId:[0-9]+}/{textId:[0-9]+}/cancel", handlers.EditTextCancelHandler).Methods("GET")
	mux.HandleFunc("/editText/{pageId:[0-9]+}/{textId:[0-9]+}", handlers.UpdateTextHandler).Methods("PUT")
	mux.HandleFunc("/editText/{pageId:[0-9]+}/{textId:[0-9]+}", handlers.DeleteTextHandler).Methods("DELETE")

	log.Println("Server running at http://localhost:8080")
	http.ListenAndServe(":8080", mux)
}
