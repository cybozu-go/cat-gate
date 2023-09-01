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

	It("should add scheduling gate and annotation to pod", func() {
		sample := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "sample",
			},
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{
					{
						Name:  "sample1",
						Image: "example.com/sample1-image:1.0.0",
					},
				},
				Containers: []corev1.Container{
					{
						Name:  "sample2",
						Image: "example.com/sample2-image:1.0.0",
					},
				},
			},
		}
		err := k8sClient.Create(ctx, sample)

		Expect(err).NotTo(HaveOccurred())

		pod := &corev1.Pod{}
		err = k8sClient.Get(ctx, client.ObjectKey{Name: "sample", Namespace: "default"}, pod)

		Expect(err).NotTo(HaveOccurred())
		Expect(pod.Spec.SchedulingGates).To(ConsistOf(corev1.PodSchedulingGate{Name: podSchedulingGateName}))
		Expect(pod.Annotations).To(HaveKeyWithValue(catGateImagesHashAnnotation, "060e64ec0b5abc015254466dc4d0ec89bc4e996121ff5b0f7fc120df3f15954e"))
	})
})
