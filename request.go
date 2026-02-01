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
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hooto/httpsrv/internal/lru"
)

type Request struct {
	*http.Request
	Time           time.Time
	ContentType    string
	acceptLanguage []*acceptLanguage
	Locale         string

	urlPath      string
	urlRoutePath string

	bodyRead   bool
	bodyBuffer bytes.Buffer
}

// A single language from the Accept-Language HTTP header.
type acceptLanguage struct {
	Language string
	Quality  float32
}

var acceptLanguageCache = lru.New(1024)

func newRequest(r *http.Request) *Request {

	req := &Request{
		Request:        r,
		ContentType:    resolveContentType(r),
		acceptLanguage: resolveAcceptLanguage(r),
		Locale:         "",
		bodyRead:       false,
	}

	if req.ContentType == "application/x-www-form-urlencoded" &&
		(r.Method == "POST" || r.Method == "PUT") && req.Body != nil {
		if _, err := io.Copy(&req.bodyBuffer, req.Body); err == nil {
			req.Body = io.NopCloser(bytes.NewReader(req.bodyBuffer.Bytes()))
			req.bodyRead = true
		}
	}

	return req
}

func (req *Request) RawBody() []byte {
	if !req.bodyRead && req.Body != nil {
		if _, err := io.Copy(&req.bodyBuffer, req.Body); err != nil {
			//
		}
		req.bodyRead = true
	}
	return req.bodyBuffer.Bytes()
}

func (req *Request) UrlPath() string {
	if req.urlPath == "" {
		req.urlPath = filepath.Clean("/" + req.Request.URL.Path)
	}
	return req.urlPath
}

func (req *Request) UrlRoutePath() string {
	return req.urlRoutePath
}

func (req *Request) RawAbsUrl() string {

	scheme := "http"

	if len(req.URL.Scheme) > 0 {
		scheme = req.URL.Scheme
	}

	return fmt.Sprintf("%s://%s%s", scheme, req.Host, req.RequestURI)
}

func (req *Request) JsonDecode(obj interface{}) error {

	if len(req.RawBody()) < 2 {
		return fmt.Errorf("No Data Found")
	}

	return jsonDecode(req.RawBody(), obj)
}

// Get the content type.
// e.g. From "multipart/form-data; boundary=--" to "multipart/form-data"
// If none is specified, returns "text/html" by default.
func resolveContentType(r *http.Request) string {
	v := r.Header.Get("Content-Type")
	if v == "" {
		return "text/html"
	}
	base, _, _ := strings.Cut(v, ";")
	return strings.ToLower(strings.TrimSpace(base))
}

// Resolve the Accept-Language header value.
//
// The results are sorted using the quality defined in the header for each language range with the
// most qualified language range as the first element in the slice.
//
// See the HTTP header fields specification
// (http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.4) for more details.
func resolveAcceptLanguage(r *http.Request) acceptLanguages {

	var k = r.Header.Get("Accept-Language")
	if k == "" || len(k) > 128 {
		return make(acceptLanguages, 0)
	}

	o, ok := acceptLanguageCache.Get(k)
	if ok {
		return o.(acceptLanguages)
	}

	var (
		rals = strings.Split(k, ",")
		als  = make(acceptLanguages, 0, len(rals))
	)

	for i, v := range rals {

		if bef, aft, ok := strings.Cut(v, ";q="); ok {
			quality, err := strconv.ParseFloat(aft, 32)
			if err != nil {
				quality = 1
			}
			als = append(als, &acceptLanguage{
				Language: bef,
				Quality:  float32(quality),
			})
		} else {
			als = append(als, &acceptLanguage{
				Language: v,
				Quality:  1,
			})
		}
		if i >= 5 {
			break
		}
	}
	if len(als) > 1 {
		sort.Sort(als)
	}

	acceptLanguageCache.Add(k, als)

	return als
}

// func (it *acceptLanguage) set(lang string, qua float32) *acceptLanguage {
// 	it.Language = lang
// 	it.Quality = qua
// 	return it
// }

// A collection of sortable acceptLanguage instances.
type acceptLanguages []*acceptLanguage

func (al acceptLanguages) Len() int           { return len(al) }
func (al acceptLanguages) Swap(i, j int)      { al[i], al[j] = al[j], al[i] }
func (al acceptLanguages) Less(i, j int) bool { return al[i].Quality > al[j].Quality }
func (al acceptLanguages) String() string {
	output := bytes.NewBufferString("")
	for i, language := range al {
		output.WriteString(fmt.Sprintf("%s (%1.1f)", language.Language, language.Quality))
		if i != len(al)-1 {
			output.WriteString(", ")
		}
	}
	return output.String()
}
