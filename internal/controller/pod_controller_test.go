package controller

import (
	"context"
	"fmt"

	"github.com/cybozu-go/cat-gate/internal/constants"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("CatGate controller", func() {

	ctx := context.Background()

	It("should remove scheduling gate when the number of pods is 1", func() {
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

		pod := &corev1.Pod{}
		Eventually(func(g Gomega) {
			err = k8sClient.Get(ctx, client.ObjectKey{Name: "sample", Namespace: "default"}, pod)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(pod.Spec.SchedulingGates).NotTo(ConsistOf(corev1.PodSchedulingGate{Name: constants.PodSchedulingGateName}))
		}).Should(Succeed())
	})

	It("should remove 1 scheduling gate when the number of pods is 8", func() {
		testName := "multiplepods"
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testName,
			},
		}
		err := k8sClient.Create(ctx, namespace)
		Expect(err).NotTo(HaveOccurred())

		for i := 0; i < 8; i++ {
			createNewPod(i, testName)
		}
		pods := &corev1.PodList{}
		Eventually(func(g Gomega) {
			err := k8sClient.List(ctx, pods)
			g.Expect(err).NotTo(HaveOccurred())
			// TODO: 全部待つ
		})

	})
})

func createNewPod(index int, testName string) {
	newPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testName,
			Name:      fmt.Sprintf("%s-pod-%d", testName, index),
		},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{
				{
					Name:  "sample1",
					Image: testName + ".example.com/sample1-image:1.0.0",
				},
			},
			Containers: []corev1.Container{
				{
					Name:  "sample2",
					Image: testName + ".example.com/sample2-image:1.0.0",
				},
			},
			Hostname: fmt.Sprintf("%s-hostname-%d", testName, index),
		},
	}
	err := k8sClient.Create(ctx, newPod)
	Expect(err).NotTo(HaveOccurred())
}
