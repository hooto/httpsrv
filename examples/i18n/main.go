// Example: loading i18n messages from local JSON files.
//
// Run:
//
//	go run ./examples/i18n
//
// Try:
//
//	curl -H 'Accept-Language: en' localhost:8080/   # Hello, World!
//	curl -H 'Accept-Language: zh' localhost:8080/   # 你好，World！
//	curl -H 'Accept-Language: de' localhost:8080/   # falls back to en
package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"strings"

	"github.com/hooto/httpsrv/v2"
)

// locales/*.json are bundled into the binary; each file is one locale, named by
// its language tag (en.json, zh.json, ...). To load files from disk at runtime
// instead, replace embed.FS with os.ReadFile("locales/en.json"), etc.
//
//go:embed locales/*.json
var localeFS embed.FS

func main() {
	i := httpsrv.NewI18n("en") // default / fallback locale

	// Load each locales/<locale>.json; the locale is the file name stem.
	entries, err := fs.ReadDir(localeFS, "locales")
	if err != nil {
		log.Fatalf("read locales: %v", err)
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		locale := strings.TrimSuffix(name, ".json")
		data, err := localeFS.ReadFile("locales/" + name)
		if err != nil {
			log.Fatalf("read %s: %v", name, err)
		}
		if err := i.LoadJSON(locale, data); err != nil {
			log.Fatalf("load %s: %v", name, err)
		}
		log.Printf("loaded locale %q", locale)
	}

	app := httpsrv.New(httpsrv.WithI18n(i))

	// Detect locale from the Accept-Language header (en default, zh supported).
	app.Use(httpsrv.AcceptLanguage("en", "zh"))

	// c.Locale() is set by the middleware; c.Translate looks up the key.
	app.Get("/", func(c httpsrv.Ctx) error {
		return c.SendString(c.Translate(c.Locale(), "greeting", "World"))
	})
	app.Get("/welcome", func(c httpsrv.Ctx) error {
		return c.SendString(c.Translate(c.Locale(), "welcome"))
	})

	fmt.Println("listening on :8080")
	log.Fatal(app.Run(":8080"))
}
