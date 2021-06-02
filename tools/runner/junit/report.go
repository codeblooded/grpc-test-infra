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
	name            string
	suites          map[string]*TestSuite
	maxSuiteSeconds float64
	junitObject     *TestSuites
	mux             sync.Mutex
}

func NewReport(name string) *Report {
	return &Report{
		name:   name,
		suites: make(map[string]*TestSuite),
		junitObject: &TestSuites{
			ID:   runner.Dashify(name),
			Name: name,
		},
	}
}

func (r *Report) WriteToStream(w io.Writer, indentSize int) error {
	bytes, err := xml.MarshalIndent(r.junitObject, "", strings.Repeat(" ", indentSize))
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

func (r *Report) RecordSuiteStart(name string, time time.Time) {
	suite := r.getOrCreateTestSuite(name)
	suite.startTime = now()

	r.mux.Lock()
	defer r.mux.Unlock()
	if r.junitObject.startTime.IsZero() {
		r.junitObject.startTime = suite.startTime
	}
}

func (r *Report) RecordSuiteStop(name string, time time.Time) {
	suite := r.getOrCreateTestSuite(name)
	suite.TimeInSeconds = time.Sub(suite.startTime).Seconds()

	r.mux.Lock()
	defer r.mux.Unlock()
	if suite.TimeInSeconds > r.maxSuiteSeconds {
		r.maxSuiteSeconds = suite.TimeInSeconds
	}
}

func (r *Report) RecordTestStart(invocation *runner.TestInvocation) {
	suite := r.getOrCreateTestSuite(invocation.QueueName)
	testCase := r.getOrCreateTestCase(invocation)
	testCase.startTime = invocation.StartTime

	r.mux.Lock()
	defer r.mux.Unlock()
	r.junitObject.TestCount += 1
	suite.TestCount += 1
}

func (r *Report) RecordTestStop(invocation *runner.TestInvocation) {
	testCase := r.getOrCreateTestCase(invocation)
	testCase.TimeInSeconds = invocation.StopTime.Sub(invocation.StartTime).Seconds()
}

func (r *Report) RecordFailure(invocation *runner.TestInvocation, failure *Failure) {
	suite := r.getOrCreateTestSuite(invocation.QueueName)
	testCase := r.getOrCreateTestCase(invocation)
	testCase.Failures = append(testCase.Failures, failure)

	r.mux.Lock()
	defer r.mux.Unlock()
	r.junitObject.FailureCount += 1
	suite.FailureCount += 1
}

func (r *Report) getOrCreateTestSuite(qName string) *TestSuite {
	suite, ok := r.suites[qName]
	if !ok {
		suite = &TestSuite{
			ID:       runner.Dashify(qName),
			Name:     qName,
			casesMap: make(map[string]*TestCase),
		}

		func() {
			r.mux.Lock()
			defer r.mux.Unlock()
			r.suites[qName] = suite
			r.junitObject.Suites = append(r.junitObject.Suites, suite)
		}()
	}

	return suite
}

func (r *Report) getOrCreateTestCase(invocation *runner.TestInvocation) *TestCase {
	suite := r.getOrCreateTestSuite(invocation.QueueName)

	testName := invocation.Name
	testCase, ok := suite.casesMap[testName]
	if !ok {
		testCase = &TestCase{
			ID:        runner.Dashify(invocation.Name),
			Name:      testName,
			startTime: now(),
		}

		suite.casesMap[testName] = testCase
		suite.Cases = append(suite.Cases, testCase)
	}

	return testCase
}
