package indexing

import (
	"context"

	"github.com/cybozu-go/cat-gate/internal/constants"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func SetupIndexForPod(ctx context.Context, mgr manager.Manager) error {
	return mgr.GetFieldIndexer().IndexField(ctx, &corev1.Pod{}, constants.ImageHashAnnotationField, func(rawObj client.Object) []string {
		val := rawObj.GetAnnotations()[constants.CatGateImagesHashAnnotation]
		if val == "" {
			return nil
		}
		return []string{val}
	})
}
