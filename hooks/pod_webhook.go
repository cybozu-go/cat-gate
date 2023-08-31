package hooks

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const podSchedulingGateName = "cat-gate.cybozu.io/gate"
const catGateImagesHashAnnotation = "cat-gate.cybozu.io/images-hash"

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

	pod.Spec.SchedulingGates = append(pod.Spec.SchedulingGates, corev1.PodSchedulingGate{Name: podSchedulingGateName})
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	// コンテナ名一覧をスライスに入れたい
	imageSet := make(map[string]struct{})
	for _, c := range pod.Spec.InitContainers {
		imageSet[c.Image] = struct{}{}
	}
	for _, c := range pod.Spec.Containers {
		imageSet[c.Image] = struct{}{}
	}
	images := make([]string, 0)
	for k := range imageSet {
		images = append(images, k)
	}
	sort.Strings(images)
	imagesByte := sha256.Sum256([]byte(strings.Join(images, ",")))
	imagesHash := string(imagesByte[:])

	pod.Annotations[catGateImagesHashAnnotation] = imagesHash

	return nil
}
