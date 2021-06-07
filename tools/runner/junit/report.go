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
	"encoding/xml"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/grpc/test-infra/tools/runner"
	"github.com/pkg/errors"
)

type Report struct {
	name      string
	suites    *TestSuites
	startTime time.Time
	mux       sync.Mutex
}

func NewReport(name string) *Report {
	return &Report{
		name: name,
		suites: &TestSuites{
			ID:   runner.Dashify(name),
			Name: name,
		},
	}
}

func (r *Report) WriteToStream(w io.Writer, indentSize int) error {
	bytes, err := xml.MarshalIndent(r.suites, "", strings.Repeat(" ", indentSize))
	if err != nil {
		return errors.Wrapf(err, "failed to write JUnit report to stream")
	}

	n := 0
	for n < len(bytes) {
		n, err = w.Write(bytes)
		if err != nil {
			return errors.Wrapf(err, "failed to write %d bytes of JUnit report to stream", len(bytes)-n)
		}
	}

	return nil
}

func (r *Report) NewReportTestSuite(name string) *ReportTestSuite {
	reportTestSuite := &ReportTestSuite{
		report: r,
		suite: &TestSuite{
			ID:   runner.Dashify(name),
			Name: name,
		},
	}

	r.suites.Suites = append(r.suites.Suites, reportTestSuite.suite)
	return reportTestSuite
}

func (r *Report) SetStartTime(t time.Time) {
	r.startTime = t
}

func (r *Report) SetStopTime(t time.Time) {
	r.suites.TimeInSeconds = t.Sub(r.startTime).Seconds()
}

func (r *Report) AddTestCount(delta int) {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.suites.TestCount += delta
}

func (r *Report) AddFailureCount(delta int) {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.suites.FailureCount += delta
}

type ReportTestSuite struct {
	report    *Report
	suite     *TestSuite
	startTime time.Time
	mux       sync.Mutex
}

func (rts *ReportTestSuite) NewReportTestCase(invocation *runner.TestInvocation) *ReportTestCase {
	reportTestCase := &ReportTestCase{
		suite: rts,
		testCase: &TestCase{
			ID:   runner.Dashify(invocation.Name),
			Name: invocation.Name,
		},
	}
	rts.suite.Cases = append(rts.suite.Cases, reportTestCase.testCase)
	rts.AddTestCount(1)
	return reportTestCase
}

func (rts *ReportTestSuite) SetStartTime(t time.Time) {
	rts.startTime = t
}

func (rts *ReportTestSuite) SetStopTime(t time.Time) {
	rts.suite.TimeInSeconds = t.Sub(rts.startTime).Seconds()
}

func (rts *ReportTestSuite) AddTestCount(delta int) {
	rts.report.AddTestCount(delta)

	rts.mux.Lock()
	defer rts.mux.Unlock()
	rts.suite.TestCount += delta
}

func (rts *ReportTestSuite) AddFailureCount(delta int) {
	rts.report.AddFailureCount(delta)

	rts.mux.Lock()
	defer rts.mux.Unlock()
	rts.suite.FailureCount += delta
}

type ReportTestCase struct {
	suite     *ReportTestSuite
	testCase  *TestCase
	startTime time.Time
}

func (rtc *ReportTestCase) SetStartTime(t time.Time) {
	rtc.startTime = t
}

func (rtc *ReportTestCase) SetStopTime(t time.Time) {
	rtc.testCase.TimeInSeconds = t.Sub(rtc.startTime).Seconds()
}

func (rtc *ReportTestCase) AddFailure(failure *Failure) {
	rtc.suite.AddFailureCount(1)
	rtc.testCase.Failures = append(rtc.testCase.Failures, failure)
}
