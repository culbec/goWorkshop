package main

import (
	"fmt"
	"net/http"
)

func main() {
	server := &http.Server{
		Addr:    "127.0.0.1:8080", // Change the port as needed
		Handler: http.DefaultServeMux,
	}

	// ping handler
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	// Start the server
	fmt.Println("Server started at: " + server.Addr)
	err := server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
