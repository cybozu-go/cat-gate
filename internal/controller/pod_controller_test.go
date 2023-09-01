package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("CatGate controller", func() {

	ctx := context.Background()
	var stopFunc func()

	BeforeEach(func() {
		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme: scheme,
		})
		Expect(err).ToNot(HaveOccurred())

		reconciler := PodReconciler{
			Client: k8sClient,
			Scheme: scheme,
		}
		err = reconciler.SetupWithManager(mgr)
		Expect(err).NotTo(HaveOccurred())

		ctx, cancel := context.WithCancel(ctx)
		stopFunc = cancel
		go func() {
			err := mgr.Start(ctx)
			if err != nil {
				panic(err)
			}
		}()
		time.Sleep(100 * time.Millisecond)
	})

	AfterEach(func() {
		stopFunc()
		time.Sleep(100 * time.Millisecond)
	})

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
