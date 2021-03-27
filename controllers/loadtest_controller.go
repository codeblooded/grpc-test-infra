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
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/config"
	"github.com/grpc/test-infra/podbuilder"
	"github.com/grpc/test-infra/status"
)

var (
	errCacheSync       = errors.New("failed to sync cache")
	errNonexistentPool = errors.New("pool does not exist")
)

// setControllerReference is a method stub from controller-runtime. It allows us
// to mock conditions where setting the controller reference fails in tests.
var setControllerReference = ctrl.SetControllerReference

// LoadTestReconciler reconciles a LoadTest object
type LoadTestReconciler struct {
	client.Client
	mgr ctrl.Manager

	// Defaults provide a configuration to the controller and also specify
	// default values for fields in tests.
	Defaults *config.Defaults

	// Log is a generic V-level logger.
	Log logr.Logger

	// Scheme is a struct capable of mapping types, performing serializations
	// and other Kubernetes API essentials.
	Scheme *runtime.Scheme

	// Timeout is a near-maximum time for each reconciliation.
	Timeout time.Duration

	// The following fields are functions which match the signatures of the
	// client.Client methods. Using these fields allows us to stub out their
	// implementations for unit testing.

	create       func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error
	get          func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error
	list         func(ctx context.Context, list runtime.Object, opts ...client.ListOption) error
	update       func(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error
	updateStatus func(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error
	delete       func(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error
}

// UserError is an error with the test configuration or test itself. It provides
// fields that are useful for updating the status of the test. It is not related
// to a problem encountered by the controller. thus the controller is not
// expected to retry the operation when it is encountered.
type UserError struct {
	// Pascal-case string that provides the reason for the error in a few words.
	// Like all Kubernetes reason strings, this is considered party of the API
	// and is safe to compare against.
	Reason string

	// User-legible string that provides a description of the encountered error.
	Message string

	// Provides access any nested error, if applicable.
	WrappedError error
}

func (ue *UserError) Error() string {
	var errStr []string
	if ue.Reason != "" {
		errStr = append(errStr, fmt.Sprintf("[%s]", ue.Reason))
	}
	if ue.Message != "" {
		errStr = append(errStr, ue.Message)
	}
	if ue.WrappedError != nil {
		errStr = append(errStr, ue.WrappedError.Error())
	}
	return strings.Join(errStr, " ")
}

// ControllerError is an unexpected error that occured during the reconciliation
// of a test. This may be an error with an operation in the controller itself,
// or it may be a problem with Kubernetes. These errors are not believed to be
// related to the test configuration or test itself. For this reason, the
// controller is expected to retry the reconcilation.
type ControllerError struct {
	// Time to wait before retrying the operation. A zero value implies the
	// retry should occur as soon as possible.
	RetryDelay time.Duration

	// User-legible string that provides a description of the encountered error.
	Message string

	// Provides access any nested error, if applicable.
	WrappedError error
}

func (ce *ControllerError) Error() string {
	return fmt.Sprintf("%s (retry in %ds)", ce.WrappedError, ce.RetryDelay)
}

var (
	userErrorType       = fmt.Sprintf("%T", UserError{})
	controllerErrorType = fmt.Sprintf("%T", ControllerError{})
)

// +kubebuilder:rbac:groups=e2etest.grpc.io,resources=loadtests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=e2etest.grpc.io,resources=loadtests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods/status,verbs=get
// +kubebuilder:rbac:groups="",resource00s=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=nodes/status,verbs=get

// Reconcile attempts to bring the current state of the load test into agreement
// with its declared spec. This may mean provisioning resources, doing nothing
// or handling the termination of its pods.
func (r *LoadTestReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	var ctx context.Context
	var cancel context.CancelFunc
	var err error
	log := r.Log.WithValues("loadtest", req.NamespacedName)

	if r.Timeout == 0 {
		ctx, cancel = context.WithCancel(context.Background())
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), r.Timeout)
	}
	defer cancel()

	rawTest := new(grpcv1.LoadTest)
	if err = r.get(ctx, req.NamespacedName, rawTest); err != nil {
		log.Error(err, "failed to get test", "name", req.NamespacedName)
		err = client.IgnoreNotFound(err)
		return ctrl.Result{Requeue: err != nil}, err
	}

	testTTL := time.Duration(rawTest.Spec.TTLSeconds) * time.Second
	testTimeout := time.Duration(rawTest.Spec.TimeoutSeconds) * time.Second

	if testTimeout > testTTL {
		log.Info("testTTL is less than testTimeout", "testTimeout", testTimeout, "testTTL", testTTL)
	}

	if rawTest.Status.State.IsTerminated() {
		if time.Now().Sub(rawTest.Status.StartTime.Time) >= testTTL {
			log.Info("test expired, deleting", "startTime", rawTest.Status.StartTime, "testTTL", testTTL)
			if err = r.delete(ctx, rawTest); err != nil {
				log.Error(err, "fail to delete test")
				return ctrl.Result{Requeue: true}, err
			}
		}
		return ctrl.Result{Requeue: false}, nil
	}

	// TODO(codeblooded): Consider moving this to a mutating webhook
	test := rawTest.DeepCopy()

	handleError := func(err error, message string, keysAndValues ...interface{}) (ctrl.Result, error) {
		switch e := err.(type) {
		case *UserError:
			log.Error(err, message, append(keysAndValues,
				"errorType", userErrorType,
				"wrappedErrorType", fmt.Sprintf("%T", e.WrappedError),
			)...)
			test.Status.State = grpcv1.Errored
			test.Status.Reason = e.Reason
			test.Status.Message = e.Message
			updateErr := r.updateStatus(ctx, test)
			if updateErr != nil {
				log.Error(updateErr, "failed to update test status after user error",
					"previousUserError", e,
					"errorType", controllerErrorType)
			}
			return ctrl.Result{Requeue: true}, updateErr
		case *ControllerError:
			log.Error(err, message, append(keysAndValues,
				"retryDelay", e.RetryDelay,
				"errorType", controllerErrorType,
				"wrappedErrorType", fmt.Sprintf("%T", e.WrappedError),
			)...)
			if e.RetryDelay > 0 {
				return ctrl.Result{RequeueAfter: e.RetryDelay}, e
			}
			return ctrl.Result{Requeue: true}, e
		default:
			log.Error(err, message, append(keysAndValues, "errorType", fmt.Sprintf("%T", e))...)
			return ctrl.Result{Requeue: true}, e
		}
	}

	if err = r.Defaults.SetLoadTestDefaults(test); err != nil {
		return handleError(
			&UserError{
				Reason:       grpcv1.FailedSettingDefaultsError,
				WrappedError: err,
			},
			fmt.Sprintf("failed to set defaults for missing fields on the test: %v", err),
			"defaults", r.Defaults,
			"testSpec", test.Spec,
		)
	}
	if !reflect.DeepEqual(rawTest, test) {
		if err = r.update(ctx, test); err != nil {
			return handleError(
				&ControllerError{
					WrappedError: err,
				},
				"failed to update test after setting defaults for missing fields",
			)
		}
	}

	if err := r.CreateConfigMapIfMissing(ctx, test); err != nil {
		return handleError(err, "failed to create a scenario config map", "testScenario", test.Spec.ScenariosJSON)
	}

	pods := new(corev1.PodList)
	if err = r.list(ctx, pods, client.InNamespace(req.Namespace)); err != nil {
		return handleError(err, "failed to list pods", "namespace", req.Namespace)
	}
	ownedPods := status.PodsForLoadTest(test, pods.Items)

	previousStatus := test.Status
	test.Status = status.ForLoadTest(test, ownedPods)
	if err = r.updateStatus(ctx, test); err != nil {
		return handleError(err, "failed to update test status")
	}

	missingPods := status.CheckMissingPods(test, ownedPods)
	if !missingPods.IsEmpty() {
		if !r.mgr.GetCache().WaitForCacheSync(ctx.Done()) {
			return handleError(errCacheSync, "could not invalidate the cache which is required to gang schedule")
		}

		nodes := new(corev1.NodeList)
		if err = r.list(ctx, nodes); err != nil {
			return handleError(err, "failed to list nodes")
		}

		clusterInfo := CurrentClusterInfo(nodes, pods, r.Defaults.DefaultPoolLabels, log)
		adjustAvailabilityForDefaults(clusterInfo, missingPods,
			defaultAdjustment{
				DefaultPoolKey: status.DefaultClientPool,
				Role:           config.ClientRole,
			},
			defaultAdjustment{
				DefaultPoolKey: status.DefaultDriverPool,
				Role:           config.DriverRole,
			},
			defaultAdjustment{
				DefaultPoolKey: status.DefaultServerPool,
				Role:           config.ServerRole,
			},
		)

		for pool, requiredNodeCount := range missingPods.NodeCountByPool {
			availableNodeCount, ok := clusterInfo.AvailabilityForPool(pool)
			if !ok {
				log.Error(errNonexistentPool, "requested pool does not exist and cannot be considered when scheduling", "requestedPool", pool)
				test.Status.State = grpcv1.Errored
				test.Status.Reason = grpcv1.PoolError
				test.Status.Message = fmt.Sprintf("requested pool %q does not exist", pool)
				if updateErr := r.updateStatus(ctx, test); updateErr != nil {
					log.Error(updateErr, "failed to update status after failure due to requesting nodes from a nonexistent pool")
				}
				return ctrl.Result{Requeue: false}, nil
			}

			if requiredNodeCount > availableNodeCount {
				log.Info("cannot schedule test: inadequate availability for pool", "pool", pool, "requiredNodeCount", requiredNodeCount, "availableNodeCount", availableNodeCount)
				return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
			}
		}

		builder := podbuilder.New(r.Defaults, test)
		createPod := func(pod *corev1.Pod) (*ctrl.Result, error) {
			if err = setControllerReference(test, pod, r.Scheme); err != nil {
				log.Error(err, "could not set controller reference on pod, pod will not be garbage collected", "pod", pod)
				return &ctrl.Result{Requeue: true}, err
			}

			if err = r.create(ctx, pod); err != nil {
				log.Error(err, "could not create new pod", "pod", pod)
				return &ctrl.Result{Requeue: true}, err
			}

			return nil, nil
		}

		for i := range missingPods.Servers {
			logWithServer := log.WithValues("server", missingPods.Servers[i])

			pod, err := builder.PodForServer(&missingPods.Servers[i])
			if err != nil {
				logWithServer.Error(err, "failed to construct a pod struct for supplied server struct")
				test.Status.State = grpcv1.Errored
				test.Status.Reason = grpcv1.ConfigurationError
				test.Status.Message = fmt.Sprintf("failed to construct a pod for server at index %d: %v", i, err)
				if updateErr := r.updateStatus(ctx, test); updateErr != nil {
					logWithServer.Error(updateErr, "failed to update status after failure to construct a pod for server")
				}
				return ctrl.Result{Requeue: false}, nil
			}

			// TODO: Better error checking in these blocks
			if pool, ok := clusterInfo.DefaultPoolForRole(config.ServerRole); ok && missingPods.Servers[i].Pool == nil {
				pod.Labels[config.PoolLabel] = pool
			} else {
				pod.Labels[config.PoolLabel] = *missingPods.Servers[i].Pool
			}

			result, err := createPod(pod)
			if result != nil && !kerrors.IsAlreadyExists(err) {
				logWithServer.Error(err, "failed to create pod for server")
				test.Status.State = grpcv1.Errored
				test.Status.Reason = grpcv1.KubernetesError
				test.Status.Message = fmt.Sprintf("failed to create pod for server at index %d: %v", i, err)
				if updateErr := r.updateStatus(ctx, test); updateErr != nil {
					logWithServer.Error(updateErr, "failed to update status after failure to create pod for server")
				}
				return *result, err
			}
		}
		for i := range missingPods.Clients {
			logWithClient := log.WithValues("client", missingPods.Clients[i])

			pod, err := builder.PodForClient(&missingPods.Clients[i])
			if err != nil {
				logWithClient.Error(err, "failed to construct a pod struct for supplied client struct")
				test.Status.State = grpcv1.Errored
				test.Status.Reason = grpcv1.ConfigurationError
				test.Status.Message = fmt.Sprintf("failed to construct a pod for client at index %d: %v", i, err)
				if updateErr := r.updateStatus(ctx, test); updateErr != nil {
					logWithClient.Error(updateErr, "failed to update status after failure to construct a pod for client")
				}
				return ctrl.Result{Requeue: false}, nil
			}

			if pool, ok := clusterInfo.DefaultPoolForRole(config.ClientRole); ok && missingPods.Clients[i].Pool == nil {
				pod.Labels[config.PoolLabel] = pool
			} else {
				pod.Labels[config.PoolLabel] = *missingPods.Clients[i].Pool
			}

			result, err := createPod(pod)
			if result != nil && !kerrors.IsAlreadyExists(err) {
				logWithClient.Error(err, "failed to create pod for client")
				test.Status.State = grpcv1.Errored
				test.Status.Reason = grpcv1.KubernetesError
				test.Status.Message = fmt.Sprintf("failed to create pod for client at index %d: %v", i, err)
				if updateErr := r.updateStatus(ctx, test); updateErr != nil {
					logWithClient.Error(updateErr, "failed to update status after failure to create pod for client")
				}
				return *result, err
			}
		}
		if missingPods.Driver != nil {
			logWithDriver := log.WithValues("driver", missingPods.Driver)

			pod, err := builder.PodForDriver(missingPods.Driver)
			if err != nil {
				logWithDriver.Error(err, "failed to construct a pod struct for supplied driver struct")
				test.Status.State = grpcv1.Errored
				test.Status.Reason = grpcv1.ConfigurationError
				test.Status.Message = fmt.Sprintf("failed to construct a pod for driver: %v", err)
				if updateErr := r.updateStatus(ctx, test); updateErr != nil {
					logWithDriver.Error(updateErr, "failed to update status after failure to construct a pod for driver")
				}
				return ctrl.Result{Requeue: false}, nil
			}

			if pool, ok := clusterInfo.DefaultPoolForRole(config.DriverRole); ok && missingPods.Driver.Pool == nil {
				pod.Labels[config.PoolLabel] = pool
			} else {
				pod.Labels[config.PoolLabel] = *missingPods.Driver.Pool
			}

			result, err := createPod(pod)
			if result != nil && !kerrors.IsAlreadyExists(err) {
				logWithDriver.Error(err, "failed to create pod for driver")
				test.Status.State = grpcv1.Errored
				test.Status.Reason = grpcv1.KubernetesError
				test.Status.Message = fmt.Sprintf("failed to create pod for driver: %v", err)
				if updateErr := r.updateStatus(ctx, test); updateErr != nil {
					logWithDriver.Error(updateErr, "failed to update status after failure to create pod for driver")
				}
				return *result, err
			}
		}
	}

	requeueTime := getRequeueTime(test, previousStatus, log)
	if requeueTime != 0 {
		return ctrl.Result{RequeueAfter: requeueTime}, nil
	}

	return ctrl.Result{Requeue: false}, nil
}

// CreateConfigMapIfMissing checks for the existence of a scenarios ConfigMap
// for the test. If one does not exist, it creates one with the same name and
// namespace as the test. The ConfigMap contains a single key "scenarios.json"
// with the contents of the ScenariosJSON field in the test spec.
//
// The ConfigMap will have the test as its owner's reference, meaning it will
// be garbage collected when the test is deleted.
//
// If the existence check, setting the owner's reference or the creation of the
// ConfigMap fail, an error is returned. Otherwise, the return value is nil.
func (r *LoadTestReconciler) CreateConfigMapIfMissing(ctx context.Context, test *grpcv1.LoadTest) error {
	nn := types.NamespacedName{Namespace: test.Namespace, Name: test.Name}
	log := r.Log.WithValues("loadtest", nn)
	cfgMap := new(corev1.ConfigMap)
	if err := r.get(ctx, nn, cfgMap); err != nil {
		log.Info("failed to find existing scenarios ConfigMap")

		if client.IgnoreNotFound(err) != nil {
			// The ConfigMap existence was not at issue, so this is likely an
			// issue with the Kubernetes API. So, we'll retry with exponential
			// backoff and allow the timeout to catch it.
			return &ControllerError{
				Message:      "failed to search for config map",
				WrappedError: err,
			}
		}

		cfgMap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      test.Name,
				Namespace: test.Namespace,
			},
			Data: map[string]string{
				"scenarios.json": test.Spec.ScenariosJSON,
			},

			// TODO: Enable ConfigMap immutability when it becomes available
			// Immutable: optional.BoolPtr(true),
		}

		if refError := setControllerReference(test, cfgMap, r.Scheme); refError != nil {
			// We should retry when we cannot set a controller reference on the
			// ConfigMap. This breaks garbage collection. If left to continue
			// for manual cleanup, it could create hidden errors when a load
			// test with the same name is created.
			return &ControllerError{
				Message:      "could not set owners reference on scenarios ConfigMap",
				WrappedError: refError,
			}
		}

		if createErr := r.create(ctx, cfgMap); createErr != nil {
			return &ControllerError{
				Message:      "failed to create scenarios ConfigMap",
				WrappedError: createErr,
			}
		}
	}

	return nil
}

// ClusterInfo provides information about the nodes in a Kubernetes cluster.
type ClusterInfo struct {
	// capacity is a map where the key is the name of the pool and the
	// value is the total number of nodes with a matching pool label.
	capacity map[string]int

	// availability is a map where the key is the name of the pool and
	// the value is the number of nodes with a matching pool label that
	// are not currently running pods for any LoadTest.
	availability map[string]int

	// defaultPools is a map where the key is the loadtest-role label
	// and the value is the name of the default pool for that role.
	defaultPools map[string]string
}

// CapacityForPool returns the total number of nodes with a matching
// pool label. In addition, it returns a boolean indicating whether the
// pool label has present on any node in the cluster.
func (ci *ClusterInfo) CapacityForPool(pool string) (cap int, ok bool) {
	cap, ok = ci.capacity[pool]
	return
}

// AvailabilityForPool returns the number of nodes with a matching pool
// label that are not currently running pods for any LoadTest. In
// addition, it returns a boolean indicating whether the pool label has
// been present on any node in the cluster.
func (ci *ClusterInfo) AvailabilityForPool(pool string) (availability int, ok bool) {
	availability, ok = ci.availability[pool]
	return
}

// DefaultPoolForRole returns the default pool for a given role. In additition,
// it returns a boolean indicating whether this role has been seen on any node
// in the cluster.
func (ci *ClusterInfo) DefaultPoolForRole(role string) (pool string, ok bool) {
	pool, ok = ci.defaultPools[role]
	return
}

// CurrentClusterInfo accepts the list of all nodes in the cluster; the list of
// all running, pending, errored, and completed pods; and the default pool
// labels (if applicable). It processes this data to create a ClusterInfo
// instance. This instance provides information like the current availability
// and pools in the cluster.
func CurrentClusterInfo(nodes *corev1.NodeList, pods *corev1.PodList, defaultPoolLabels *config.PoolLabelMap, log logr.Logger) *ClusterInfo {
	info := &ClusterInfo{
		capacity:     make(map[string]int),
		availability: make(map[string]int),
		defaultPools: make(map[string]string),
	}

	for _, node := range nodes.Items {
		pool, ok := node.Labels[config.PoolLabel]
		if !ok && log != nil {
			log.Info("encountered a node without a pool label", "nodeName", node.Name)
			continue
		}

		if defaultPoolLabels != nil {
			if _, ok := info.defaultPools[config.ClientRole]; !ok {
				if _, ok = node.Labels[defaultPoolLabels.Client]; ok {
					info.defaultPools[config.ClientRole] = pool
				}
			}
			if _, ok := info.defaultPools[config.DriverRole]; !ok {
				if _, ok = node.Labels[defaultPoolLabels.Driver]; ok {
					info.defaultPools[config.DriverRole] = pool
				}
			}
			if _, ok := info.defaultPools[config.ServerRole]; !ok {
				if _, ok = node.Labels[defaultPoolLabels.Server]; ok {
					info.defaultPools[config.ServerRole] = pool
				}
			}

			if _, ok = info.capacity[pool]; !ok {
				info.capacity[pool] = 0
			}
		}

		info.capacity[pool]++
	}

	poolAvailabilities := make(map[string]int)
	for pool, capacity := range info.capacity {
		poolAvailabilities[pool] = capacity
	}
	for _, pod := range pods.Items {
		pool, ok := pod.Labels[config.PoolLabel]
		if !ok && log != nil {
			log.Info("encountered a pod without a pool label", "pod", pod)
			continue
		}
		if pod.Status.Phase != corev1.PodSucceeded && pod.Status.Phase != corev1.PodFailed {
			poolAvailabilities[pool]--
		}
	}

	return info
}

// defaultAdjustment contains the information of which default pool placeholders
// should be removed. See the adjustAvailabilityForDefaults function for more
// details.
type defaultAdjustment struct {
	// DefaultPoolKey is the key that is used as a placeholder for a default
	// pool when calculating the number of nodes it requires.
	DefaultPoolKey string

	// Role is the function of these pods in the test. For example, clients will
	// be config.ClientRole; servers will be config.ServerRole; and drivers will
	// be config.DriverRole.
	Role string
}

// adjustAvailabilityForDefaults uses the current information known about a
// cluster to apply a set of adjustments to the LoadTestMissing struct. These
// adjustments add the total number of nodes needed from unspecified pools,
// "default pools", to specific pools. This way the system can adequately
// determine if enough nodes are available to schedule the test.
func adjustAvailabilityForDefaults(clusterInfo *ClusterInfo, missingPods *status.LoadTestMissing, adjustments ...defaultAdjustment) {
	for _, adjustment := range adjustments {
		defaultPoolForRole, ok := clusterInfo.DefaultPoolForRole(adjustment.Role)
		if !ok {
			return
		}

		defaultPoolNodeCount, ok := missingPods.NodeCountByPool[adjustment.DefaultPoolKey]
		if !ok {
			return
		}

		missingPods.NodeCountByPool[defaultPoolForRole] += defaultPoolNodeCount
		delete(missingPods.NodeCountByPool, adjustment.DefaultPoolKey)
	}
}

// getRequeueTime takes a LoadTest and its previous status, compares the
// previous status of the load test with its updated status, and returns a
// calculated requeue time. If the test has just been assigned a start time
// (i.e., it has just started), the requeue time is set to the timeout value
// specified in the LoadTest. If the test has just been assigned a stop time
// (i.e., it has just terminated), the requeue time is set to the time-to-live
// specified in the LoadTest, minus its actual running time. In other cases,
// the requeue time is set to zero.
func getRequeueTime(updatedLoadTest *grpcv1.LoadTest, previousStatus grpcv1.LoadTestStatus, log logr.Logger) time.Duration {
	requeueTime := time.Duration(0)

	if previousStatus.StartTime == nil && updatedLoadTest.Status.StartTime != nil {
		requeueTime = time.Duration(updatedLoadTest.Spec.TimeoutSeconds) * time.Second
		log.Info("just started, should be marked as error if still running at :" + time.Now().Add(requeueTime).String())
		return requeueTime
	}

	if previousStatus.StopTime == nil && updatedLoadTest.Status.StopTime != nil {
		requeueTime = time.Duration(updatedLoadTest.Spec.TTLSeconds)*time.Second - updatedLoadTest.Status.StopTime.Sub(updatedLoadTest.Status.StartTime.Time)
		log.Info("just end, should be deleted at :" + time.Now().Add(requeueTime).String())
		return requeueTime
	}

	return requeueTime
}

// SetupWithManager configures a controller-runtime manager.
func (r *LoadTestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.mgr = mgr
	r.create = r.Create
	r.get = r.Get
	r.list = r.List
	r.update = r.Update
	r.updateStatus = r.Status().Update
	r.delete = r.Delete

	return ctrl.NewControllerManagedBy(mgr).
		For(&grpcv1.LoadTest{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
