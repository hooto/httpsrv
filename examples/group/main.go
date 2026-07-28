// Example: groups, path params, static files, and template rendering.
//
// This example reads ./views and ./static from the working directory, so run it
// from its own folder:
//
//	( cd examples/mod && go run . )
//
// Try:
//
//	curl localhost:3000/demo/index
//	curl localhost:3000/demo/hello/httpsrv
//	curl localhost:3000/demo/hello/template
//	curl localhost:3000/static/text.txt
package main

import (
	"log"

	"github.com/hooto/httpsrv/v2"
	"github.com/hooto/httpsrv/v2/middleware/static"
)

func main() {
	// Parse every .html/.tpl under ./views once; Ctx.Render looks them up by name.
	r, err := httpsrv.TemplatesDir("./views", nil)
	if err != nil {
		log.Fatal(err)
	}

	app := httpsrv.New(httpsrv.WithViews(r))

	// Static files under /static: a FileServer registered on a catch-all route.
	// Directory listing is disabled; explicit routes always take priority over
	// the catch-all. Use app.All to serve every HTTP method.
	app.Get("/static/{*path}", static.New("./static"))

	// A Group shares a URL prefix.
	demo := app.Group("/demo")

	demo.Get("/index", func(c httpsrv.Ctx) error {
		return c.SendString("mod: index")
	})

	// {name} is a path param. (A literal segment like "template" below wins
	// over the param at the same position, regardless of registration order.)
	demo.Get("/hello/{name}", func(c httpsrv.Ctx) error {
		return c.SendString("hello " + c.Params("name"))
	})

	// Render views/hello/template.tpl with bind data.
	demo.Get("/hello/template", func(c httpsrv.Ctx) error {
		return c.Render("hello/template.tpl", map[string]string{"Name": "httpsrv"})
	})

	log.Fatal(app.Run(":3000"))
}
