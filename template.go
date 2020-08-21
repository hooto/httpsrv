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

	"github.com/hooto/hlog4g/hlog"
)

// This object handles loading and parsing of templates.
// Everything below the application's views directory is treated as a template.
type TemplateLoader struct {
	mu sync.RWMutex

	// Map from template name to the path from whence it was loaded.
	templatePaths map[string]string

	// This is the set of all templates under views
	templateSets map[string]*template.Template
}

func (it *TemplateLoader) Clean(modName string) {

	it.mu.Lock()
	defer it.mu.Unlock()

	if _, ok := it.templateSets[modName]; ok {
		delete(it.templateSets, modName)
	}

	for k := range it.templatePaths {

		if strings.HasPrefix(k, modName+".") {
			delete(it.templatePaths, k)
		}
	}
}

func (it *TemplateLoader) Set(modName string, viewpaths []string, viewfss []http.FileSystem) {

	it.mu.Lock()
	defer it.mu.Unlock()

	loaderTemplateSet, ok := it.templateSets[modName]
	if ok {
		return
	}

	addTemplate := func(templateFile, fileStr string) error {

		templateName := strings.Trim(templateFile, "/")
		templateNameL := strings.ToLower(templateName)

		if _, ok := it.templatePaths[modName+"."+templateNameL]; ok {
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
					it.templateSets[modName] = loaderTemplateSet
				}
			}()

		} else {

			_, err = loaderTemplateSet.New(templateName).Parse(fileStr)
		}

		if err == nil {

			if templateNameL != templateName {
				loaderTemplateSet.New(templateNameL).Parse(fileStr)
			}

			it.templatePaths[modName+"."+templateNameL] = templateFile
		} else {
			hlog.Printf("warn", "template (%s/%s) parse err %s", modName, templateFile, err.Error())
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
				if _, err = io.Copy(&buf, fp); err != nil {
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

func (it *TemplateLoader) Render(wr io.Writer, modName, tplPath string, arg interface{}) error {

	it.mu.RLock()
	defer it.mu.RUnlock()

	defer func() {
		if err := recover(); err != nil {
			hlog.Printf("debug", "template (%s/%s) render err %v", modName, tplPath, err)
		}
	}()

	tplSet, ok := it.templateSets[modName]
	if !ok || tplSet == nil {
		return fmt.Errorf("module %s not found", modName)
	}
	// return tplSet.ExecuteTemplate(wr, tplPath, arg)

	tpl := tplSet.Lookup(tplPath)
	if tpl == nil {
		if tplPathl := strings.ToLower(tplPath); tplPathl != tplPath {
			tpl = tplSet.Lookup(tplPathl)
		}
		if tpl == nil {
			return fmt.Errorf("template %s/%s not found", modName, tplPath)
		}
	}
	return tpl.Execute(wr, arg)
}
