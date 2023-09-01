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

	It("should schedule 1 pod when single pod is created", func() {
		testName := "single-pod"
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testName,
			},
		}
		err := k8sClient.Create(ctx, namespace)
		Expect(err).NotTo(HaveOccurred())

		createNewPod(0, testName)

		pod := &corev1.Pod{}
		Eventually(func(g Gomega) {
			err = k8sClient.Get(ctx, client.ObjectKey{Name: fmt.Sprintf("%s-pod-%d", testName, 0), Namespace: testName}, pod)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(pod.Spec.SchedulingGates).NotTo(ConsistOf(corev1.PodSchedulingGate{Name: constants.PodSchedulingGateName}))
		}).Should(Succeed())
	})

	It("should schedule 1 pod when multiple pods are created", func() {
		testName := "multiple-pods"
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
			err := k8sClient.List(ctx, pods, &client.ListOptions{Namespace: testName})
			g.Expect(err).NotTo(HaveOccurred())
			numSchedulable := 0
			for _, pod := range pods.Items {
				if !existsSchedulingGate(pod) {
					numSchedulable += 1
				}
			}
			g.Expect(numSchedulable).To(Equal(1))
		}).Should(Succeed())
	})

	It("should schedule x pods when 2 pods are already scheduled", func() {
		testName := "multiple-pods"
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
			err := k8sClient.List(ctx, pods, &client.ListOptions{Namespace: testName})
			g.Expect(err).NotTo(HaveOccurred())
			numSchedulable := 0
			for _, pod := range pods.Items {
				if !existsSchedulingGate(pod) {
					numSchedulable += 1
				}
			}
			g.Expect(numSchedulable).To(Equal(1))
		}).Should(Succeed())
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
