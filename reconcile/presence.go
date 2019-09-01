package reconcile

import (
	"context"
	"net/http"

	"github.com/ericchiang/k8s"

	"github.com/thcyron/skop/skop"
)

func Presence(ctx context.Context, client skop.Client, res k8s.Resource) error {
	err := client.Create(ctx, res)
	if err != nil {
		if apiErr, ok := err.(*k8s.APIError); ok && apiErr.Code == http.StatusConflict {
			return nil
		}
		return err
	}
	return nil
}
