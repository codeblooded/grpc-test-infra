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

package main

import (
	"log"
	"os"
	"text/template"
)

type config struct {
	Version         string
	DriverVersion   string
	DriverPool      string
	WorkerPool      string
	InitImagePrefix string
	ImagePrefix     string
}

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("usage: go run configure.go <config template> <output file>")
	}

	templ, err := template.ParseFiles(os.Args[1])
	if err != nil {
		log.Fatalf("could not parse template config file: %v", err)
	}

	outputFile, err := os.Create(os.Args[2])
	if err != nil {
		log.Fatalf("could not create output file: %v", err)
	}

	if err := templ.Execute(outputFile, &config{
		Version:         getEnvOrFail("VERSION"),
		DriverVersion:   getEnvOrFail("DRIVER_VERSION"),
		DriverPool:      getEnvOrFail("DRIVER_POOL"),
		WorkerPool:      getEnvOrFail("WORKER_POOL"),
		InitImagePrefix: getEnvOrFail("INIT_IMAGE_PREFIX"),
		ImagePrefix:     getEnvOrFail("IMAGE_PREFIX"),
	}); err != nil {
		log.Fatalf("could not write config file: %v", err)
	}
}

func getEnvOrFail(envVar string) string {
	val, ok := os.LookupEnv(envVar)
	if !ok {
		log.Fatalf("$%s environment variable not set", envVar)
	}
	return val
}
