package reconcile

import (
	"context"
	"net/http"

	"github.com/ericchiang/k8s"
	appsv1 "github.com/ericchiang/k8s/apis/apps/v1"

	"github.com/thcyron/skop/skop"
)

func Deployment(ctx context.Context, client skop.Client, expected *appsv1.Deployment) error {
	existing := new(appsv1.Deployment)
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
	existing.Metadata.Labels = expected.Metadata.Labels
	existing.Metadata.Annotations = expected.Metadata.Annotations
	existing.Spec = expected.Spec
	return client.Update(ctx, existing)
}
