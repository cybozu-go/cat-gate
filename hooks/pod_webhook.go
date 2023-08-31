package hooks

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
// var podLogger = logf.Log.WithName("pod-defaulter")

func SetupPodWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&corev1.Pod{}).
		WithDefaulter(&PodDefaulter{}).
		Complete()
}

//+kubebuilder:webhook:path=/mutate--v1-pod,mutating=true,failurePolicy=fail,sideEffects=None,groups=core,resources=pods,verbs=create;update,versions=v1,name=mpod.kb.io,admissionReviewVersions=v1

type PodDefaulter struct{}

var _ admission.CustomDefaulter = &PodDefaulter{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (*PodDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	_, ok := obj.(*corev1.Pod)
	if !ok {
		return fmt.Errorf("unknown newObj type %T", obj)
	}
	return nil
}
