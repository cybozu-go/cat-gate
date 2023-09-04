/*
Copyright 2023.

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

package controller

import (
	"context"
	"errors"
	"time"

	"github.com/cybozu-go/cat-gate/internal/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=pods/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Pod object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	reqPod := &corev1.Pod{}
	err := r.Get(ctx, req.NamespacedName, reqPod)
	if err != nil {
		return ctrl.Result{}, err
	}

	annotations := reqPod.Annotations
	if _, ok := annotations[constants.CatGateImagesHashAnnotation]; !ok {
		logger.Error(errors.New("not found pod annotation"), "not found pod annotation")
		err := r.removeSchedulingGate(ctx, reqPod)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	reqImagesHash := annotations[constants.CatGateImagesHashAnnotation]

	pods := &corev1.PodList{}
	err = r.List(ctx, pods)
	if err != nil {
		return ctrl.Result{}, err
	}

	nodeSet := make(map[string]struct{})

	numSchedulablePods := 0
	numImagePulledPods := 0

	for _, pod := range pods.Items {
		if _, ok := pod.Annotations[constants.CatGateImagesHashAnnotation]; !ok {
			continue
		}
		if pod.Annotations[constants.CatGateImagesHashAnnotation] != reqImagesHash {
			continue
		}

		if existsSchedulingGate(pod) {
			continue
		}
		numSchedulablePods += 1

		allStarted := true
		statuses := pod.Status.ContainerStatuses
		for _, status := range statuses {
			if status.State.Running == nil && status.State.Terminated == nil {
				allStarted = false
				break
			}
		}

		if allStarted && len(pod.Spec.Containers) == len(statuses) {
			nodeSet[pod.Spec.Hostname] = struct{}{}
			numImagePulledPods += 1
		}
	}

	const scaleRate = 2
	const minimumCapacity = 1

	capacity := len(nodeSet) * scaleRate

	if capacity < minimumCapacity {
		capacity = minimumCapacity
	}

	numImagePullingPods := numSchedulablePods - numImagePulledPods

	if capacity > numImagePullingPods {
		err := r.removeSchedulingGate(ctx, reqPod)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: 3 * time.Second,
	}, nil
}

func (r *PodReconciler) removeSchedulingGate(ctx context.Context, pod *corev1.Pod) error {
	var filterdGates []corev1.PodSchedulingGate
	existsGate := false
	for _, gate := range pod.Spec.SchedulingGates {
		if gate.Name == constants.PodSchedulingGateName {
			existsGate = true
			continue
		}
		filterdGates = append(filterdGates, gate)
	}
	pod.Spec.SchedulingGates = filterdGates
	if existsGate {
		logger := log.FromContext(ctx)
		err := r.Update(ctx, pod)
		if err != nil {
			return err
		}
		logger.Info("Scheduling gate deleted")
	}
	return nil
}

func existsSchedulingGate(pod corev1.Pod) bool {
	for _, gate := range pod.Spec.SchedulingGates {
		if gate.Name == constants.PodSchedulingGateName {
			return true
		}
	}
	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// schedulingGateがついているかでfiltering
	// reconcile頻度を減らす

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(r)
}
