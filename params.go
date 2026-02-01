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
	"net/http"
	"net/url"
	"strconv"
)

type Params struct {
	inited bool

	// A unified view of all the individual param maps below
	values url.Values

	request *http.Request
}

func ParamsFilter(c *Controller) {
	if c.Params == nil {
		c.Params = &Params{
			request: c.Request.Request,
		}
	} else {
		c.Params.request = c.Request.Request
	}
}

// func (p *Params) reset() *Params {
// 	p.values = make(url.Values)
// 	p.inited = false
// 	p.request = nil
// 	return p
// }

func (p *Params) init() {
	if p.inited {
		return
	}
	p.inited = true
	if p.request != nil {
		p.values = p.request.URL.Query()
		if p.request.Method == "POST" ||
			p.request.Method == "PUT" ||
			p.request.Method == "PATCH" {
			p.request.ParseForm()
		}
	}
}

func (p *Params) SetValue(key, value string) {
	p.init()
	if p.values == nil {
		p.values = make(url.Values)
	}
	if p.values.Has(key) {
		p.values[key] = append(p.values[key], value)
	} else {
		p.values[key] = []string{value}
	}
}

func (p *Params) Value(key string) string {
	p.init()
	if v := p.request.PathValue(key); v != "" {
		return v
	}
	if p.values != nil && p.values.Has(key) {
		return p.values.Get(key)
	}
	if p.request.Form != nil && p.request.Form.Has(key) {
		return p.request.Form.Get(key)
	}
	if p.request.PostForm != nil && p.request.PostForm.Has(key) {
		return p.request.PostForm.Get(key)
	}
	return ""
}

func (p *Params) IntValue(key string) int64 {
	if s := p.Value(key); s != "" {
		if i, e := strconv.ParseInt(s, 10, 64); e == nil {
			return i
		}
	}
	return 0
}

func (p *Params) FloatValue(key string) float64 {
	if s := p.Value(key); s != "" {
		if f, e := strconv.ParseFloat(s, 64); e == nil {
			return f
		}
	}
	return 0
}
