// Example: the smallest possible v2 app.
//
// Run:
//
//	go run ./examples/hello
//
// Try:
//
//	curl localhost:3000/demo/hello/world
package main

import (
	"log"

	"github.com/hooto/httpsrv/v2"
)

func main() {
	app := httpsrv.New()

	app.Get("/demo/hello/world", func(c httpsrv.Ctx) error {
		return c.SendString("hello world")
	})

	log.Fatal(app.Run(":3000"))
}
