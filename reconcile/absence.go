package reconcile

import (
	"context"
	"net/http"

	"github.com/ericchiang/k8s"

	"github.com/thcyron/skop/skop"
)

func Absence(ctx context.Context, client skop.Client, res k8s.Resource) error {
	err := client.Delete(ctx, res)
	if err == nil {
		return nil
	}
	if apiErr, ok := err.(*k8s.APIError); ok && apiErr.Code == http.StatusNotFound {
		return nil
	}
	return err
}
