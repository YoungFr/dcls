package main

import (
	"log"

	"github.com/youngfr/dcls/internal/server"
)

func main() {
	log.Fatal(server.NewHTTPServer(":8080").ListenAndServe())
}
