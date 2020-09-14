package helpers

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PodsForLoadTest", func() {
	It("returns nil when no load test is supplied", func() {
		pods := PodsForLoadTest(nil, []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ignored-pod",
				},
			},
		})

		Expect(pods).To(BeNil())
	})

	It("returns empty when no pods are supplied", func() {
		loadtest := new(grpcv1.LoadTest)
		loadtest.Name = "empty-pods-loadtest"

		pods := PodsForLoadTest(loadtest, nil)
		Expect(pods).To(BeEmpty())

		pods = PodsForLoadTest(loadtest, []corev1.Pod{})
		Expect(pods).To(BeEmpty())
	})

	It("includes only pods with matching labels", func() {
		loadtest := new(grpcv1.LoadTest)
		loadtest.Name = "pods-matching-labels-loadtest"

		allPods := []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "good-pod-1",
					Labels: map[string]string{
						config.LoadTestLabel: loadtest.Name,
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "bad-pod-1",
					Labels: map[string]string{
						config.LoadTestLabel: "other-load-test",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "good-pod-2",
					Labels: map[string]string{
						config.LoadTestLabel: loadtest.Name,
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "bad-pod-2",
					Labels: nil,
				},
			},
		}

		pods := PodsForLoadTest(loadtest, allPods)
		Expect(pods).To(ConsistOf(&allPods[0], &allPods[2]))
	})
})
