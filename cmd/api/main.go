package main

import (
	"faas-project/internal/api/handlers"
	"faas-project/internal/message"
	"fmt"
	"net/http"
)

func main() {

	nc, err := message.Connect("nats://nats:4222")
	if err != nil {
		fmt.Println(err)
		return
	}
	message.InitNats(nc)

	http.HandleFunc("/", handlers.DefaultHandler)
	http.HandleFunc("/login", handlers.LoginHandler)
	http.HandleFunc("/register", handlers.RegisterHandler)
	http.HandleFunc("/function", handlers.RegisterFunctionHandler)
	http.HandleFunc("/function/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodDelete:
			handlers.DeleteFunctionHandler(w, r)
		case http.MethodPost:
			handlers.ExecuteFunctionHandler(w, r)
		default:
			http.NotFound(w, r)
		}
	})

	fmt.Println("Starting server at port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println(err)
	}
}
