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
	if _, ok := annotations[constants.CatGateImagesHashAnnotation] ; !ok {
		logger.Error(errors.New("not found pod annotation"), "not found pod annotation")
		removeSchedulingGate(reqPod)
		return ctrl.Result{}, nil
	}
	reqImagesHash := annotations[constants.CatGateImagesHashAnnotation]

	pods := &corev1.PodList{}
	err = r.List(ctx, pods)
	if err != nil {
		return ctrl.Result{}, err
	}

	// まず，イメージがありそうなノードの一覧を取得して，[]nodenameに入れる
	for _, pod := range pods.Items {
		// TODO: reqのannotationで絞る
		if _, ok := pod.Annotations[constants.CatGateImagesHashAnnotation] ; !ok {
			continue
		}
		if pod.Annotations[constants.CatGateImagesHashAnnotation] != reqImagesHash {
			continue
		}

		// capacity := len(xxxx) * settings.ScaleTimes(2とか)

		// schedulingGateが外れている（=イメージ取得を開始した(A)，あるいは完了した(B)）Podの数を取得
		// numSchedulable :=
		// 起動済み（=イメージ取得を完了した(B)）Podの数を取得
		// numPulledImage :=
		// イメージ取得中のPodの数を計算(=イメージ取得を開始した(A)もののみが得られる)
		// numPullingImage := numSchedulable - numPulledImage
	}
	// schedulingGateを外す処理
	// if capacity > numPullingImage { //余裕があれば
	// 外す
	// }

	return ctrl.Result{}, nil
}

func removeSchedulingGate(pod *corev1.Pod) {
	var filterdGates []corev1.PodSchedulingGate
	for _, gate := range pod.Spec.SchedulingGates {
		if gate.Name == constants.PodSchedulingGateName {
			continue
		}
		filterdGates = append(filterdGates, gate)
	}
	pod.Spec.SchedulingGates = filterdGates
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// schedulingGateがついているかでfiltering
	// reconcile頻度を減らす

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(r)
}
