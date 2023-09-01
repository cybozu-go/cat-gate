package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("CatGate controller", func() {

	ctx := context.Background()

	It("should success", func() {
		newPod := &corev1.Pod{
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
		err := k8sClient.Create(ctx, newPod)
		Expect(err).NotTo(HaveOccurred())
	})

})
