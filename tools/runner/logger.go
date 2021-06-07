/*
Copyright 2021 gRPC authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package runner

import (
	"fmt"
	"io"
	"log"
	"time"
)

type Logger interface {
	Started(invocation *TestInvocation, t time.Time)
	Stopped(invocation *TestInvocation, t time.Time)
	Info(invocation *TestInvocation, detailsFmt string, args ...interface{})
	Warning(invocation *TestInvocation, brief, detailsFmt string, args ...interface{})
	Error(invocation *TestInvocation, brief, detailsFmt string, args ...interface{})
}

type LoggerList []Logger

var _ Logger = &LoggerList{}

func (ll LoggerList) Started(invocation *TestInvocation, t time.Time) {
	for _, l := range ll {
		l.Started(invocation, t)
	}
}

func (ll LoggerList) Stopped(invocation *TestInvocation, t time.Time) {
	for _, l := range ll {
		l.Stopped(invocation, t)
	}
}

func (ll LoggerList) Info(invocation *TestInvocation, detailsFmt string, args ...interface{}) {
	for _, l := range ll {
		l.Info(invocation, detailsFmt, args...)
	}
}

func (ll LoggerList) Warning(invocation *TestInvocation, brief, detailsFmt string, args ...interface{}) {
	for _, l := range ll {
		l.Warning(invocation, brief, detailsFmt, args...)
	}
}

func (ll LoggerList) Error(invocation *TestInvocation, brief, detailsFmt string, args ...interface{}) {
	for _, l := range ll {
		l.Error(invocation, brief, detailsFmt, args...)
	}
}

type TextLogger struct {
	log       *log.Logger
	prefixFmt string
}

var _ Logger = &TextLogger{}

func NewTextLogger(out io.Writer, prefixFmt string, flag int) *TextLogger {
	return &TextLogger{
		log:       log.New(out, "", flag),
		prefixFmt: prefixFmt,
	}
}

func (tl *TextLogger) Started(invocation *TestInvocation, t time.Time) {
	tl.log.Printf("%s Started at %s", tl.prefix(invocation), t.Format("2021/01/02 15:04:05 MST"))
}

func (tl *TextLogger) Stopped(invocation *TestInvocation, t time.Time) {
	tl.log.Printf("%s Stopped at %s", tl.prefix(invocation), t.Format("2021/01/02 15:04:05 MST"))
}

func (tl *TextLogger) Info(invocation *TestInvocation, detailsFmt string, args ...interface{}) {
	tl.log.Printf("%s %s", tl.prefix(invocation), fmt.Sprintf(detailsFmt, args...))
}

func (tl *TextLogger) Warning(invocation *TestInvocation, _, detailsFmt string, args ...interface{}) {
	tl.log.Printf("%s %s", tl.prefix(invocation), fmt.Sprintf(detailsFmt, args...))
}

func (tl *TextLogger) Error(invocation *TestInvocation, _, detailsFmt string, args ...interface{}) {
	tl.log.Printf("%s %s", tl.prefix(invocation), fmt.Sprintf(detailsFmt, args...))
}

func (tl *TextLogger) prefix(invocation *TestInvocation) string {
	return fmt.Sprintf(tl.prefixFmt, invocation)
}
