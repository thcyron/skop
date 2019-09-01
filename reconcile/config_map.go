package reconcile

import (
	"context"
	"net/http"

	"github.com/ericchiang/k8s"
	corev1 "github.com/ericchiang/k8s/apis/core/v1"

	"github.com/thcyron/skop/skop"
)

func ConfigMap(ctx context.Context, client skop.Client, configMap *corev1.ConfigMap) error {
	existing := new(corev1.ConfigMap)
	err := client.Get(
		ctx,
		configMap.GetMetadata().GetNamespace(),
		configMap.GetMetadata().GetName(),
		existing,
	)
	if err != nil {
		if apiErr, ok := err.(*k8s.APIError); ok {
			if apiErr.Code == http.StatusNotFound {
				return client.Create(ctx, configMap)
			}
		}
		return err
	}
	existing.Data = configMap.Data
	existing.BinaryData = configMap.BinaryData
	return client.Update(ctx, existing)
}
