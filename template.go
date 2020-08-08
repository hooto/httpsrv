// Copyright 2015 Eryx <evorui аt gmаil dοt cοm>, All rights reserved.
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
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path"
	"strings"
	"sync"
)

var tlock sync.Mutex

// This object handles loading and parsing of templates.
// Everything below the application's views directory is treated as a template.
type TemplateLoader struct {
	// Map from template name to the path from whence it was loaded.
	templatePaths map[string]string

	// This is the set of all templates under views
	templateSets map[string]*template.Template
}

type iTemplate interface {
	Render(wr io.Writer, arg interface{}) error
}

func (loader *TemplateLoader) Clean(modname string) {

	tlock.Lock()
	defer tlock.Unlock()

	if _, ok := loader.templateSets[modname]; ok {
		delete(loader.templateSets, modname)
	}

	for k := range loader.templatePaths {

		if strings.HasPrefix(k, modname+".") {
			delete(loader.templatePaths, k)
		}
	}
}

func (loader *TemplateLoader) Set(modname string, viewpaths []string, viewfss []http.FileSystem) {

	tlock.Lock()
	defer tlock.Unlock()

	loaderTemplateSet, ok := loader.templateSets[modname]
	if ok {
		return
	}

	addTemplate := func(templateFile, fileStr string) error {

		templateName := strings.ToLower(strings.Trim(templateFile, "/"))

		if _, ok := loader.templatePaths[modname+"."+templateName]; ok {
			return nil
		}

		var err error

		if loaderTemplateSet == nil {

			func() {

				defer func() {
					if e := recover(); e != nil {
						err = errors.New("Panic (Template Loader)")
					}
				}()

				loaderTemplateSet = template.New(templateName).Funcs(TemplateFuncs)

				if _, err = loaderTemplateSet.Parse(fileStr); err == nil {
					loader.templateSets[modname] = loaderTemplateSet
				}
			}()

		} else {

			_, err = loaderTemplateSet.New(templateName).Parse(fileStr)
		}

		if err == nil {
			loader.templatePaths[modname+"."+templateName] = templateFile
		}

		return err
	}

	var hfsWalk func(fs http.FileSystem, dir string) error

	hfsWalk = func(fs http.FileSystem, dir string) error {

		fp, err := fs.Open(dir)
		if err != nil {
			return err
		}
		defer fp.Close()

		st, err := fp.Stat()
		if err != nil {
			return err
		}

		if !st.IsDir() {
			if strings.HasSuffix(dir, ".tpl") {
				var buf bytes.Buffer
				_, err = io.Copy(&buf, fp)
				if err != nil {
					return err
				}
				addTemplate(dir, buf.String())
			}
			return nil
		}

		nodes, err := fp.Readdir(-1)
		if err != nil {
			return err
		}

		for _, n := range nodes {

			if n.Name() == "." || n.Name() == ".." {
				continue
			}

			if err = hfsWalk(fs, path.Join(dir, n.Name())); err != nil {
				return err
			}
		}

		return nil
	}

	for _, baseDir := range viewpaths {
		hfsWalk(http.Dir(baseDir), "/")
	}

	for _, fs := range viewfss {
		hfsWalk(fs, "/")
	}
}

func (loader *TemplateLoader) Template(modname, tplname string) (iTemplate, error) {

	set, ok := loader.templateSets[modname]
	if !ok || set == nil {
		return nil, fmt.Errorf("Template %s not found.", tplname)
	}

	tplname = strings.ToLower(tplname)

	tmpl := set.Lookup(tplname)
	if tmpl == nil {
		return nil, fmt.Errorf("Template %s:%s not found.", modname, tplname)
	}

	return goTemplate{tmpl, loader}, nil
}

// Adapter for Go Templates.
type goTemplate struct {
	*template.Template
	loader *TemplateLoader
}

// return a 'httpsrv.goTemplate' from Go's template.
func (gotmpl goTemplate) Render(wr io.Writer, arg interface{}) error {

	defer func() {
		if err := recover(); err != nil {
			//
		}
	}()

	return gotmpl.Execute(wr, arg)
}
