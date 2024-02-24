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
	"compress/gzip"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/andybalholm/brotli"
)

type regHandler struct {
	service *Service
	method  string
	pattern string

	params map[string]string

	handler           *http.Handler
	handlerFunc       func(w http.ResponseWriter, r *http.Request)
	handlerController *handlerController
	handlerContext    func(ctx *Context) error
	handlerStatic     *handlerStaticFile

	invoker interface{}
}

type handlerStaticFile struct {
	binFs    http.FileSystem
	filepath string
}

var (
	genArgs = []reflect.Value{}
)

func (it *regHandler) handle(w http.ResponseWriter, r *http.Request) {

	if it.handlerStatic != nil {

		urlPath := filepath.Clean(r.URL.Path)
		if !strings.HasPrefix(urlPath, it.pattern) {
			return
		}
		subPath := urlPath[len(it.pattern)-1:]

		if it.handlerStatic.binFs != nil {
			if fp, err := it.handlerStatic.binFs.Open(subPath); err == nil {
				defer fp.Close()
				if st, err := fp.Stat(); err == nil {
					http.ServeContent(w, r, st.Name(), st.ModTime(), fp)
					return
				}
			}
			http.NotFound(w, r)
			return
		}

		file := filepath.Clean(it.handlerStatic.filepath + "/" + subPath)

		finfo, err := os.Stat(file)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		if finfo.IsDir() {
			http.NotFound(w, r)
			return
		}

		http.ServeFile(w, r, file)
		return
	}

	if it.handlerFunc != nil {
		it.handlerFunc(w, r)
		return
	}

	var (
		req  = newRequest(r)
		resp = newResponse(w)
		c    = newController(it.service, req, resp)
		ae   = r.Header.Get("Accept-Encoding")
	)

	for _, filter := range it.service.Filters {
		filter(c)
	}

	if len(it.params) > 0 {
		for name, _ := range it.params {
			c.Params.setValue(name, r.PathValue(name))
		}
	}

	if it.handlerController != nil {

		var (
			appControllerPtr  = reflect.New(it.handlerController.ctrlType)
			appControllerInst = appControllerPtr.Elem()
			cValue            = reflect.ValueOf(c)
		)

		for _, index := range it.handlerController.ctrlIndexes {
			appControllerInst.FieldByIndex(index).Set(cValue)
		}

		appController := appControllerPtr.Interface()

		//
		execController := reflect.ValueOf(appController).MethodByName("Init")
		if execController.Kind() != reflect.Invalid {
			if iv := execController.Call(genArgs)[0]; iv.Kind() == reflect.Int {
				if iv.Int() != 0 {
					return
				}
			}
		}

		//
		execController = reflect.ValueOf(appController).MethodByName(it.handlerController.ActionName + "Action")
		if execController.Kind() == reflect.Invalid && it.handlerController.ActionName != "Index" {
			execController = reflect.ValueOf(appController).MethodByName("IndexAction")
			if execController.Kind() == reflect.Invalid {
				return
			}
		}

		c.modPath = it.handlerController.ModPath
		c.Name = it.handlerController.Name
		c.ActionName = it.handlerController.ActionName

		//
		if execController.Type().IsVariadic() {
			execController.CallSlice(genArgs)
		} else {
			execController.Call(genArgs)
		}

	} else if it.handlerContext != nil {
		it.handlerContext(&Context{
			c: c,
		})
	}

	if c.AutoRender {
		c.Render()
	}

	if ae != "" && it.service.Config.CompressResponse {
		if strings.Contains(ae, "gzip") {
			resp.compWriter, ae = gzip.NewWriter(resp.buf), "gzip"
		} else if strings.Contains(ae, "br") {
			resp.compWriter, ae = brotli.NewWriterLevel(resp.buf, 5), "br"
		}
	}

	if resp.compWriter != nil {
		resp.compWriter.Flush()
		resp.compWriter.Close()

		if w.Header().Get("Content-Encoding") == "" && resp.buf.Len() > 0 {
			if ae == "gzip" {
				w.Header().Set("Content-Encoding", "gzip")
			} else if ae == "br" {
				w.Header().Set("Content-Encoding", "br")
			}
		}
	}

	if resp.buf != nil && resp.buf.Len() > 0 {
		w.Header().Set("Content-Length", strconv.Itoa(resp.buf.Len()))
		if resp.Status > 0 {
			w.WriteHeader(resp.Status)
		}
		w.Write(resp.buf.Bytes())
	} else if resp.Status > 0 {
		w.WriteHeader(resp.Status)
	}
}

var defaultHandlers = []*regHandler{
	{
		pattern: "/",
		handlerFunc: func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, `page not found`)
		},
	},
}

func handlerPathSlice(path string) (string, []string) {
	path = strings.Replace(filepath.Clean(path), " ", "", -1)
	if runtime.GOOS == "windows" {
		path = strings.Replace(path, "\\", "/", -1)
	}
	path = strings.Trim(path, "/")
	return "/" + path, strings.Split(path, "/")
}
