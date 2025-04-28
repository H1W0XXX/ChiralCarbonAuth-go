// File: main.go
package main

import (
	"log"
	"net/http"
)

func main() {
	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/api/challenge/verify", handleVerify)
	http.HandleFunc("/api/challenge/start", handleStart)

	log.Println("Server listening on :28416")
	log.Fatal(http.ListenAndServe(":28416", nil))
}
