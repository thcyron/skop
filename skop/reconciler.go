package skop

import "context"

type Reconciler interface {
	Reconcile(ctx context.Context, op *Operator, res Resource) error
}

type ReconcilerFunc func(ctx context.Context, op *Operator, res Resource) error

func (f ReconcilerFunc) Reconcile(ctx context.Context, op *Operator, res Resource) error {
	return f(ctx, op, res)
}
