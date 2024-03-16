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
	"compress/gzip"
	"fmt"
	"testing"

	"github.com/andybalholm/brotli"
)

var (
	compressTestText = []byte(`<!DOCTYPE html>
<html>
<head>
<title>Error</title>
<style>
html { color-scheme: light dark; }
body { width: 35em; margin: 0 auto;
font-family: Tahoma, Verdana, Arial, sans-serif; }
</style>
</head>
<body>
<h1>An error occurred.</h1>
<p>Sorry, the page you are looking for is currently unavailable.<br/>
Please try again later.</p>
<p>If you are the system administrator of this resource then you should check
the error log for details.</p>
<p><em>Faithfully yours, nginx.</em></p>
</body>
</html>`)
)

func Test_gzip(t *testing.T) {

	var buf bytes.Buffer

	for l := gzip.BestSpeed; l <= gzip.BestCompression; l++ {

		buf.Reset()
		w, _ := gzip.NewWriterLevel(&buf, l)
		w.Write(compressTestText)
		w.Flush()
		w.Close()

		t.Logf("gzip level %d ratio %f", l, float64(buf.Len())/float64(len(compressTestText)))
	}
}

func Test_brotli(t *testing.T) {

	var buf bytes.Buffer

	for l := brotli.BestSpeed; l <= brotli.BestCompression; l++ {

		buf.Reset()
		w := brotli.NewWriterLevel(&buf, l)
		w.Write(compressTestText)
		w.Flush()
		w.Close()

		t.Logf("brotli level %d ratio %f", l, float64(buf.Len())/float64(len(compressTestText)))
	}
}

func Benchmark_gzip_encoding(b *testing.B) {

	for l := gzip.BestSpeed; l <= gzip.BestCompression; l++ {
		b.Run(fmt.Sprintf("level_%d", l), func(b *testing.B) {

			var (
				buf  bytes.Buffer
				w, _ = gzip.NewWriterLevel(&buf, l)
			)
			defer w.Close()

			for i := 0; i < b.N; i++ {

				buf.Reset()
				w.Reset(&buf)

				w.Write(compressTestText)
				w.Flush()
			}
		})
	}
}

func Benchmark_brotli_encoding(b *testing.B) {
	for l := brotli.BestSpeed; l <= brotli.BestCompression; l++ {
		b.Run(fmt.Sprintf("level_%d", l), func(b *testing.B) {

			var (
				buf bytes.Buffer
				w   = brotli.NewWriterLevel(&buf, l)
			)
			defer w.Close()

			for i := 0; i < b.N; i++ {

				buf.Reset()
				w.Reset(&buf)

				w.Write(compressTestText)
				w.Flush()
			}
		})
	}
}
