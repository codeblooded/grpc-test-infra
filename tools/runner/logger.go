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
)

type Logger interface {
	QueueStarted(name string)
	QueueStopped(name string)
	TestStarted(invocation *TestInvocation)
	TestStopped(invocation *TestInvocation)
	Info(invocation *TestInvocation, detailsFmt string, args ...interface{})
	Warning(invocation *TestInvocation, brief, detailsFmt string, args ...interface{})
	Error(invocation *TestInvocation, brief, detailsFmt string, args ...interface{})
}

type LoggerList []Logger

// Ensure LoggerList implements the Logger interface.
var _ Logger = &LoggerList{}

func (ll LoggerList) QueueStarted(qName string) {
	for _, l := range ll {
		l.QueueStarted(qName)
	}
}

func (ll LoggerList) QueueStopped(qName string) {
	for _, l := range ll {
		l.QueueStopped(qName)
	}
}

func (ll LoggerList) TestStarted(invocation *TestInvocation) {
	for _, l := range ll {
		l.TestStarted(invocation)
	}
}

func (ll LoggerList) TestStopped(invocation *TestInvocation) {
	for _, l := range ll {
		l.TestStopped(invocation)
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

// Ensure TextLogger implements the Logger interface.
var _ Logger = &TextLogger{}

func NewTextLogger(out io.Writer, prefixFmt string, flag int) *TextLogger {
	return &TextLogger{
		log:       log.New(out, "", flag),
		prefixFmt: prefixFmt,
	}
}

func (tl *TextLogger) QueueStarted(qName string) {
	tl.log.Printf("Starting tests in queue %s", qName)
}

func (tl *TextLogger) QueueStopped(qName string) {
	tl.log.Printf("Finished tests in queue %s", qName)
}

func (tl *TextLogger) TestStarted(invocation *TestInvocation) {
	tl.log.Printf("%s Starting test %d in queue %s", tl.prefix(invocation), invocation.Index, invocation.QueueName)
}

func (tl *TextLogger) TestStopped(invocation *TestInvocation) {
	tl.log.Printf("%s Finished test in queue %s", tl.prefix(invocation), invocation.QueueName)
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
	return fmt.Sprintf(tl.prefixFmt, invocation.QueueName, invocation.Index)
}
