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

// Package runner contains code for a test runner that can run a list of
// load tests, wait for them to complete, and report on the results.
package runner

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	clientset "github.com/grpc/test-infra/clientset"
)

// AfterIntervalFunction returns a function that stops for a time interval.
// This function is provided so it can be replaced with a fake for testing.
func AfterIntervalFunction(d time.Duration) func() {
	return func() {
		<-time.After(d)
	}
}

// Runner contains the information needed to run multiple sets of LoadTests.
type Runner struct {
	// loadTestGetter interacts with the cluster to create, get and delete
	// LoadTests.
	loadTestGetter clientset.LoadTestGetter
	// afterInterval stops for a set time interval before returning.
	// It is used to set a polling interval.
	afterInterval func()
	// retries is the number of times to retry create and poll operations before
	// failing each test.
	retries uint
}

// NewRunner creates a new Runner object.
func NewRunner(loadTestGetter clientset.LoadTestGetter, afterInterval func(), retries uint) *Runner {
	return &Runner{
		loadTestGetter: loadTestGetter,
		afterInterval:  afterInterval,
		retries:        retries,
	}
}

// Run runs a set of LoadTests at a given concurrency level.
func (r *Runner) Run(qName string, configs []*grpcv1.LoadTest, logger Logger, concurrencyLevel int, done chan string) {
	var count, n int

	logger.QueueStarted(qName)
	defer logger.QueueStopped(qName)

	testDone := make(chan *TestInvocation)
	for _, config := range configs {
		for n >= concurrencyLevel {
			invocation := <-testDone
			invocation.StopTime = time.Now()
			logger.TestStopped(invocation)
			n--
			count++
		}
		n++
		invocation := NewTestInvocation(qName, count, config)
		invocation.StartTime = time.Now()
		logger.TestStarted(invocation)
		go r.runTest(invocation, logger, testDone)
	}
	for n > 0 {
		invocation := <-testDone
		invocation.StopTime = time.Now()
		logger.TestStopped(invocation)
		n--
		count++
	}

	done <- qName
}

// runTest creates a single LoadTest and monitors it to completion.
func (r *Runner) runTest(invocation *TestInvocation, logger Logger, done chan<- *TestInvocation) {
	config := invocation.Config
	var s, status string
	var retries uint

	for {
		loadTest, err := r.loadTestGetter.Create(config, metav1.CreateOptions{})
		if err != nil {
			if retries < r.retries {
				retries++
				logger.Info(invocation, "Failed to create test, scheduling retry %d/%d: %v", retries, r.retries, err)
				r.afterInterval()
				continue
			}
			logger.Error(invocation, "Error creating the test", "Aborting after %d retries to create test %s: %v", r.retries, invocation.Name, err)
			done <- invocation
			return
		}
		retries = 0
		invocation.Config.Status = loadTest.Status
		logger.Info(invocation, "Created test %s", invocation.Name)
		break
	}

	for {
		loadTest, err := r.loadTestGetter.Get(config.Name, metav1.GetOptions{})
		if err != nil {
			if retries < r.retries {
				retries++
				logger.Info(invocation, "Failed to poll test, scheduling retry %d/%d: %v", retries, r.retries, err)
				r.afterInterval()
				continue
			}
			logger.Error(invocation, "Error polling the test", "Aborting after %d retries to poll test %s: %v", r.retries, invocation.Name, err)
			done <- invocation
			return
		}
		retries = 0
		config.Status = loadTest.Status
		s = status
		status = statusString(config)
		switch {
		case loadTest.Status.State.IsTerminated():
			if status != "Succeeded" {
				logger.Error(invocation, "Test failed", "Test failed with reason %q: %v", loadTest.Status.Reason, loadTest.Status.Message)
			} else {
				logger.Info(invocation, "Test terminated with a status of %q", status)
			}
			done <- invocation
			return
		case loadTest.Status.State == grpcv1.Running:
			logger.Info(invocation, "%s", status)
			r.afterInterval()
		default:
			if s != status {
				logger.Info(invocation, "%s", status)
			}
			// Use a longer polling interval for tests that have not started.
			r.afterInterval()
			r.afterInterval()
		}
	}
}

type TestInvocation struct {
	QueueName string
	Index     int
	Name      string
	Config    *grpcv1.LoadTest
	StartTime time.Time
	StopTime  time.Time
}

func NewTestInvocation(qName string, index int, config *grpcv1.LoadTest) *TestInvocation {
	return &TestInvocation{
		QueueName: qName,
		Index:     index,
		Name:      nameString(config),
		Config:    config,
	}
}

func (id TestInvocation) String() string {
	return id.Name
}

// nameString returns a string to represent the test name in logs.
// This string consists of two names: (1) the test name in the LoadTest
// metadata, (2) a test name derived from the prefix, scenario and uniquifier
// (if these elements are present in labels and annotations). This is a
// workaround for the fact that we cannot use the second name in the metadata.
// The LoadTest name is currently used as a label in pods, to refer back to the
// correspondingLoadTest (instead of the LoadTest UID). Labels are limited to
// 63 characters, while names themselves can go up to 253.
func nameString(config *grpcv1.LoadTest) string {
	var prefix, scenario string
	var ok bool
	if prefix, ok = config.Labels["prefix"]; !ok {
		return config.Name
	}
	if scenario, ok = config.Annotations["scenario"]; !ok {
		return config.Name
	}
	elems := []string{prefix}
	if scenario != "" {
		elems = append(elems, strings.Split(scenario, "_")...)
	}
	if uniquifier := config.Annotations["uniquifier"]; uniquifier != "" {
		elems = append(elems, uniquifier)
	}
	name := strings.Join(elems, "-")
	if name == config.Name {
		return config.Name
	}
	return fmt.Sprintf("%s [%s]", name, config.Name)
}

// statusString returns a string to represent the test status in logs.
// The string consists of state, reason and message (each omitted if empty).
func statusString(config *grpcv1.LoadTest) string {
	s := []string{string(config.Status.State)}
	if reason := strings.TrimSpace(config.Status.Reason); reason != "" {
		s = append(s, reason)
	}
	if message := strings.TrimSpace(config.Status.Message); message != "" {
		s = append(s, message)
	}
	return strings.Join(s, "; ")
}

// Dashify returns the input string where all whitespace and underscore
// characters have been replaced by dashes and, aside from dashes, only
// alphanumeric characters remain.
func Dashify(str string) string {
	// TODO: Move this into another shared package.
	b := strings.Builder{}
	for _, rune := range str {
		if string(rune) == "_" || unicode.IsSpace(rune) {
			b.WriteString("-")
		} else if string(rune) == "-" || unicode.IsLetter(rune) || unicode.IsNumber(rune) {
			b.WriteRune(rune)
		}
	}
	return b.String()
}
