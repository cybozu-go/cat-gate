package hooks

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const podSchedulingGateName = "cat-gate.cybozu.io/gate"
const catGateGroupAnnotation = "cat-gate.cybozu.io/group"

// log is for logging in this package.
// var podLogger = logf.Log.WithName("pod-defaulter")

func SetupPodWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&corev1.Pod{}).
		WithDefaulter(&PodDefaulter{}).
		Complete()
}

//+kubebuilder:webhook:path=/mutate--v1-pod,mutating=true,failurePolicy=fail,sideEffects=None,groups=core,resources=pods,verbs=create,versions=v1,name=mpod.kb.io,admissionReviewVersions=v1

type PodDefaulter struct{}

var _ admission.CustomDefaulter = &PodDefaulter{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (*PodDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return fmt.Errorf("unknown newObj type %T", obj)
	}

	var owner string
	for _, or := range pod.OwnerReferences {
		if or.Controller != nil && *or.Controller {
			owner = string(or.UID)
		}
	}
	if owner == "" {
		return nil
	}

	pod.Spec.SchedulingGates = append(pod.Spec.SchedulingGates, corev1.PodSchedulingGate{Name: podSchedulingGateName})
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}
	pod.Annotations[catGateGroupAnnotation] = owner

	return nil
}
