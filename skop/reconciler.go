package skop

import (
	"context"

	"github.com/ericchiang/k8s"
)

type Reconciler interface {
	Reconcile(ctx context.Context, op *Operator, res k8s.Resource) error
}

type ReconcilerFunc func(ctx context.Context, op *Operator, res k8s.Resource) error

func (f ReconcilerFunc) Reconcile(ctx context.Context, op *Operator, res k8s.Resource) error {
	return f(ctx, op, res)
}
