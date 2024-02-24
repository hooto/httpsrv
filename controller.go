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
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
)

type Controller struct {
	Name       string // The controller name, e.g. "App"
	ActionName string // The action name, e.g. "Index"
	Request    *Request
	Response   *Response
	Params     *Params  // Parameters from URL and form (including multipart).
	Session    *Session // Session, stored in cookie, signed.
	AutoRender bool
	Data       map[string]interface{}
	modPath    string
	service    *Service
}

type handlerController struct {
	ModPath     string
	Name        string
	ActionName  string
	ctrlType    reflect.Type
	ctrlIndexes [][]int
}

var (
	controllerPtrType = reflect.TypeOf(&Controller{})
)

func newController(srv *Service, req *Request, resp *Response) *Controller {

	return &Controller{
		Name:       "Index",
		ActionName: "Index",
		service:    srv,
		Request:    req,
		Response:   resp,
		Params:     &Params{},
		AutoRender: true,
		Data:       map[string]interface{}{},
	}
}

func (c *Controller) Render(args ...interface{}) {

	c.AutoRender = false

	modPath, templatePath := c.modPath, c.Name+"/"+c.ActionName+".tpl"

	if len(args) == 2 &&
		reflect.TypeOf(args[0]).Kind() == reflect.String &&
		reflect.TypeOf(args[1]).Kind() == reflect.String {

		modPath, templatePath = args[0].(string), args[1].(string)

	} else if len(args) == 1 &&
		reflect.TypeOf(args[0]).Kind() == reflect.String {

		templatePath = args[0].(string)
	}

	// Handle panics when rendering templates.
	defer func() {
		if err := recover(); err != nil {
			// println(err)
		}
	}()

	err := c.service.TemplateLoader.Render(c.Response, modPath, templatePath, c.Data)
	if err != nil {
		c.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
		c.Response.Out.WriteHeader(http.StatusBadRequest)
		c.Response.Out.Write([]byte("400 Bad Request"))
	} else {
		c.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
		c.Response.WriteHeader(http.StatusOK)
	}
}

func (c *Controller) RenderError(status int, msg string) {
	c.AutoRender = false
	c.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Response.Out.WriteHeader(status)
	c.Response.Out.Write([]byte(msg))
}

func (c *Controller) UrlBase(path string) string {

	urlBase := ""
	if c.Request.TLS != nil {
		urlBase = "https://" + c.Request.Host
	} else {
		urlBase = "http://" + c.Request.Host
	}

	if c.service.Config.UrlBasePath != "" {
		urlBase += "/" + c.service.Config.UrlBasePath
	}

	if len(path) > 0 {
		path = filepath.Clean(path)
	}

	if path != "" {
		urlBase += "/" + path
	}

	return urlBase
}

func (c *Controller) UrlModuleBase(path string) string {
	return c.UrlBase(c.modPath + "/" + path)
}

func (c *Controller) Redirect(url string) {

	c.AutoRender = false

	if len(url) < 1 {
		return
	}

	if url[0] != '/' && !strings.HasPrefix(url, "http") {

		if c.service.Config.UrlBasePath != "" {
			c.Response.Header().Set("Location", "/"+c.service.Config.UrlBasePath+"/"+url)
		} else {
			c.Response.Header().Set("Location", "/"+url)
		}
	} else {
		c.Response.Header().Set("Location", url)
	}

	c.Response.WriteHeader(http.StatusFound)
}

func (c *Controller) RenderString(v string) {
	c.AutoRender = false
	c.Response.Write([]byte(v))
}

func (c *Controller) RenderJson(obj interface{}) {
	c.RenderJsonIndent(obj, "")
}

func (c *Controller) RenderJsonIndent(obj interface{}, indent string) {

	c.AutoRender = false

	c.Response.Header().Set("Access-Control-Allow-Origin", "*")
	c.Response.Header().Set("Content-type", "application/json")

	if js, err := jsonEncode(obj, indent); err == nil {
		c.Response.Write(js)
	}
}

func (c *Controller) Translate(msg string, args ...interface{}) string {
	return i18nTranslate(c.Request.Locale, msg, args...)
}

func findControllers(appControllerType reflect.Type) (indexes [][]int) {

	// It might be a multi-level embedding. To find the controllers, we follow
	// every anonymous field, using breadth-first search.
	type nodeType struct {
		val   reflect.Value
		index []int
	}

	var (
		appControllerPtr = reflect.New(appControllerType)
		queue            = []nodeType{{appControllerPtr, []int{}}}
	)

	for len(queue) > 0 {
		// Get the next value and de-reference it if necessary.
		var (
			node     = queue[0]
			elem     = node.val
			elemType = elem.Type()
		)

		if elemType.Kind() == reflect.Ptr {
			elem = elem.Elem()
			elemType = elem.Type()
		}

		queue = queue[1:]

		// Look at all the struct fields.
		for i := 0; i < elem.NumField(); i++ {

			// If this is not an anonymous field, skip it.
			structField := elemType.Field(i)
			if !structField.Anonymous {
				continue
			}

			fieldValue := elem.Field(i)
			fieldType := structField.Type

			// If it's a Controller, record the field indexes to get here.
			if fieldType == controllerPtrType {
				indexes = append(indexes, append(node.index, i))
				continue
			}

			queue = append(queue, nodeType{fieldValue,
				append(append([]int{}, node.index...), i)})
		}
	}

	return
}
