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

package junit

import (
	"fmt"
	"time"

	"github.com/grpc/test-infra/tools/runner"
)

// Logger implements the runner.Logger interface, filling out a JUnit report.
type Logger struct {
	report *Report
}

// Ensure Logger implements the runner.Logger interface.
var _ runner.Logger = &Logger{}

// NewLogger creates a new Logger instance, given a Report.
func NewLogger(report *Report) *Logger {
	return &Logger{
		report: report,
	}
}

func (jl *Logger) QueueStarted(qName string) {
	jl.report.RecordSuiteStart(qName, now())
}

func (jl *Logger) QueueStopped(qName string) {
	jl.report.RecordSuiteStop(qName, now())
}

func (jl *Logger) TestStarted(invocation *runner.TestInvocation) {
	jl.report.RecordTestStart(invocation)
}

func (jl *Logger) TestStopped(invocation *runner.TestInvocation) {
	jl.report.RecordTestStop(invocation)
}

func (jl *Logger) Info(invocation *runner.TestInvocation, detailsFmt string, args ...interface{}) {
	// info messages are not included in JUnit reports
}

func (jl *Logger) Warning(invocation *runner.TestInvocation, brief, detailsFmt string, args ...interface{}) {
	jl.report.RecordFailure(invocation, &Failure{
		Type:    Warning,
		Message: brief,
		Text:    fmt.Sprintf(detailsFmt, args...),
	})
}

func (jl *Logger) Error(invocation *runner.TestInvocation, brief, detailsFmt string, args ...interface{}) {
	jl.report.RecordFailure(invocation, &Failure{
		Type:    Error,
		Message: brief,
		Text:    fmt.Sprintf(detailsFmt, args...),
	})
}

// now provides a timestamp for the current moment. By default, it wraps the
// time.Now function. This wrapping allows time to be mocked for testing.
var now = func() time.Time {
	return time.Now()
}
