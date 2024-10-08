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
	"encoding/json"
	"hash/crc64"
	"sync"
)

var bytesBufferPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

func jsonEncode(v interface{}, indent string) ([]byte, error) {
	if indent != "" {
		return json.MarshalIndent(v, "", indent)
	}
	return json.Marshal(v)
}

func jsonDecode(src []byte, v interface{}) error {
	return json.Unmarshal(src, &v)
}

func isAlnum(v byte) bool {
	if (v >= 'A' && v <= 'Z') ||
		(v >= 'a' && v <= 'z') ||
		(v >= '0' && v <= '9') {
		return true
	}
	return false
}

func isUpper(v byte) bool {
	if v >= 'A' && v <= 'Z' {
		return true
	}
	return false
}

var crc64ecma182 = crc64.MakeTable(crc64.ECMA)

func crc64Checksum(b []byte) uint64 {
	return crc64.Checksum(b, crc64ecma182)
}
