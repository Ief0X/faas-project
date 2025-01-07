package main

import (
	"faas-project/internal/api/handlers"
	"fmt"
	"net/http"
	"faas-project/internal/message"
)

func main() {

	nc, err := message.Connect("nats://localhost:4222")
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
