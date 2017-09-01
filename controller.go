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
	"io"
	"net/http"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

type Controller struct {
	Name          string // The controller name, e.g. "App"
	ActionName    string // The action name, e.g. "Index"
	Request       *Request
	Response      *Response
	Params        *Params  // Parameters from URL and form (including multipart).
	Session       *Session // Session, stored in cookie, signed.
	AutoRender    bool
	Data          map[string]interface{}
	appController interface{} // The controller that was instantiated.
	mod_name      string
	mod_urlbase   string
	service       *Service
}

type controllerType struct {
	Type              reflect.Type
	Methods           []string
	ControllerIndexes [][]int
}

var (
	controllerPtrType = reflect.TypeOf(&Controller{})
)

func NewController(srv *Service, req *Request, resp *Response) *Controller {

	return &Controller{
		Name:       "Index",
		ActionName: "Index",
		service:    srv,
		Request:    req,
		Response:   resp,
		Params:     newParams(),
		AutoRender: true,
		Data:       map[string]interface{}{},
	}
}

var (
	gen_args = []reflect.Value{}
)

func ActionInvoker(c *Controller) {

	//
	if c.appController == nil {
		return
	}

	//
	execController := reflect.ValueOf(c.appController).MethodByName("Init")
	if execController.Kind() != reflect.Invalid {

		if iv := execController.Call(gen_args)[0]; iv.Kind() == reflect.Int {

			if iv.Int() != 0 {
				return
			}
		}
	}

	//
	execController = reflect.ValueOf(c.appController).MethodByName(c.ActionName + "Action")
	if execController.Kind() == reflect.Invalid && c.ActionName != "Index" {

		execController = reflect.ValueOf(c.appController).MethodByName("IndexAction")

		if execController.Kind() == reflect.Invalid {
			return
		}
	}

	//
	if execController.Type().IsVariadic() {
		execController.CallSlice(gen_args)
	} else {
		execController.Call(gen_args)
	}

	if c.AutoRender {
		c.Render()
	}
}

func (c *Controller) Render(args ...interface{}) {

	c.AutoRender = false

	mod_name, templatePath := c.mod_name, c.Name+"/"+c.ActionName+".tpl"

	if len(args) == 2 &&
		reflect.TypeOf(args[0]).Kind() == reflect.String &&
		reflect.TypeOf(args[1]).Kind() == reflect.String {

		mod_name, templatePath = args[0].(string), args[1].(string)

	} else if len(args) == 1 &&
		reflect.TypeOf(args[0]).Kind() == reflect.String {

		templatePath = args[0].(string)
	}

	// Handle panics when rendering templates.
	defer func() {
		if err := recover(); err != nil {

		}
	}()

	template, err := c.service.TemplateLoader.Template(mod_name, templatePath)
	if err != nil {
		return //c.RenderError(err)
	}

	if c.Response.Status == 0 {
		c.Response.Status = http.StatusOK
	}
	c.Response.WriteHeader(c.Response.Status, "text/html; charset=utf-8")

	out := io.Writer(c.Response.Out)
	if err = template.Render(out, c.Data); err != nil {
		println(err)
	}
}

func (c *Controller) RenderError(status int, msg string) {

	c.AutoRender = false

	c.Response.WriteHeader(status, "text/html; charset=utf-8")
	io.WriteString(c.Response.Out, msg)
}

func (c *Controller) UrlBase(path string) string {

	url_base := ""
	if c.Request.TLS != nil {
		url_base = "https://" + c.Request.Host
	} else {
		url_base = "http://" + c.Request.Host
	}

	if c.service.Config.UrlBasePath != "" {
		url_base += "/" + c.service.Config.UrlBasePath
	}

	if len(path) > 0 {
		path = filepath.Clean(path)
	}

	if path != "" {
		url_base += "/" + path
	}

	return url_base
}

func (c *Controller) UrlModuleBase(path string) string {
	return c.UrlBase(c.mod_urlbase + "/" + path)
}

func (c *Controller) Redirect(url string) {

	c.AutoRender = false

	if len(url) < 1 {
		return
	}

	if url[0] != '/' && !strings.HasPrefix(url, "http") {

		if c.service.Config.UrlBasePath != "" {
			c.Response.Out.Header().Set("Location", "/"+c.service.Config.UrlBasePath+"/"+url)
		} else {
			c.Response.Out.Header().Set("Location", "/"+url)
		}
	} else {
		c.Response.Out.Header().Set("Location", url)
	}

	c.Response.Out.WriteHeader(http.StatusFound)
}

func (c *Controller) RenderString(body string) {

	c.AutoRender = false

	io.WriteString(c.Response.Out, body)
}

func (c *Controller) RenderJson(obj interface{}) {
	c.RenderJsonIndent(obj, "")
}

func (c *Controller) RenderJsonIndent(obj interface{}, indent string) {

	c.AutoRender = false

	c.Response.Out.Header().Set("Access-Control-Allow-Origin", "*")
	c.Response.Out.Header().Set("Content-type", "application/json")

	if js, err := json_encode(obj, indent); err == nil {
		c.Response.Out.Header().Set("Content-Length", strconv.Itoa(len(js)))
		c.Response.Out.Write(js)
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
