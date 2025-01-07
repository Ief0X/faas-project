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

	fmt.Println("Starting server at port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println(err)
	}
}
