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
	"fmt"
	"log"
)

type Logger interface {
	Debugf(format string, v ...interface{})
	Infof(format string, v ...interface{})
	Warnf(format string, v ...interface{})
	Errorf(format string, v ...interface{})
	Fatalf(format string, v ...interface{})
}

var defaultLogger Logger = &emptyLogger{}

type emptyLogger struct{}

func NewEmptyLogger() Logger {
	return &emptyLogger{}
}

func (emptyLogger) Debugf(format string, v ...interface{}) {}

func (emptyLogger) Infof(format string, v ...interface{}) {}

func (emptyLogger) Warnf(format string, v ...interface{}) {}

func (emptyLogger) Errorf(format string, v ...interface{}) {}

func (emptyLogger) Fatalf(format string, v ...interface{}) {}

type rawLogger struct {
	log *log.Logger
}

func NewRawLogger() Logger {
	return &rawLogger{
		log: log.Default(),
	}
}

func (it *rawLogger) Debugf(format string, v ...interface{}) {
	it.log.Output(2, fmt.Sprintf(format, v...))
}

func (it *rawLogger) Infof(format string, v ...interface{}) {
	it.log.Output(2, fmt.Sprintf(format, v...))
}

func (it *rawLogger) Warnf(format string, v ...interface{}) {
	it.log.Output(2, fmt.Sprintf(format, v...))
}

func (it *rawLogger) Errorf(format string, v ...interface{}) {
	it.log.Output(2, fmt.Sprintf(format, v...))
}

func (it *rawLogger) Fatalf(format string, v ...interface{}) {
	it.log.Output(2, fmt.Sprintf(format, v...))
}
