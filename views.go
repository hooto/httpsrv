// Copyright 2015 Eryx <evorui at gmail dot com>, All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package httpsrv

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"time"
)

// Views is the interface that wraps the Render function. A template engine
// implements it so Handler code can call Ctx.Render. The built-in *Renderer
// satisfies it; plug in a custom engine via WithViews or
// WithConfig(Config{Views: ...}).
//
// Default: nil
type Views interface {
	// Load is called once to load/parse templates. The built-in Renderer
	// parses at construction, so its Load is a no-op.
	Load() error

	// Render writes the named template (with optional layouts) to w.
	Render(w io.Writer, name string, bind any, layout ...string) error
}

// Renderer is the template engine. It parses html/template files from a
// filesystem (a directory via TemplatesDir, or an embed/other fs.FS via
// TemplatesFS) once at construction, then renders them by name with optional
// layouts. Built-in template functions (raw, replace, upper, lower, date,
// datetime) are always available; pass extraFuncs to add more (e.g. an I18n
// store's Funcs() to get T). *Renderer implements Views.
type Renderer struct {
	fsys fs.FS
	set  *template.Template // shared, parsed set (read-only after construction)
}

// Load is a no-op: templates are parsed once at construction (TemplatesFS/
// TemplatesDir), so by the time a Renderer is attached there is nothing left
// to load. It satisfies the Views interface.
func (r *Renderer) Load() error { return nil }

// Render renders name with bind into w, wrapping the output in each layout in
// order (the last layout is the outermost). It implements Views.
func (r *Renderer) Render(w io.Writer, name string, bind any, layout ...string) error {
	return r.execute(w, name, bind, layout...)
}

// TemplatesFS builds a Renderer from fsys (e.g. an embed.FS after fs.Sub). All
// .html/.tpl files are parsed immediately, so {{template "x"}} includes work.
func TemplatesFS(fsys fs.FS, extraFuncs template.FuncMap) (*Renderer, error) {
	r := &Renderer{fsys: fsys}
	r.set = template.New("").Funcs(builtinFuncs())
	if extraFuncs != nil {
		r.set = r.set.Funcs(extraFuncs)
	}
	if err := r.loadAll(); err != nil {
		return nil, err
	}
	return r, nil
}

// TemplatesDir builds a Renderer from the filesystem directory at root.
func TemplatesDir(root string, extraFuncs template.FuncMap) (*Renderer, error) {
	return TemplatesFS(os.DirFS(root), extraFuncs)
}

// loadAll parses every .html/.tpl file under the root into the shared set.
func (r *Renderer) loadAll() error {
	return fs.WalkDir(r.fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !(strings.HasSuffix(p, ".html") || strings.HasSuffix(p, ".tpl")) {
			return nil
		}
		data, err := fs.ReadFile(r.fsys, p)
		if err != nil {
			return fmt.Errorf("read template %s: %w", p, err)
		}
		name := strings.TrimPrefix(p, "./")
		if _, err := r.set.New(name).Parse(string(data)); err != nil {
			return fmt.Errorf("parse template %s: %w", p, err)
		}
		return nil
	})
}

// execute renders name with bind, wrapping the output in each layout in order
// (the last layout is the outermost). Each layout receives a map whose "Content"
// key holds the inner rendered HTML; when bind is a map, its entries are merged
// in (so layouts can use the same fields as the content template).
func (r *Renderer) execute(w io.Writer, name string, bind any, layouts ...string) error {
	name = cleanTemplateName(name)
	if r.set.Lookup(name) == nil {
		return fmt.Errorf("template %q not found", name)
	}

	var buf bytes.Buffer
	if err := r.set.ExecuteTemplate(&buf, name, bind); err != nil {
		return err
	}
	content := buf.String()

	for _, l := range layouts {
		l = cleanTemplateName(l)
		if r.set.Lookup(l) == nil {
			return fmt.Errorf("layout %q not found", l)
		}
		buf.Reset()
		if err := r.set.ExecuteTemplate(&buf, l, layoutData(bind, content)); err != nil {
			return err
		}
		content = buf.String()
	}
	_, err := io.WriteString(w, content)
	return err
}

// cleanTemplateName normalizes a template name (leading slash removed, "."/".."
// resolved) to match the names loadAll registered.
func cleanTemplateName(name string) string {
	return strings.TrimPrefix(path.Clean("/"+name), "/")
}

func layoutData(bind any, content string) any {
	// Content is the rendered inner template (already escaped by its own
	// execution), so mark it safe to avoid double-escaping in the layout.
	m := map[string]any{"Content": template.HTML(content)}
	if bm, ok := bind.(map[string]any); ok {
		for k, v := range bm {
			if k != "Content" {
				m[k] = v
			}
		}
	}
	return m
}

// builtinFuncs returns the always-available template functions. i18n (T) is
// opt-in via I18n.Funcs() passed as extraFuncs — it is not loaded by default.
func builtinFuncs() template.FuncMap {
	return template.FuncMap{
		"raw":      func(s string) template.HTML { return template.HTML(s) },
		"replace":  func(s, old, new string) string { return strings.ReplaceAll(s, old, new) },
		"upper":    strings.ToUpper,
		"lower":    strings.ToLower,
		"date":     func(t time.Time) string { return t.Format("2006-01-02") },
		"datetime": func(t time.Time) string { return t.Format("2006-01-02 15:04") },
	}
}
