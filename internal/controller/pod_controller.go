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
	"slices"
	"sync"
	"time"

	"github.com/cybozu-go/cat-gate/internal/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// scaleRate is the rate at which scheduling gates are opened per node with image.
const scaleRate = 2

// minimumCapacity the number of scheduling gates to remove when no node have the image.
const minimumCapacity = 1

var requeueSeconds = 10
var gateRemovalDelayMilliSecond = 10
var GateRemovalHistories = sync.Map{}

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=pods/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	reqPod := &corev1.Pod{}
	err := r.Get(ctx, req.NamespacedName, reqPod)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !existsSchedulingGate(reqPod) {
		return ctrl.Result{}, nil
	}

	if reqPod.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	annotations := reqPod.Annotations
	if _, ok := annotations[constants.CatGateImagesHashAnnotation]; !ok {
		logger.V(constants.LevelWarning).Info("pod annotation not found")
		err := r.removeSchedulingGate(ctx, reqPod)
		if err != nil {
			logger.Error(err, "failed to remove scheduling gate")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	reqImagesHash := annotations[constants.CatGateImagesHashAnnotation]

	// prevents removing the scheduling gate based on information before the cache is updated.
	if value, ok := GateRemovalHistories.Load(reqImagesHash); ok {
		lastGateRemovalTime := value.(time.Time)
		if time.Since(lastGateRemovalTime) < time.Duration(gateRemovalDelayMilliSecond)*time.Millisecond {
			logger.V(constants.LevelDebug).Info("perform retry processing to avoid race conditions", "lastGateRemovalTime", lastGateRemovalTime)
			return ctrl.Result{RequeueAfter: time.Duration(gateRemovalDelayMilliSecond) * time.Millisecond}, nil
		}
	}

	var reqImageList []string
	for _, initContainer := range reqPod.Spec.InitContainers {
		if initContainer.Image == "" {
			continue
		}
		reqImageList = append(reqImageList, initContainer.Image)
	}
	for _, container := range reqPod.Spec.Containers {
		if container.Image == "" {
			continue
		}
		reqImageList = append(reqImageList, container.Image)
	}

	nodes := &corev1.NodeList{}
	err = r.List(ctx, nodes)
	if err != nil {
		logger.Error(err, "failed to list nodes")
		return ctrl.Result{}, err
	}

	nodeImageSet := make(map[string][]string)
	for _, node := range nodes.Items {
		for _, image := range node.Status.Images {
			nodeImageSet[node.Name] = append(nodeImageSet[node.Name], image.Names...)
		}
	}

	nodeSet := make(map[string]struct{})
	for nodeName, images := range nodeImageSet {
		allImageExists := true
		for _, reqImage := range reqImageList {
			if !slices.Contains(images, reqImage) {
				allImageExists = false
				break
			}
		}
		if allImageExists {
			nodeSet[nodeName] = struct{}{}
		}
	}

	pods := &corev1.PodList{}
	err = r.List(ctx, pods, client.MatchingFields{constants.ImageHashAnnotationField: reqImagesHash})
	if err != nil {
		logger.Error(err, "failed to list pods")
		return ctrl.Result{}, err
	}

	numSchedulablePods := 0
	numImagePulledPods := 0

	for _, pod := range pods.Items {
		if existsSchedulingGate(&pod) {
			continue
		}
		numSchedulablePods += 1

		if pod.Status.Phase != corev1.PodPending {
			numImagePulledPods += 1
		}
	}

	capacity := len(nodeSet) * scaleRate

	if capacity < minimumCapacity {
		capacity = minimumCapacity
	}
	logger.V(constants.LevelDebug).Info("schedule capacity", "capacity", capacity, "len(nodeSet)", len(nodeSet))

	numImagePullingPods := numSchedulablePods - numImagePulledPods
	logger.V(constants.LevelDebug).Info("scheduling progress", "numSchedulablePods", numSchedulablePods, "numImagePulledPods", numImagePulledPods, "numImagePullingPods", numImagePullingPods)

	if capacity > numImagePullingPods {
		err := r.removeSchedulingGate(ctx, reqPod)
		if err != nil {
			logger.Error(err, "failed to remove scheduling gate")
			return ctrl.Result{}, err
		}
		now := time.Now()
		GateRemovalHistories.Store(reqImagesHash, now)
		return ctrl.Result{}, nil
	}

	return ctrl.Result{
		RequeueAfter: time.Duration(requeueSeconds) * time.Second,
	}, nil
}

func (r *PodReconciler) removeSchedulingGate(ctx context.Context, pod *corev1.Pod) error {
	var filteredGates []corev1.PodSchedulingGate
	existsGate := false
	for _, gate := range pod.Spec.SchedulingGates {
		if gate.Name == constants.PodSchedulingGateName {
			existsGate = true
			continue
		}
		filteredGates = append(filteredGates, gate)
	}
	pod.Spec.SchedulingGates = filteredGates
	if existsGate {
		logger := log.FromContext(ctx)
		err := r.Update(ctx, pod)
		if err != nil {
			return err
		}
		logger.Info("scheduling gate deleted")
	}
	return nil
}

func existsSchedulingGate(pod *corev1.Pod) bool {
	for _, gate := range pod.Spec.SchedulingGates {
		if gate.Name == constants.PodSchedulingGateName {
			return true
		}
	}
	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pred := func(obj client.Object) bool {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return false
		}
		return existsSchedulingGate(pod)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool { return pred(e.Object) },
			UpdateFunc: func(e event.UpdateEvent) bool { return pred(e.ObjectNew) },
			DeleteFunc: func(e event.DeleteEvent) bool { return pred(e.Object) },
		}).
		Complete(r)
}
