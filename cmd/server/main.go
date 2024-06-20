package main

import (
	"log"

	"github.com/zrma/proglog/pkg/server"
)

func main() {
	svr := server.NewHTTPServer(":8080")
	log.Fatal(svr.ListenAndServe())
}
