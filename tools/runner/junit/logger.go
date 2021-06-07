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

type TestCaseLogger struct {
	reportTestCase *ReportTestCase
}

var _ runner.Logger = &TestCaseLogger{}

func NewTestCaseLogger(rtc *ReportTestCase) *TestCaseLogger {
	return &TestCaseLogger{
		reportTestCase: rtc,
	}
}

func (tcl *TestCaseLogger) Started(_ *runner.TestInvocation, t time.Time) {
	tcl.reportTestCase.SetStartTime(t)
}

func (tcl *TestCaseLogger) Stopped(_ *runner.TestInvocation, t time.Time) {
	tcl.reportTestCase.SetStopTime(t)
}

func (tcl *TestCaseLogger) Info(_ *runner.TestInvocation, detailsFmt string, args ...interface{}) {}

func (tcl *TestCaseLogger) Warning(_ *runner.TestInvocation, brief, detailsFmt string, args ...interface{}) {
	tcl.reportTestCase.AddFailure(&Failure{
		Type:    Warning,
		Message: brief,
		Text:    fmt.Sprintf(detailsFmt, args...),
	})
}

func (tcl *TestCaseLogger) Error(_ *runner.TestInvocation, brief, detailsFmt string, args ...interface{}) {
	tcl.reportTestCase.AddFailure(&Failure{
		Type:    Error,
		Message: brief,
		Text:    fmt.Sprintf(detailsFmt, args...),
	})
}
