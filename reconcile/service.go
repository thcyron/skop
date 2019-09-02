package reconcile

import (
	"context"
	"net/http"

	"github.com/ericchiang/k8s"
	corev1 "github.com/ericchiang/k8s/apis/core/v1"

	"github.com/thcyron/skop/skop"
)

func Service(ctx context.Context, client skop.Client, expected *corev1.Service) error {
	existing := new(corev1.Service)
	err := client.Get(
		ctx,
		expected.GetMetadata().GetName(),
		existing,
	)
	if err != nil {
		if apiErr, ok := err.(*k8s.APIError); ok {
			if apiErr.Code == http.StatusNotFound {
				return client.Create(ctx, expected)
			}
		}
		return err
	}
	clusterIP := existing.Spec.ClusterIP
	existing.Spec = expected.Spec
	existing.Spec.ClusterIP = clusterIP
	return client.Update(ctx, existing)
}
