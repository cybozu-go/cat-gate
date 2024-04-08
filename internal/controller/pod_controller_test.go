package controller

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/cybozu-go/cat-gate/internal/constants"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("CatGate controller", func() {

	ctx := context.Background()
	requeueSeconds = 1

	It("should schedule a pod if it is created solely", func() {
		testName := "single-pod"
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testName,
			},
		}
		err := k8sClient.Create(ctx, namespace)
		Expect(err).NotTo(HaveOccurred())

		createNewPod(testName, 0)

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
			createNewPod(testName, i)
		}
		pods := &corev1.PodList{}
		Eventually(func(g Gomega) {
			err := k8sClient.List(ctx, pods, &client.ListOptions{Namespace: testName})
			g.Expect(err).NotTo(HaveOccurred())
			numSchedulable := 0
			for _, pod := range pods.Items {
				if !existsSchedulingGate(&pod) {
					numSchedulable += 1
				}
			}
			g.Expect(numSchedulable).To(Equal(1))
		}).Should(Succeed())
	})

	It("should schedule pods exponentially", func() {
		testName := "multiple-pods-with-running-pods"
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testName,
			},
		}
		err := k8sClient.Create(ctx, namespace)
		Expect(err).NotTo(HaveOccurred())

		for i := 0; i < 10; i++ {
			createNewPod(testName, i)
			createNewNode(testName, i)
		}

		pods := &corev1.PodList{}

		Eventually(func(g Gomega) {
			err := k8sClient.List(ctx, pods, &client.ListOptions{Namespace: testName})
			g.Expect(err).NotTo(HaveOccurred())
			numSchedulable := 0
			for _, pod := range pods.Items {
				if !existsSchedulingGate(&pod) {
					numSchedulable += 1
				}
			}
			// no pod is already running, so 1 pods should be scheduled
			g.Expect(numSchedulable).To(Equal(1))
		}).Should(Succeed())
		scheduleAndStartPods(testName)

		Eventually(func(g Gomega) {
			err := k8sClient.List(ctx, pods, &client.ListOptions{Namespace: testName})
			g.Expect(err).NotTo(HaveOccurred())
			numSchedulable := 0
			for _, pod := range pods.Items {
				if !existsSchedulingGate(&pod) {
					numSchedulable += 1
				}
			}
			// 1 pod is already running, so 3(1 + 1*2) pods should be scheduled
			g.Expect(numSchedulable).To(Equal(3))
		}).Should(Succeed())
		scheduleAndStartPods(testName)

		Eventually(func(g Gomega) {
			err := k8sClient.List(ctx, pods, &client.ListOptions{Namespace: testName})
			g.Expect(err).NotTo(HaveOccurred())
			numSchedulable := 0
			for _, pod := range pods.Items {
				if !existsSchedulingGate(&pod) {
					numSchedulable += 1
				}
			}
			// 3 pods are already running, so 9 (3 + 3*2) pods should be scheduled
			g.Expect(numSchedulable).To(Equal(9))
		}).Should(Succeed())
	})

	It("should remove scheduling gate when annotation is removed intentionally", func() {
		testName := "remove-scheduling-gate-fail-safe"
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testName,
			},
		}
		err := k8sClient.Create(ctx, namespace)
		Expect(err).NotTo(HaveOccurred())

		createNewPod(testName, 0)

		pod := &corev1.Pod{}
		Eventually(func(g Gomega) {
			err = k8sClient.Get(ctx, client.ObjectKey{Name: fmt.Sprintf("%s-pod-%d", testName, 0), Namespace: testName}, pod)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(pod.Spec.SchedulingGates).NotTo(ConsistOf(corev1.PodSchedulingGate{Name: constants.PodSchedulingGateName}))
		}).Should(Succeed())

		pod = createNewPod(testName, 1)
		delete(pod.Annotations, constants.CatGateImagesHashAnnotation)
		err = k8sClient.Update(ctx, pod)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func(g Gomega) {
			err = k8sClient.Get(ctx, client.ObjectKey{Name: fmt.Sprintf("%s-pod-%d", testName, 1), Namespace: testName}, pod)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(pod.Spec.SchedulingGates).NotTo(ConsistOf(corev1.PodSchedulingGate{Name: constants.PodSchedulingGateName}))
		}).Should(Succeed())
	})

	It("should limit the number of schedulable pods based on status", func() {
		testName := "multiple-pods-with-some-running-pods"
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testName,
			},
		}
		err := k8sClient.Create(ctx, namespace)
		Expect(err).NotTo(HaveOccurred())

		for i := 0; i < 10; i++ {
			createNewPod(testName, i)
			createNewNode(testName, i)
		}
		pods := &corev1.PodList{}

		Eventually(func(g Gomega) {
			err := k8sClient.List(ctx, pods, &client.ListOptions{Namespace: testName})
			g.Expect(err).NotTo(HaveOccurred())
			numSchedulable := 0
			for _, pod := range pods.Items {
				if !existsSchedulingGate(&pod) {
					numSchedulable += 1
				}
			}
			// no pod is already running, so 1 pods should be scheduled
			g.Expect(numSchedulable).To(Equal(1))
		}).Should(Succeed())
		scheduleAndStartPods(testName)

		Eventually(func(g Gomega) {
			err := k8sClient.List(ctx, pods, &client.ListOptions{Namespace: testName})
			g.Expect(err).NotTo(HaveOccurred())
			numSchedulable := 0
			for _, pod := range pods.Items {
				if !existsSchedulingGate(&pod) {
					numSchedulable += 1
				}
			}
			// 1 pod is already running, so 3(1 + 1*2) pods should be scheduled
			g.Expect(numSchedulable).To(Equal(3))
		}).Should(Succeed())
		scheduleAndStartOnePod(testName)

		Eventually(func(g Gomega) {
			err := k8sClient.List(ctx, pods, &client.ListOptions{Namespace: testName})
			g.Expect(err).NotTo(HaveOccurred())
			numSchedulable := 0
			for _, pod := range pods.Items {
				if !existsSchedulingGate(&pod) {
					numSchedulable += 1
				}
			}
			// 2 pods are already running, so 6 (2 + 2*2) pods should be scheduled
			g.Expect(numSchedulable).To(Equal(6))
		}).Should(Succeed())
	})

	It("should allow scheduling of additional pods when multiple pods are running on a single node", func() {
		testName := "multiple-pods-running-on-single-node"
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testName,
			},
		}
		err := k8sClient.Create(ctx, namespace)
		Expect(err).NotTo(HaveOccurred())

		for i := 0; i < 3; i++ {
			createNewPod(testName, i)
		}
		for i := 0; i < 1; i++ {
			createNewNode(testName, i)
		}

		nodes := &corev1.NodeList{}
		err = k8sClient.List(ctx, nodes, &client.ListOptions{Namespace: testName})
		Expect(err).NotTo(HaveOccurred())

		pods := &corev1.PodList{}
		Eventually(func(g Gomega) {
			err := k8sClient.List(ctx, pods, &client.ListOptions{Namespace: testName})
			g.Expect(err).NotTo(HaveOccurred())
			numSchedulable := 0
			for _, pod := range pods.Items {
				if !existsSchedulingGate(&pod) {
					numSchedulable += 1
				}
			}
			// no pod is already running, so 1 pods should be scheduled
			g.Expect(numSchedulable).To(Equal(1))
		}).Should(Succeed())
		scheduleSpecificNodeAndStartOnePod(testName, nodes.Items[0].Name)

		Eventually(func(g Gomega) {
			err := k8sClient.List(ctx, pods, &client.ListOptions{Namespace: testName})
			g.Expect(err).NotTo(HaveOccurred())
			numSchedulable := 0
			for _, pod := range pods.Items {
				if !existsSchedulingGate(&pod) {
					numSchedulable += 1
				}
			}
			// 1 pod is already running, so 3(1 + 1*2) pods should be scheduled
			g.Expect(numSchedulable).To(Equal(3))
		}).Should(Succeed())
		scheduleSpecificNodeAndStartOnePod(testName, nodes.Items[0].Name)

		Eventually(func(g Gomega) {
			err := k8sClient.List(ctx, pods, &client.ListOptions{Namespace: testName})
			g.Expect(err).NotTo(HaveOccurred())
			numSchedulable := 0
			for _, pod := range pods.Items {
				if !existsSchedulingGate(&pod) {
					numSchedulable += 1
				}
			}
			// 2 pods are already running on a same node, so 3 pods should be scheduled
			g.Expect(numSchedulable).To(Equal(3))
		}).Should(Succeed())

	})

	It("Should the schedule not increase if the pod is not Running", func() {
		testName := "crash-pod"
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testName,
			},
		}
		err := k8sClient.Create(ctx, namespace)
		Expect(err).NotTo(HaveOccurred())

		for i := 0; i < 3; i++ {
			createNewPod(testName, i)
		}
		for i := 0; i < 3; i++ {
			createNewNode(testName, i)
		}

		pods := &corev1.PodList{}
		Eventually(func(g Gomega) {
			err := k8sClient.List(ctx, pods, &client.ListOptions{Namespace: testName})
			g.Expect(err).NotTo(HaveOccurred())
			numSchedulable := 0
			for _, pod := range pods.Items {
				if !existsSchedulingGate(&pod) {
					numSchedulable += 1
				}
			}
			// no pod is already running, so 1 pods should be scheduled
			g.Expect(numSchedulable).To(Equal(1))
		}).Should(Succeed())
		scheduleAndStartOneUnhealthyPod(testName)

		Eventually(func(g Gomega) {
			err := k8sClient.List(ctx, pods, &client.ListOptions{Namespace: testName})
			g.Expect(err).NotTo(HaveOccurred())
			numSchedulable := 0
			for _, pod := range pods.Items {
				if !existsSchedulingGate(&pod) {
					numSchedulable += 1
				}
			}
			// 1 pod is already scheduled, but it is not running, so 1 pod should be scheduled
			g.Expect(numSchedulable).To(Equal(1))
		}).Should(Succeed())
	})
})

func createNewPod(testName string, index int) *corev1.Pod {
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
		},
	}
	err := k8sClient.Create(ctx, newPod)
	Expect(err).NotTo(HaveOccurred())
	return newPod
}

func createNewNode(testName string, index int) {
	newNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-node-%d", testName, index),
		},
		Status: corev1.NodeStatus{},
	}
	err := k8sClient.Create(ctx, newNode)
	Expect(err).NotTo(HaveOccurred())
}

func scheduleAndStartPods(namespace string) {
	pods := &corev1.PodList{}
	err := k8sClient.List(ctx, pods, client.InNamespace(namespace))
	Expect(err).NotTo(HaveOccurred())

	for _, pod := range pods.Items {
		if !existsSchedulingGate(&pod) {
			updatePodStatus(&pod, corev1.ContainerState{Running: &corev1.ContainerStateRunning{}})

			node := &corev1.Node{}
			err = k8sClient.Get(ctx, client.ObjectKey{Name: pod.Status.HostIP}, node)
			Expect(err).NotTo(HaveOccurred())
			updateNodeImageStatus(node, pod.Spec.Containers)
		}
	}
}

func scheduleAndStartOnePod(namespace string) {
	pods := &corev1.PodList{}
	err := k8sClient.List(ctx, pods, client.InNamespace(namespace))
	Expect(err).NotTo(HaveOccurred())

	for _, pod := range pods.Items {
		if !existsSchedulingGate(&pod) && len(pod.Status.ContainerStatuses) == 0 {
			updatePodStatus(&pod, corev1.ContainerState{Running: &corev1.ContainerStateRunning{}})

			node := &corev1.Node{}
			err = k8sClient.Get(ctx, client.ObjectKey{Name: pod.Status.HostIP}, node)
			Expect(err).NotTo(HaveOccurred())
			updateNodeImageStatus(node, pod.Spec.Containers)
			break
		}
	}
}

func scheduleSpecificNodeAndStartOnePod(namespace, nodeName string) {
	pods := &corev1.PodList{}
	err := k8sClient.List(ctx, pods, client.InNamespace(namespace))
	Expect(err).NotTo(HaveOccurred())

	for _, pod := range pods.Items {
		if !existsSchedulingGate(&pod) && len(pod.Status.ContainerStatuses) == 0 {
			updatePodStatusWithHostIP(&pod, corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}, nodeName)
			node := &corev1.Node{}
			err = k8sClient.Get(ctx, client.ObjectKey{Name: pod.Status.HostIP}, node)
			Expect(err).NotTo(HaveOccurred())
			updateNodeImageStatus(node, pod.Spec.Containers)
			break
		}
	}
}

func scheduleAndStartOneUnhealthyPod(namespace string) {
	pods := &corev1.PodList{}
	err := k8sClient.List(ctx, pods, client.InNamespace(namespace))
	Expect(err).NotTo(HaveOccurred())

	for _, pod := range pods.Items {
		if !existsSchedulingGate(&pod) && len(pod.Status.ContainerStatuses) == 0 {
			updatePodStatus(&pod, corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "RunContainerError"}})

			node := &corev1.Node{}
			err = k8sClient.Get(ctx, client.ObjectKey{Name: pod.Status.HostIP}, node)
			Expect(err).NotTo(HaveOccurred())
			updateNodeImageStatus(node, pod.Spec.Containers)
			break
		}
	}
}

func updatePodStatus(pod *corev1.Pod, state corev1.ContainerState) {
	pod.Status.ContainerStatuses = []corev1.ContainerStatus{
		{
			State: state,
		},
	}
	podName := pod.GetObjectMeta().GetName()
	regex, err := regexp.Compile(`pod-\d+$`)
	Expect(err).NotTo(HaveOccurred())
	idx := regex.FindStringIndex(podName)
	nodeName := podName[:idx[0]] + strings.Replace(podName[idx[0]:], "pod", "node", 1)
	pod.Status.HostIP = nodeName
	err = k8sClient.Status().Update(ctx, pod)
	Expect(err).NotTo(HaveOccurred())
}

func updatePodStatusWithHostIP(pod *corev1.Pod, state corev1.ContainerState, nodeName string) {
	pod.Status.ContainerStatuses = []corev1.ContainerStatus{
		{
			State: state,
		},
	}
	pod.Status.HostIP = nodeName
	err := k8sClient.Status().Update(ctx, pod)
	Expect(err).NotTo(HaveOccurred())
}

func updateNodeImageStatus(node *corev1.Node, containers []corev1.Container) {
	for _, container := range containers {
		node.Status.Images = append(node.Status.Images, corev1.ContainerImage{
			Names: []string{container.Image},
		})
	}
	err := k8sClient.Status().Update(ctx, node)
	Expect(err).NotTo(HaveOccurred())
}
