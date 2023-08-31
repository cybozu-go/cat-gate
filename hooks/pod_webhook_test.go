package hooks

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Webhook Test", func() {
	ctx := context.Background()

	It("should add scheduling gate to pod", func() {
		sample := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "invalid-sample",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "sample",
						Image: "example.com/sample-image:1.0.0",
					},
				},
			},
		}
		err := k8sClient.Create(ctx, sample)

		Expect(err).NotTo(HaveOccurred())

		pod := &corev1.Pod{}
		err = k8sClient.Get(ctx, client.ObjectKey{Name: "invalid-sample", Namespace: "default"}, pod)

		Expect(err).NotTo(HaveOccurred())
		Expect(pod.Spec.SchedulingGates).To(ConsistOf(corev1.PodSchedulingGate{Name: podSchedulingGateName}))
	})
})
