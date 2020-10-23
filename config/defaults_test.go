/*
Copyright 2020 gRPC authors.

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

package config

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	grpcv1 "github.com/grpc/test-infra/api/v1"
)

var _ = Describe("Defaults", func() {
	var test *grpcv1.LoadTest
	var defaults *Defaults
	var defaultImageMap *imageMap

	BeforeEach(func() {
		test = completeLoadTest.DeepCopy()

		defaults = &Defaults{
			ComponentNamespace: "component-default",
			DriverPool:         "drivers",
			WorkerPool:         "workers-8core",
			DriverPort:         10000,
			ServerPort:         10010,
			CloneImage:         "gcr.io/grpc-fake-project/test-infra/clone",
			DriverImage:        "gcr.io/grpc-fake-project/test-infra/driver",
			Languages: []LanguageDefault{
				{
					Language:   "cxx",
					BuildImage: "l.gcr.io/google/bazel:latest",
					RunImage:   "gcr.io/grpc-fake-project/test-infra/cxx",
				},
				{
					Language:   "go",
					BuildImage: "golang:1.14",
					RunImage:   "gcr.io/grpc-fake-project/test-infra/go",
				},
				{
					Language:   "java",
					BuildImage: "java:jdk8",
					RunImage:   "gcr.io/grpc-fake-project/test-infra/java",
				},
			},
		}

		defaultImageMap = newImageMap(defaults.Languages)
	})

	Context("metadata", func() {
		It("sets default namespace when unset", func() {
			test.Namespace = ""

			namespace := "foobar-buzz"
			defaults.ComponentNamespace = namespace

			err := defaults.SetLoadTestDefaults(test)
			Expect(err).ToNot(HaveOccurred())
			Expect(test.Namespace).To(Equal(namespace))
		})

		It("does not override namespace when set", func() {
			namespace := "experimental"
			test.Namespace = namespace

			err := defaults.SetLoadTestDefaults(test)
			Expect(err).ToNot(HaveOccurred())
			Expect(test.Namespace).To(Equal(namespace))

		})
	})

	Context("driver", func() {
		var driver *grpcv1.Driver
		var component *grpcv1.Component

		BeforeEach(func() {
			driver = test.Spec.Driver
			Expect(driver).ToNot(BeNil())

			component = &driver.Component
		})

		It("sets default driver when nil", func() {
			test.Spec.Driver = nil

			err := defaults.SetLoadTestDefaults(test)
			Expect(err).ToNot(HaveOccurred())
			Expect(test.Spec.Driver).ToNot(BeNil())
		})

		It("does not override driver when set", func() {
			driver := new(grpcv1.Driver)
			test.Spec.Driver = driver

			err := defaults.SetLoadTestDefaults(test)
			Expect(err).ToNot(HaveOccurred())
			Expect(test.Spec.Driver).To(Equal(driver))
		})

		It("sets default name when unspecified", func() {
			component.Name = nil
			err := defaults.SetLoadTestDefaults(test)
			Expect(err).ToNot(HaveOccurred())
			Expect(component.Name).ToNot(BeNil())
		})

		It("sets default pool when unspecified", func() {
			component.Pool = nil
			err := defaults.SetLoadTestDefaults(test)
			Expect(err).ToNot(HaveOccurred())
			Expect(component.Pool).ToNot(BeNil())
			Expect(*component.Pool).To(Equal(defaults.DriverPool))
		})

		It("does not override pool when specified", func() {
			pool := "example-pool"
			component.Pool = &pool

			err := defaults.SetLoadTestDefaults(test)
			Expect(err).ToNot(HaveOccurred())
			Expect(component.Pool).ToNot(BeNil())
			Expect(*component.Pool).To(Equal(pool))
		})

		It("sets missing image for clone init container", func() {
			repo := "https://github.com/grpc/grpc.git"
			gitRef := "master"

			component.Clone = new(grpcv1.Clone)
			component.Clone.Repo = &repo
			component.Clone.GitRef = &gitRef
			component.Clone.Image = nil

			err := defaults.SetLoadTestDefaults(test)
			Expect(err).ToNot(HaveOccurred())
			Expect(component.Clone).ToNot(BeNil())
			Expect(component.Clone.Image).ToNot(BeNil())
			Expect(*component.Clone.Image).To(Equal(defaults.CloneImage))
		})

		It("sets missing image for build init container", func() {
			build := new(grpcv1.Build)
			build.Image = nil
			build.Command = []string{"bazel"}

			component.Language = "cxx"
			component.Build = build

			expectedBuildImage, err := defaultImageMap.buildImage(component.Language)
			Expect(err).ToNot(HaveOccurred())

			err = defaults.SetLoadTestDefaults(test)
			Expect(err).ToNot(HaveOccurred())

			Expect(component.Build).ToNot(BeNil())
			Expect(component.Build.Image).ToNot(BeNil())
			Expect(*component.Build.Image).To(Equal(expectedBuildImage))
		})

		It("errors if image for build init container cannot be inferred", func() {
			build := new(grpcv1.Build)
			build.Image = nil // no explicit image
			build.Command = []string{"make"}

			component.Language = "fortran" // unknown language
			component.Build = build

			err := defaults.SetLoadTestDefaults(test)
			Expect(err).To(HaveOccurred())
		})

		It("does not error if build init container image cannot be inferred but is set", func() {
			image := "test-image"

			build := new(grpcv1.Build)
			build.Image = &image
			build.Command = []string{"make"}

			component.Language = "fortran" // unknown language
			component.Build = build

			err := defaults.SetLoadTestDefaults(test)
			Expect(err).ToNot(HaveOccurred())

		})

		It("sets missing image for run container", func() {
			component.Language = "cxx"
			component.Run.Image = nil

			err := defaults.SetLoadTestDefaults(test)
			Expect(err).ToNot(HaveOccurred())

			Expect(component.Run.Image).ToNot(BeNil())
			Expect(*component.Run.Image).To(Equal(defaults.DriverImage))
		})

		It("does not error if run container image cannot be inferred but is set", func() {
			image := "example-image"

			component.Language = "fortran" // unknown language
			component.Run.Image = &image
			component.Run.Command = []string{"do-stuff"}

			err := defaults.SetLoadTestDefaults(test)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("server", func() {
		var server *grpcv1.Server
		var component *grpcv1.Component

		BeforeEach(func() {
			server = &test.Spec.Servers[0]
			component = &server.Component
		})

		It("sets default name when unspecified", func() {
			component.Name = nil
			err := defaults.SetLoadTestDefaults(test)
			Expect(err).ToNot(HaveOccurred())
			Expect(component.Name).ToNot(BeNil())
		})

		It("sets default pool when unspecified", func() {
			component.Pool = nil
			err := defaults.SetLoadTestDefaults(test)
			Expect(err).ToNot(HaveOccurred())
			Expect(component.Pool).ToNot(BeNil())
			Expect(*component.Pool).To(Equal(defaults.WorkerPool))
		})

		It("does not override pool when specified", func() {
			pool := "example-pool"
			component.Pool = &pool

			err := defaults.SetLoadTestDefaults(test)
			Expect(err).ToNot(HaveOccurred())
			Expect(component.Pool).ToNot(BeNil())
			Expect(*component.Pool).To(Equal(pool))
		})

		It("sets missing image for clone init container", func() {
			repo := "https://github.com/grpc/grpc.git"
			gitRef := "master"

			component.Clone = new(grpcv1.Clone)
			component.Clone.Repo = &repo
			component.Clone.GitRef = &gitRef
			component.Clone.Image = nil

			err := defaults.SetLoadTestDefaults(test)
			Expect(err).ToNot(HaveOccurred())
			Expect(component.Clone).ToNot(BeNil())
			Expect(component.Clone.Image).ToNot(BeNil())
			Expect(*component.Clone.Image).To(Equal(defaults.CloneImage))
		})

		It("sets missing image for build init container", func() {
			build := new(grpcv1.Build)
			build.Image = nil
			build.Command = []string{"bazel"}

			component.Language = "cxx"
			component.Build = build

			expectedBuildImage, err := defaultImageMap.buildImage(component.Language)
			Expect(err).ToNot(HaveOccurred())

			err = defaults.SetLoadTestDefaults(test)
			Expect(err).ToNot(HaveOccurred())

			Expect(component.Build).ToNot(BeNil())
			Expect(component.Build.Image).ToNot(BeNil())
			Expect(*component.Build.Image).To(Equal(expectedBuildImage))
		})

		It("errors if image for build init container cannot be inferred", func() {
			build := new(grpcv1.Build)
			build.Image = nil // no explicit image
			build.Command = []string{"make"}

			component.Language = "fortran" // unknown language
			component.Build = build

			err := defaults.SetLoadTestDefaults(test)
			Expect(err).To(HaveOccurred())
		})

		It("does not error if build init container image cannot be inferred but is set", func() {
			image := "test-image"

			build := new(grpcv1.Build)
			build.Image = &image
			build.Command = []string{"make"}

			component.Language = "fortran" // unknown language
			component.Build = build

			err := defaults.SetLoadTestDefaults(test)
			Expect(err).ToNot(HaveOccurred())

		})

		It("sets missing image for run container", func() {
			component.Language = "cxx"
			component.Run.Image = nil

			expectedRunImage, err := defaultImageMap.runImage(component.Language)
			Expect(err).ToNot(HaveOccurred())

			err = defaults.SetLoadTestDefaults(test)
			Expect(err).ToNot(HaveOccurred())

			Expect(component.Run.Image).ToNot(BeNil())
			Expect(*component.Run.Image).To(Equal(expectedRunImage))
		})

		It("errors if image for run container cannot be inferred", func() {
			component.Build = nil // disable build to ensure run is the problem

			component.Language = "fortran" // unknown language
			component.Run.Image = nil      // no explicit image
			component.Run.Command = []string{"do-stuff"}

			err := defaults.SetLoadTestDefaults(test)
			Expect(err).To(HaveOccurred())
		})

		It("does not error if run container image cannot be inferred but is set", func() {
			image := "example-image"

			component.Language = "fortran" // unknown language
			component.Run.Image = &image
			component.Run.Command = []string{"do-stuff"}

			err := defaults.SetLoadTestDefaults(test)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("client", func() {
		var client *grpcv1.Client
		var component *grpcv1.Component

		BeforeEach(func() {
			client = &test.Spec.Clients[0]
			component = &client.Component
		})

		It("sets default name when unspecified", func() {
			component.Name = nil
			err := defaults.SetLoadTestDefaults(test)
			Expect(err).ToNot(HaveOccurred())
			Expect(component.Name).ToNot(BeNil())
		})

		It("sets default pool when unspecified", func() {
			component.Pool = nil
			err := defaults.SetLoadTestDefaults(test)
			Expect(err).ToNot(HaveOccurred())
			Expect(component.Pool).ToNot(BeNil())
			Expect(*component.Pool).To(Equal(defaults.WorkerPool))
		})

		It("does not override pool when specified", func() {
			pool := "example-pool"
			component.Pool = &pool

			err := defaults.SetLoadTestDefaults(test)
			Expect(err).ToNot(HaveOccurred())
			Expect(component.Pool).ToNot(BeNil())
			Expect(*component.Pool).To(Equal(pool))
		})

		It("sets missing image for clone init container", func() {
			repo := "https://github.com/grpc/grpc.git"
			gitRef := "master"

			component.Clone = new(grpcv1.Clone)
			component.Clone.Repo = &repo
			component.Clone.GitRef = &gitRef
			component.Clone.Image = nil

			err := defaults.SetLoadTestDefaults(test)
			Expect(err).ToNot(HaveOccurred())
			Expect(component.Clone).ToNot(BeNil())
			Expect(component.Clone.Image).ToNot(BeNil())
			Expect(*component.Clone.Image).To(Equal(defaults.CloneImage))
		})

		It("sets missing image for build init container", func() {
			build := new(grpcv1.Build)
			build.Image = nil
			build.Command = []string{"bazel"}

			component.Language = "cxx"
			component.Build = build

			expectedBuildImage, err := defaultImageMap.buildImage(component.Language)
			Expect(err).ToNot(HaveOccurred())

			err = defaults.SetLoadTestDefaults(test)
			Expect(err).ToNot(HaveOccurred())

			Expect(component.Build).ToNot(BeNil())
			Expect(component.Build.Image).ToNot(BeNil())
			Expect(*component.Build.Image).To(Equal(expectedBuildImage))
		})

		It("errors if image for build init container cannot be inferred", func() {
			build := new(grpcv1.Build)
			build.Image = nil // no explicit image
			build.Command = []string{"make"}

			component.Language = "fortran" // unknown language
			component.Build = build

			err := defaults.SetLoadTestDefaults(test)
			Expect(err).To(HaveOccurred())
		})

		It("does not error if build init container image cannot be inferred but is set", func() {
			image := "test-image"

			build := new(grpcv1.Build)
			build.Image = &image
			build.Command = []string{"make"}

			component.Language = "fortran" // unknown language
			component.Build = build

			err := defaults.SetLoadTestDefaults(test)
			Expect(err).ToNot(HaveOccurred())

		})

		It("sets missing image for run container", func() {
			component.Language = "cxx"
			component.Run.Image = nil

			expectedRunImage, err := defaultImageMap.runImage(component.Language)
			Expect(err).ToNot(HaveOccurred())

			err = defaults.SetLoadTestDefaults(test)
			Expect(err).ToNot(HaveOccurred())

			Expect(component.Run.Image).ToNot(BeNil())
			Expect(*component.Run.Image).To(Equal(expectedRunImage))
		})

		It("errors if image for run container cannot be inferred", func() {
			component.Language = "fortran" // unknown language
			component.Run.Image = nil      // no explicit image
			component.Run.Command = []string{"do-stuff"}

			err := defaults.SetLoadTestDefaults(test)
			Expect(err).To(HaveOccurred())
		})

		It("does not error if run container image cannot be inferred but is set", func() {
			image := "example-image"

			component.Language = "fortran" // unknown language
			component.Run.Image = &image
			component.Run.Command = []string{"do-stuff"}

			err := defaults.SetLoadTestDefaults(test)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

var completeLoadTest = func() *grpcv1.LoadTest {
	cloneImage := "docker.pkg.github.com/grpc/test-infra/clone"
	cloneRepo := "https://github.com/grpc/grpc.git"
	cloneGitRef := "master"

	buildImage := "l.gcr.io/google/bazel:latest"
	buildCommand := []string{"bazel"}
	buildArgs := []string{"build", "//test/cpp/qps:qps_worker"}

	driverImage := "docker.pkg.github.com/grpc/test-infra/driver"
	runImage := "docker.pkg.github.com/grpc/test-infra/cxx"
	runCommand := []string{"bazel-bin/test/cpp/qps/qps_worker"}
	clientRunArgs := []string{"--driver_port=10000"}
	serverRunArgs := append(clientRunArgs, "--server_port=10010")

	bigQueryTable := "grpc-testing.e2e_benchmark.foobarbuzz"

	driverPool := "drivers"
	workerPool := "workers-8core"

	driverComponentName := "driver"
	serverComponentName := "server"
	clientComponentName := "client-1"

	return &grpcv1.LoadTest{
		Spec: grpcv1.LoadTestSpec{
			Driver: &grpcv1.Driver{
				Component: grpcv1.Component{
					Name:     &driverComponentName,
					Language: "cxx",
					Pool:     &driverPool,
					Run: grpcv1.Run{
						Image: &driverImage,
					},
				},
			},

			Servers: []grpcv1.Server{
				{
					Component: grpcv1.Component{
						Name:     &serverComponentName,
						Language: "cxx",
						Pool:     &workerPool,
						Clone: &grpcv1.Clone{
							Image:  &cloneImage,
							Repo:   &cloneRepo,
							GitRef: &cloneGitRef,
						},
						Build: &grpcv1.Build{
							Image:   &buildImage,
							Command: buildCommand,
							Args:    buildArgs,
						},
						Run: grpcv1.Run{
							Image:   &runImage,
							Command: runCommand,
							Args:    serverRunArgs,
						},
					},
				},
			},

			Clients: []grpcv1.Client{
				{
					Component: grpcv1.Component{
						Name:     &clientComponentName,
						Language: "cxx",
						Pool:     &workerPool,
						Clone: &grpcv1.Clone{
							Image:  &cloneImage,
							Repo:   &cloneRepo,
							GitRef: &cloneGitRef,
						},
						Build: &grpcv1.Build{
							Image:   &buildImage,
							Command: buildCommand,
							Args:    buildArgs,
						},
						Run: grpcv1.Run{
							Image:   &runImage,
							Command: runCommand,
							Args:    clientRunArgs,
						},
					},
				},
			},

			Results: &grpcv1.Results{
				BigQueryTable: &bigQueryTable,
			},

			Scenarios: []grpcv1.Scenario{
				{Name: "cpp-example-scenario"},
			},
		},
	}
}()
