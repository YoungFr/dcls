package main

import (
	"log"

	"github.com/youngfr/dcls/internal/server"
)

func main() {
	// curl -X POST localhost:8080 -d '{"record": {"value": "TGV0J3MgR28gIzEK"}}'
	// curl -X GET localhost:8080 -d '{"offset": 0}'
	log.Fatal(server.NewHTTPServer(":8080").ListenAndServe())
}
