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

type Ctx interface {
	Request() *Request

	Response() *Response

	Header(key string, defaultValue ...string) string

	Body() []byte

	Params() *Params

	Status(status int) Ctx

	JSON(data any) error

	Send(body []byte) error
}

type ctxImpl struct {
	c *Controller
}

func (it *ctxImpl) Request() *Request {
	return it.c.Request
}

func (it *ctxImpl) Response() *Response {
	return it.c.Response
}

func (it *ctxImpl) Header(key string, defaultValue ...string) string {
	v := it.c.Request.Header.Get(key)
	if v == "" && len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return v
}

func (it *ctxImpl) Body() []byte {
	return it.c.Request.RawBody()
}

func (it *ctxImpl) Params() *Params {
	return it.c.Params
}

func (it *ctxImpl) Status(status int) Ctx {
	it.c.Response.Status = status
	return it
}

func (it *ctxImpl) JSON(data any) error {
	it.c.RenderJson(data)
	return nil
}

func (it *ctxImpl) Send(body []byte) error {
	it.c.Response.buf.Write(body)
	return nil
}
