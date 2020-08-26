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

package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/pkg/defaults"
)

// reconcileTimeout specifies the maximum amount of time any set of API
// requests should take for a single invocation of the Reconcile method.
const reconcileTimeout = 1 * time.Minute

// cloneInitContainer holds the name of the init container that obtains a copy
// of the code at a specific point in time.
const cloneInitContainer = "clone"

// buildInitContainer holds the name of the init container that assembles a
// binary or other bundle required to run the tests.
const buildInitContainer = "build"

// runContainer holds the name of the main container where the test is executed.
const runContainer = "run"

// scenarioMountPath specifies where the JSON file with the scenario should be
// mounted in the driver container.
const scenarioMountPath = "/src/scenarios"

// scenarioFileEnv specifies the name of an env variable that specifies the path
// to a JSON file with a scenario.
const scenarioFileEnv = "SCENARIO_FILE"

// CloneRepoEnv specifies the name of the env variable that contains the git
// repository to clone.
const CloneRepoEnv = "CLONE_REPO"

// CloneGitRefEnv specifies the name of the env variable that contains the
// commit, tag or branch to checkout after cloning a git repository.
const CloneGitRefEnv = "CLONE_GIT_REF"

// LoadTestReconciler reconciles a LoadTest object
type LoadTestReconciler struct {
	client.Client
	Defaults *defaults.Defaults
	Log      logr.Logger
	Scheme   *runtime.Scheme
}

// +kubebuilder:rbac:groups=e2etest.grpc.io,resources=loadtests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=e2etest.grpc.io,resources=loadtests/status,verbs=get;update;patch

// Reconcile attempts to bring the current state of the load test into agreement
// with its declared spec. This may mean provisioning resources, doing nothing
// or handling the termination of its pods.
func (r *LoadTestReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	var err error

	log := r.Log.WithValues("loadtest", req.NamespacedName)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Fetch the current state of the world.

	nodes := new(corev1.NodeList)
	if err = r.List(ctx, nodes); err != nil {
		log.Error(err, "failed to list nodes")

		// attempt to requeue with exponential back-off
		return ctrl.Result{Requeue: true}, err
	}

	pods := new(corev1.PodList)
	if err = r.List(ctx, pods, client.InNamespace(req.Namespace)); err != nil {
		log.Error(err, "failed to list pods", "namespace", req.Namespace)

		// attempt to requeue with exponential back-off
		return ctrl.Result{Requeue: true}, err
	}

	loadtests := new(grpcv1.LoadTestList)
	if err = r.List(ctx, loadtests); err != nil {
		log.Error(err, "failed to list loadtests")

		// attempt to requeue with exponential back-off
		return ctrl.Result{Requeue: true}, err
	}

	fetchedLoadTest := new(grpcv1.LoadTest)
	if err = r.Get(ctx, req.NamespacedName, fetchedLoadTest); err != nil {
		log.Error(err, "failed to get loadtest", "name", req.NamespacedName)

		// do not requeue, may have been garbage collected
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	loadtest := fetchedLoadTest.DeepCopy()
	if err = r.Defaults.SetLoadTestDefaults(loadtest); err != nil {
		log.Error(err, "failed to find defaults for loadtest")

		// do not requeue, something has gone horribly wrong
		return ctrl.Result{}, err
	}
	if err = r.Update(ctx, loadtest); err != nil {
		log.Error(err, "failed to update loadtest with defaults")
		return ctrl.Result{Requeue: true}, err
	}

	// Check if the loadtest has terminated.

	// TODO: Do nothing if the loadtest has terminated.

	// Check the status of any running pods.

	// TODO: Add method to get list of owned pods and method to check their status.

	// Create any missing pods that the loadtest needs.

	var pod *corev1.Pod
	missingPods := checkMissingPods(loadtest, pods)

	if len(missingPods.Servers) > 0 {
		pod, err = newServerPod(loadtest, &missingPods.Servers[0].Component)
	} else if len(missingPods.Clients) > 0 {
		pod, err = newClientPod(loadtest, &missingPods.Clients[0].Component)
	} else if missingPods.Driver != nil {
		pod, err = newDriverPod(loadtest, &missingPods.Driver.Component)
	}

	if err != nil {
		log.Error(err, "could not initialize new pod", "pod", pod)
	}
	if pod != nil {
		if err = ctrl.SetControllerReference(loadtest, pod, r.Scheme); err != nil {
			log.Error(err, "could not set controller reference on pod", "pod", pod)
			return ctrl.Result{Requeue: true}, err
		}

		if err = r.Create(ctx, pod); err != nil {
			log.Error(err, "could not create new pod", "pod", pod)
			return ctrl.Result{Requeue: true}, err
		}
	}

	// TODO: Add logic to schedule the next missing pod.

	// PLACEHOLDERS!
	_ = nodes
	_ = pods
	_ = loadtests
	_ = loadtest
	return ctrl.Result{}, nil
}

// LoadTestMissing categorize missing components based on their roles at specific
// moment. The struct is a wrapper to help us get role information associate
// with components.
type LoadTestMissing struct {
	// Driver is the component that orchestrates the test. If Driver is not set
	// that means we already have the Driver running.
	Driver *grpcv1.Driver `json:"driver,omitempty"`

	// Servers are a list of components that receive traffic from. The list
	// indicates the Servers still in need.
	Servers []grpcv1.Server `json:"servers,omitempty"`

	// Clients are a list of components that send traffic to servers. The list
	// indicates the Clients still in need.
	Clients []grpcv1.Client `json:"clients,omitempty"`
}

// checkMissingPods attempts to check if any required component is missing from
// the current load test. It takes reference of the current load test and a pod
// list that contains all running pods at the moment, returning all missing
// components required from the current load test with their roles.
func checkMissingPods(currentLoadTest *grpcv1.LoadTest, allRunningPods *corev1.PodList) *LoadTestMissing {

	currentMissing := &LoadTestMissing{Servers: []grpcv1.Server{}, Clients: []grpcv1.Client{}}

	requiredClientMap := make(map[string]*grpcv1.Client)
	requiredServerMap := make(map[string]*grpcv1.Server)
	foundDriver := false

	for i := 0; i < len(currentLoadTest.Spec.Clients); i++ {
		requiredClientMap[*currentLoadTest.Spec.Clients[i].Name] = &currentLoadTest.Spec.Clients[i]
	}
	for i := 0; i < len(currentLoadTest.Spec.Servers); i++ {
		requiredServerMap[*currentLoadTest.Spec.Servers[i].Name] = &currentLoadTest.Spec.Servers[i]
	}

	if allRunningPods != nil {

		for _, eachPod := range allRunningPods.Items {

			if eachPod.Labels == nil {
				continue
			}

			loadTestLabel := eachPod.Labels[defaults.LoadTestLabel]
			roleLabel := eachPod.Labels[defaults.RoleLabel]
			componentNameLabel := eachPod.Labels[defaults.ComponentNameLabel]

			if loadTestLabel != currentLoadTest.Name {
				continue
			}
			if roleLabel == defaults.DriverRole {
				if *currentLoadTest.Spec.Driver.Component.Name == componentNameLabel {
					foundDriver = true
				}
			} else if roleLabel == defaults.ClientRole {
				if _, ok := requiredClientMap[componentNameLabel]; ok {
					delete(requiredClientMap, componentNameLabel)
				}
			} else if roleLabel == defaults.ServerRole {
				if _, ok := requiredServerMap[componentNameLabel]; ok {
					delete(requiredServerMap, componentNameLabel)
				}
			}
		}
	}

	for _, eachMissingClient := range requiredClientMap {
		currentMissing.Clients = append(currentMissing.Clients, *eachMissingClient)
	}

	for _, eachMissingServer := range requiredServerMap {
		currentMissing.Servers = append(currentMissing.Servers, *eachMissingServer)
	}

	if !foundDriver {
		currentMissing.Driver = currentLoadTest.Spec.Driver
	}

	return currentMissing
}

// SetupWithManager configures a controller-runtime manager.
func (r *LoadTestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&grpcv1.LoadTest{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}

// newClientPod creates a client given a load test and a reference to its
// component. It returns an error if a pod cannot be constructed.
func newClientPod(loadtest *grpcv1.LoadTest, component *grpcv1.Component) (*corev1.Pod, error) {
	pod, err := newPod(loadtest, component, defaults.ClientRole)
	if err != nil {
		return nil, err
	}

	addDriverPort(&pod.Spec.Containers[0])

	return pod, nil
}

// scenarioVolumeName accepts the name of a ConfigMap with a scenario and
// generates a name for a volume.
func scenarioVolumeName(scenario string) string {
	return fmt.Sprintf("scenario-%s", scenario)
}

// newScenarioVolume accepts the name of a scenario ConfigMap and returns a
// volume that can mount it.
func newScenarioVolume(scenario string) corev1.Volume {
	return corev1.Volume{
		Name: scenarioVolumeName(scenario),
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: scenario,
				},
			},
		},
	}
}

// newScenarioVolumeMount accepts the name of a scenario ConfigMap and returns a
// volume mount that will place it at /src/scenarios.
func newScenarioVolumeMount(scenario string) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      scenarioVolumeName(scenario),
		MountPath: scenarioMountPath,
		ReadOnly:  true,
	}
}

// newScenarioEnvVar accepts the name of a scenario ConfigMap and returns a
// an environment variable used to locate the scenario in a mounted volume.
func newScenarioFileEnvVar(scenario string) corev1.EnvVar {
	scenarioFile := strings.ReplaceAll(scenario, "-", "_") + ".json"
	return corev1.EnvVar{
		Name:  scenarioFileEnv,
		Value: scenarioMountPath + "/" + scenarioFile,
	}
}

// newDriverPod creates a driver given a load test and a reference to its
// component. It returns an error if a pod cannot be constructed.
func newDriverPod(loadtest *grpcv1.LoadTest, component *grpcv1.Component) (*corev1.Pod, error) {
	pod, err := newPod(loadtest, component, defaults.DriverRole)
	if err != nil {
		return nil, err
	}

	podSpec := &pod.Spec
	testSpec := &loadtest.Spec

	// TODO: Avoid referencing containers by index, use names
	testContainer := &podSpec.Containers[0]

	addDriverPort(testContainer)

	// TODO: Handle more than 1 scenario
	if len(testSpec.Scenarios) > 0 {
		scenario := testSpec.Scenarios[0].Name
		podSpec.Volumes = append(podSpec.Volumes, newScenarioVolume(scenario))
		testContainer.VolumeMounts = append(testContainer.VolumeMounts, newScenarioVolumeMount(scenario))
		testContainer.Env = append(testContainer.Env, newScenarioFileEnvVar(scenario))
	}

	return pod, nil
}

// addDriverPort decorates a container with an additional port for the driver.
func addDriverPort(container *corev1.Container) {
	container.Ports = append(container.Ports, newContainerPort("driver", 10000))
}

// addServerPort decorates a container with an additional port for the server.
func addServerPort(container *corev1.Container) {
	container.Ports = append(container.Ports, newContainerPort("server", 10010))
}

// newContainerPort creates a Kubernetes ContainerPort object with the provided
// name and portNumber. The name should uniquely identify the port and the port
// number must be within the standard port range. The protocol is assumed to be
// TCP.
func newContainerPort(name string, portNumber int32) corev1.ContainerPort {
	return corev1.ContainerPort{
		Name:          name,
		Protocol:      corev1.ProtocolTCP,
		ContainerPort: portNumber,
	}
}

// newServerPod creates a server given a load test and a reference to its
// component. It returns an error if a pod cannot be constructed.
func newServerPod(loadtest *grpcv1.LoadTest, component *grpcv1.Component) (*corev1.Pod, error) {
	pod, err := newPod(loadtest, component, defaults.ServerRole)
	if err != nil {
		return nil, err
	}

	addDriverPort(&pod.Spec.Containers[0])
	addServerPort(&pod.Spec.Containers[0])

	return pod, nil
}

// newCloneContainer constructs a container given a grpcv1.Clone pointer. If
// the pointer is nil, an empty container is returned.
func newCloneContainer(clone *grpcv1.Clone) corev1.Container {
	if clone == nil {
		return corev1.Container{}
	}

	var env []corev1.EnvVar

	if clone.Repo != nil {
		env = append(env, corev1.EnvVar{Name: CloneRepoEnv, Value: *clone.Repo})
	}

	if clone.GitRef != nil {
		env = append(env, corev1.EnvVar{Name: CloneGitRefEnv, Value: *clone.GitRef})
	}

	return corev1.Container{
		Name:  cloneInitContainer,
		Image: safeStrUnwrap(clone.Image),
		Env:   env,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "workspace",
				MountPath: "/src/workspace",
			},
		},
	}
}

// newBuildContainer constructs a container given a grpcv1.Build pointer. If
// the pointer is nil, an empty container is returned.
func newBuildContainer(build *grpcv1.Build) corev1.Container {
	if build == nil {
		return corev1.Container{}
	}

	return corev1.Container{
		Name:       buildInitContainer,
		Image:      *build.Image,
		Command:    build.Command,
		Args:       build.Args,
		Env:        build.Env,
		WorkingDir: "/src/workspace",
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "workspace",
				MountPath: "/src/workspace",
			},
		},
	}
}

// newRunContainer constructs a container given a grpcv1.Run object.
func newRunContainer(run grpcv1.Run) corev1.Container {
	return corev1.Container{
		Name:    runContainer,
		Image:   *run.Image,
		Command: run.Command,
		Args:    run.Args,
		Env:     run.Env,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "workspace",
				MountPath: "/src/workspace",
			},
		},
	}
}

// newPod constructs a Kubernetes pod.
func newPod(loadtest *grpcv1.LoadTest, component *grpcv1.Component, role string) (*corev1.Pod, error) {
	var initContainers []corev1.Container

	if component.Clone != nil {
		initContainers = append(initContainers, newCloneContainer(component.Clone))
	}

	if component.Build != nil {
		initContainers = append(initContainers, newBuildContainer(component.Build))
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-%s", loadtest.Name, role, *component.Name),
			Namespace: loadtest.Namespace,
			Labels: map[string]string{
				defaults.LoadTestLabel:      loadtest.Name,
				defaults.RoleLabel:          role,
				defaults.ComponentNameLabel: *component.Name,
			},
		},
		Spec: corev1.PodSpec{
			NodeSelector: map[string]string{
				"pool": *component.Pool,
			},
			InitContainers: initContainers,
			Containers:     []corev1.Container{newRunContainer(component.Run)},
			RestartPolicy:  corev1.RestartPolicyNever,
			Affinity: &corev1.Affinity{
				PodAntiAffinity: &corev1.PodAntiAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
						{
							LabelSelector: &metav1.LabelSelector{
								MatchExpressions: []metav1.LabelSelectorRequirement{
									{
										Key:      "generated",
										Operator: metav1.LabelSelectorOpExists,
									},
								},
							},
							TopologyKey: "kubernetes.io/hostname",
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "workspace",
				},
			},
		},
	}, nil
}

// safeStrUnwrap accepts a string pointer, returning the dereferenced string or
// an empty string if the pointer is nil.
func safeStrUnwrap(strPtr *string) string {
	if strPtr == nil {
		return ""
	}

	return *strPtr
}
