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
	"net/http"
)

type Response struct {
	Status     int
	Out        http.ResponseWriter
	buf        *bytes.Buffer
	compWriter compressWriter
}

type compressWriter interface {
	Write(b []byte) (int, error)
	Flush() error
	Close() error
}

func NewResponse(w http.ResponseWriter) *Response {
	return &Response{Out: w}
}

func (resp *Response) Write(b []byte) (int, error) {
	if resp.compWriter != nil {
		return resp.compWriter.Write(b)
	}
	return resp.buf.Write(b)
}

func (resp *Response) Header() http.Header {
	return resp.Out.Header()
}

func (resp *Response) WriteHeader(status int) {
	if status > resp.Status {
		resp.Status = status
	}
}
